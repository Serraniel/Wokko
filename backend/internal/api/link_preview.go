package api

import (
	"net/http"
	"os"

	"github.com/serraniel/wokko/backend/internal/app"
	"github.com/serraniel/wokko/backend/internal/media"
	"github.com/serraniel/wokko/backend/internal/web"
)

// pageMeta gives each page its link preview. Image and page addresses need PUBLIC_URL: built
// from the request's Host header they could be pointed anywhere, so without it only text is sent.
func (s *Server) pageMeta(r *http.Request) web.Meta {
	p := s.svc.LinkPreview(r.Context(), r.URL.Path, s.anonLocale(r))
	m := web.Meta{SiteName: p.SiteName, Title: p.Title, Description: p.Description, Type: "website", NoIndex: p.NoIndex}
	if p.Article {
		m.Type = "article"
	}
	if base := s.svc.Cfg.PublicURL; base != "" {
		m.URL = base + r.URL.Path
		if p.Image != "" {
			m.Image = base + p.Image
		}
	}
	return m
}

func (s *Server) linkPreviewImage(w http.ResponseWriter, r *http.Request) error {
	path, err := s.svc.LinkPreviewImage(r.Context())
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return app.ErrNotFound
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeFile(w, r, path)
	return nil
}

func (s *Server) getLinkPreview(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	out, err := s.svc.AdminLinkPreviewSettings(r.Context(), v)
	return reply(w, out, err)
}

func (s *Server) updateLinkPreview(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return decodeAnd(w, r, 200, func(in app.LinkPreviewSettings) (app.LinkPreviewSettings, error) {
		return s.svc.UpdateLinkPreviewSettings(r.Context(), v, in)
	})
}

func (s *Server) uploadLinkPreviewImage(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	r.Body = http.MaxBytesReader(w, r.Body, media.MaxUploadBytes+1<<20)
	file, _, err := r.FormFile("photo")
	if err != nil {
		return &app.Error{Status: http.StatusBadRequest, Code: "invalid_image", Message: "Please choose a photo no larger than 20 MB"}
	}
	defer file.Close()
	out, err := s.svc.SetLinkPreviewImage(r.Context(), v, file)
	return reply(w, out, err)
}

func (s *Server) deleteLinkPreviewImage(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	out, err := s.svc.SetLinkPreviewImage(r.Context(), v, nil)
	return reply(w, out, err)
}
