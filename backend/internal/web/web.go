// Package web serves the compiled frontend from inside the binary.
package web

import (
	"bytes"
	"compress/gzip"
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"
)

//go:embed all:dist
var dist embed.FS

// Handler serves static files and falls back to the app shell for client-side routes.
func Handler() http.Handler {
	root, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err) // the embed path is fixed at compile time
	}
	return handler(root)
}

// worthCompressing lists text formats. Images and fonts are compressed already.
func worthCompressing(name string) bool {
	switch path.Ext(name) {
	case ".js", ".css", ".html", ".json", ".svg", ".webmanifest", ".txt":
		return true
	}
	return false
}

func handler(root fs.FS) http.Handler {
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
		if serve(w, r, "index.html", "text/html; charset=utf-8") {
			return
		}
		shell, err := fs.ReadFile(root, "index.html")
		if err != nil {
			http.Error(w, "The frontend was not included in this build.", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(shell)
	})
}
