// Package importer turns a recipe page or pasted text into a draft recipe. Nothing here
// saves anything: the person reviews the result in the editor first.
package importer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"
)

var (
	ErrBadURL      = errors.New("not a usable web address")
	ErrPrivate     = errors.New("address is on a private network")
	ErrBlocked     = errors.New("the site refused the request")
	ErrUnreachable = errors.New("the site could not be reached")
	ErrNotPage     = errors.New("the address is not a web page")
	ErrNoRecipe    = errors.New("no recipe found")
)

const (
	maxPageBytes  = 10 << 20
	maxImageBytes = 10 << 20
	maxRedirects  = 5
)

// cgnat and nat64 are not covered by netip's own checks, and can both reach internal hosts.
var (
	cgnat = netip.MustParsePrefix("100.64.0.0/10")
	nat64 = netip.MustParsePrefix("64:ff9b::/96")
)

func publicIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	return ip.IsValid() && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() &&
		!ip.IsInterfaceLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified() && !cgnat.Contains(ip) && !nat64.Contains(ip)
}

// Fetcher downloads pages on a person's behalf. Because the server makes the request, a
// link must never be able to reach the server's own network. The address is checked when
// the connection opens, after DNS, so a hostname or redirect can't get around it.
type Fetcher struct {
	client       *http.Client
	userAgent    string
	allowPrivate bool
}

// NewFetcher returns a Fetcher. allowPrivate exists for tests against a local server.
func NewFetcher(version string, allowPrivate bool) *Fetcher {
	f := &Fetcher{userAgent: "Wokko/" + version + " (self-hosted recipe manager; imports one page when a user asks)", allowPrivate: allowPrivate}
	dialer := &net.Dialer{Timeout: 8 * time.Second, Control: f.checkDial}
	f.client = &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			Proxy:               nil, // A proxy would make the connection check meaningless.
			DialContext:         dialer.DialContext,
			TLSHandshakeTimeout: 8 * time.Second,
			ForceAttemptHTTP2:   true,
		},
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return ErrUnreachable
			}
			return f.checkURL(r.URL)
		},
	}
	return f
}

func (f *Fetcher) checkDial(_, address string, _ syscall.RawConn) error {
	if f.allowPrivate {
		return nil
	}
	ap, err := netip.ParseAddrPort(address)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPrivate, err)
	}
	if !publicIP(ap.Addr()) {
		return ErrPrivate
	}
	return nil
}

func (f *Fetcher) checkURL(u *url.URL) error {
	if (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return ErrBadURL
	}
	if p := u.Port(); p != "" && p != "80" && p != "443" && !f.allowPrivate {
		return ErrBadURL
	}
	return nil
}

func (f *Fetcher) get(ctx context.Context, raw, accept string, limit int64) (body []byte, contentType, final string, err error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, "", "", ErrBadURL
	}
	if err := f.checkURL(u); err != nil {
		return nil, "", "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", "", ErrBadURL
	}
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", accept)
	res, err := f.client.Do(req)
	if err != nil {
		for _, known := range []error{ErrPrivate, ErrBadURL} {
			if errors.Is(err, known) {
				return nil, "", "", known
			}
		}
		return nil, "", "", fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == 401, res.StatusCode == 403, res.StatusCode == 429, res.StatusCode == 451, res.StatusCode == 503:
		return nil, "", "", ErrBlocked
	case res.StatusCode < 200 || res.StatusCode > 299:
		return nil, "", "", fmt.Errorf("%w: status %d", ErrUnreachable, res.StatusCode)
	}
	body, err = io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil {
		return nil, "", "", fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	if int64(len(body)) > limit {
		return nil, "", "", ErrNotPage
	}
	return body, res.Header.Get("Content-Type"), res.Request.URL.String(), nil
}

// Page downloads an HTML page and returns it with its content type and final address.
func (f *Fetcher) Page(ctx context.Context, raw string) ([]byte, string, string, error) {
	body, ct, final, err := f.get(ctx, raw, "text/html,application/xhtml+xml;q=0.9,*/*;q=0.1", maxPageBytes)
	if err != nil {
		return nil, "", "", err
	}
	if m := strings.ToLower(ct); m != "" && !strings.Contains(m, "html") && !strings.Contains(m, "xml") && !strings.Contains(m, "text/plain") {
		return nil, "", "", ErrNotPage
	}
	return body, ct, final, nil
}

// Image downloads a photo. The caller's normal upload path decides if it's a real image.
func (f *Fetcher) Image(ctx context.Context, raw string) ([]byte, string, error) {
	body, ct, _, err := f.get(ctx, raw, "image/jpeg,image/png,image/webp;q=0.9", maxImageBytes)
	if err != nil {
		return nil, "", err
	}
	m := strings.ToLower(strings.TrimSpace(strings.SplitN(ct, ";", 2)[0]))
	switch m {
	case "image/jpeg", "image/png", "image/webp":
		return body, m, nil
	}
	if s := http.DetectContentType(body); s == "image/jpeg" || s == "image/png" || s == "image/webp" {
		return body, s, nil
	}
	return nil, "", ErrNotPage
}
