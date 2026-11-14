package web

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func fetch(t *testing.T, h http.Handler, target, encoding string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, target, nil)
	if encoding != "" {
		r.Header.Set("Accept-Encoding", encoding)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestAssetsAreCompressedAndCached(t *testing.T) {
	js := strings.Repeat("console.log(1);", 200)
	h := handler(nil, fstest.MapFS{
		"index.html":                {Data: []byte("<html>shell</html>")},
		"_app/immutable/app.js":     {Data: []byte(js)},
		"_app/immutable/font.woff2": {Data: []byte("binary")},
	})

	w := fetch(t, h, "/_app/immutable/app.js", "br, gzip")
	if w.Header().Get("Content-Encoding") != "gzip" || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") || !strings.Contains(w.Header().Get("Content-Type"), "javascript") {
		t.Fatalf("headers: %v", w.Header())
	}
	zr, err := gzip.NewReader(bytes.NewReader(w.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := io.ReadAll(zr); string(got) != js {
		t.Fatal("the compressed body differs from the file")
	}
	if w.Body.Len() >= len(js) {
		t.Fatalf("not smaller: %d", w.Body.Len())
	}

	if w := fetch(t, h, "/_app/immutable/app.js", ""); w.Header().Get("Content-Encoding") != "" || w.Body.String() != js {
		t.Fatal("clients without gzip must get the plain file")
	}
	if w := fetch(t, h, "/_app/immutable/app.js", "gzip;q=0"); w.Header().Get("Content-Encoding") != "" {
		t.Fatal("q=0 means the client refuses gzip")
	}
	if w := fetch(t, h, "/_app/immutable/font.woff2", "gzip"); w.Header().Get("Content-Encoding") != "" {
		t.Fatal("fonts are compressed already")
	}
}

func TestUnknownPathsGetTheAppShell(t *testing.T) {
	h := handler(nil, fstest.MapFS{"index.html": {Data: []byte("<html>shell</html>")}})
	for _, enc := range []string{"", "gzip"} {
		w := fetch(t, h, "/recipes/12", enc)
		if w.Code != 200 || w.Header().Get("Cache-Control") != "no-cache" {
			t.Fatalf("%q: %d %v", enc, w.Code, w.Header())
		}
	}
	if w := fetch(t, h, "/recipes/12", ""); w.Body.String() != "<html>shell</html>" {
		t.Fatalf("body %q", w.Body.String())
	}
}

func TestFaviconIsAnImage(t *testing.T) {
	h := handler(nil, fstest.MapFS{"index.html": {Data: []byte("<html>shell</html>")}, "favicon.ico": {Data: []byte("\x00\x00\x01\x00")}})
	if w := fetch(t, h, "/favicon.ico", "gzip"); w.Code != 200 || w.Header().Get("Content-Type") != "image/x-icon" {
		t.Fatalf("%d %v", w.Code, w.Header())
	}
}

func TestLinkPreviewTagsAreEscaped(t *testing.T) {
	shell := []byte(`<html><head><title>Wokko</title></head><body></body></html>`)
	h := handler(func(*http.Request) Meta {
		return Meta{SiteName: "Wokko", Title: `Soup "<script>alert(1)</script>"`, Description: "Hot & sour", Image: "https://w.example/media/a.jpg", Type: "article", NoIndex: true}
	}, fstest.MapFS{"index.html": {Data: shell}})
	w := fetch(t, h, "/s/abc", "")
	body := w.Body.String()
	for _, want := range []string{
		`<title>Soup &#34;&lt;script&gt;alert(1)&lt;/script&gt;&#34;</title>`,
		`<meta property="og:description" content="Hot &amp; sour" />`,
		`<meta property="og:image" content="https://w.example/media/a.jpg" />`,
		`<meta name="twitter:card" content="summary_large_image" />`,
		`<meta name="robots" content="noindex" />`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s in %s", want, body)
		}
	}
	if strings.Contains(body, "<script>") || strings.Count(body, "<title>") != 1 {
		t.Fatalf("not escaped or doubled: %s", body)
	}
	if gz := fetch(t, h, "/s/abc", "gzip"); gz.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("the shell with tags should still be compressed")
	}
}
