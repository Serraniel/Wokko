// Package web serves the compiled frontend from inside the binary.
package web

import (
	"bytes"
	"compress/gzip"
	"embed"
	"html"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"slices"
	"strings"
	"sync"
)

//go:embed all:dist
var dist embed.FS

// Meta is what a link preview, such as Discord's or a messenger's, shows for a page. The app
// renders in the browser, and preview bots don't run it, so the server puts these into the page.
type Meta struct {
	Title       string
	Description string
	Image       string // an absolute address, or empty
	URL         string
	Type        string // website or article
	SiteName    string
	NoIndex     bool
}

// Handler serves static files and falls back to the app shell for client-side routes. meta
// decides the link preview of each page the shell answers for.
func Handler(meta func(*http.Request) Meta) http.Handler {
	root, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err) // the embed path is fixed at compile time
	}
	return handler(meta, root)
}

// withMeta puts a page's title and preview tags into the shell. Every value is escaped.
func withMeta(shell []byte, m Meta) []byte {
	e := html.EscapeString
	var b strings.Builder
	b.WriteString("<title>" + e(m.Title) + "</title>")
	tag := func(attr, key, value string) {
		if value != "" {
			b.WriteString(`<meta ` + attr + `="` + key + `" content="` + e(value) + `" />`)
		}
	}
	tag("name", "description", m.Description)
	tag("property", "og:site_name", m.SiteName)
	tag("property", "og:title", m.Title)
	tag("property", "og:description", m.Description)
	tag("property", "og:type", m.Type)
	tag("property", "og:url", m.URL)
	tag("property", "og:image", m.Image)
	card := "summary"
	if m.Image != "" {
		card = "summary_large_image"
	}
	tag("name", "twitter:card", card)
	if m.NoIndex {
		tag("name", "robots", "noindex")
	}
	head := []byte(b.String())
	if i := bytes.Index(shell, []byte("<title>Wokko</title>")); i >= 0 {
		return slices.Concat(shell[:i], head, shell[i+len("<title>Wokko</title>"):])
	}
	if i := bytes.Index(shell, []byte("</head>")); i >= 0 {
		return slices.Concat(shell[:i], head, shell[i:])
	}
	return shell
}

// worthCompressing lists text formats. Images and fonts are compressed already.
func worthCompressing(name string) bool {
	switch path.Ext(name) {
	case ".js", ".css", ".html", ".json", ".svg", ".webmanifest", ".txt":
		return true
	}
	return false
}

// Go only knows a few types itself, and the container has no list of its own. Without this,
// favicon.ico would go out as application/octet-stream, which icon fetchers may ignore.
func init() {
	if err := mime.AddExtensionType(".ico", "image/x-icon"); err != nil {
		panic(err)
	}
}

func handler(meta func(*http.Request) Meta, root fs.FS) http.Handler {
	files := http.FileServerFS(root)
	// The files never change while the program runs, so each is compressed once.
	var compressed sync.Map
	gzipped := func(name string) ([]byte, bool) {
		if v, ok := compressed.Load(name); ok {
			return v.([]byte), true
		}
		raw, err := fs.ReadFile(root, name)
		if err != nil {
			return nil, false
		}
		var buf bytes.Buffer
		zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
		if err != nil {
			return nil, false
		}
		if _, err := zw.Write(raw); err != nil {
			return nil, false
		}
		if err := zw.Close(); err != nil {
			return nil, false
		}
		compressed.Store(name, buf.Bytes())
		return buf.Bytes(), true
	}
	acceptsGzip := func(r *http.Request) bool {
		for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
			if name, q, _ := strings.Cut(strings.TrimSpace(part), ";"); strings.EqualFold(name, "gzip") && strings.TrimSpace(q) != "q=0" {
				return true
			}
		}
		return false
	}
	serve := func(w http.ResponseWriter, r *http.Request, name, contentType string) bool {
		if !worthCompressing(name) || !acceptsGzip(r) {
			return false
		}
		body, ok := gzipped(name)
		if !ok {
			return false
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		if r.Method != http.MethodHead {
			_, _ = w.Write(body)
		}
		return true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p != "" {
			if info, err := fs.Stat(root, p); err == nil && !info.IsDir() {
				if strings.HasPrefix(p, "_app/immutable/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					w.Header().Set("Cache-Control", "no-cache")
				}
				contentType := mime.TypeByExtension(path.Ext(p))
				if contentType == "" {
					contentType = "application/octet-stream"
				}
				if !serve(w, r, p, contentType) {
					files.ServeHTTP(w, r)
				}
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		if meta == nil && serve(w, r, "index.html", "text/html; charset=utf-8") {
			return
		}
		shell, err := fs.ReadFile(root, "index.html")
		if err != nil {
			http.Error(w, "The frontend was not included in this build.", http.StatusServiceUnavailable)
			return
		}
		if meta != nil {
			shell = withMeta(shell, meta(r))
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if acceptsGzip(r) {
			var buf bytes.Buffer
			zw := gzip.NewWriter(&buf)
			if _, err := zw.Write(shell); err == nil && zw.Close() == nil {
				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Add("Vary", "Accept-Encoding")
				shell = buf.Bytes()
			}
		}
		if r.Method != http.MethodHead {
			_, _ = w.Write(shell)
		}
	})
}
