package api

import (
	"net/http"

	"github.com/serraniel/wokko/backend/internal/app"
)

func (s *Server) listBooks(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	out, err := s.svc.ListBooks(r.Context(), v)
	return reply(w, out, err)
}

func (s *Server) createBook(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in app.RecipeBookInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	out, err := s.svc.CreateBook(r.Context(), v, in)
	return replyCreated(w, out, err)
}

func (s *Server) getBook(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	out, err := s.svc.GetBook(r.Context(), v, id)
	return reply(w, out, err)
}

func (s *Server) updateBook(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in app.RecipeBookInput
	if err := readJSON(r, &in); err != nil {
		return err
	}
	out, err := s.svc.UpdateBook(r.Context(), v, id, in)
	return reply(w, out, err)
}

func (s *Server) deleteBook(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return s.withID(w, r, func(id int64) error { return s.svc.DeleteBook(r.Context(), v, id) })
}

func (s *Server) addToBook(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return s.withID(w, r, func(id int64) error {
		rid, err := pathID(r, "rid")
		if err != nil {
			return err
		}
		_, err = s.svc.AddToBook(r.Context(), v, id, rid)
		return err
	})
}

func (s *Server) removeFromBook(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return s.withID(w, r, func(id int64) error {
		rid, err := pathID(r, "rid")
		if err != nil {
			return err
		}
		return s.svc.RemoveFromBook(r.Context(), v, id, rid)
	})
}

func (s *Server) recipeBooks(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if _, err := s.svc.GetRecipe(r.Context(), v, id); err != nil {
		return err
	}
	out, err := s.svc.BooksContaining(r.Context(), v, id)
	return reply(w, out, err)
}

func (s *Server) listCollectionLinks(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	out, err := s.svc.ListCollectionLinks(r.Context(), v)
	return reply(w, out, err)
}

func (s *Server) createCollectionLink(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	var in struct {
		ExpiresInDays int64 `json:"expires_in_days"`
	}
	if err := readJSON(r, &in); err != nil {
		return err
	}
	out, err := s.svc.CreateCollectionLink(r.Context(), v, in.ExpiresInDays)
	return replyCreated(w, out, err)
}

func (s *Server) revokeCollectionLink(w http.ResponseWriter, r *http.Request, v app.Viewer) error {
	return s.withID(w, r, func(id int64) error { return s.svc.RevokeCollectionLink(r.Context(), v, id) })
}

func (s *Server) listCollection(w http.ResponseWriter, r *http.Request) error {
	out, err := s.svc.ListCollection(r.Context(), s.anonLocale(r), r.PathValue("token"), r.URL.Query().Get("q"), queryInt(r, "limit"), queryInt(r, "offset"))
	return reply(w, out, err)
}

func (s *Server) getCollectionRecipe(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	out, err := s.svc.GetCollectionRecipe(r.Context(), s.anonLocale(r), r.PathValue("token"), id)
	return reply(w, out, err)
}
