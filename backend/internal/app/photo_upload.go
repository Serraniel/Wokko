package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"strings"
	"time"

	"github.com/serraniel/wokko/backend/internal/auth"
	"github.com/serraniel/wokko/backend/internal/media"
	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
)

// photoLinkLifetime is how long a photo upload link works. Short, because anyone holding it can
// set that one recipe's photo.
const photoLinkLifetime = 15 * time.Minute

// SetPhotoFromBase64 sets a recipe's photo from image data an assistant holds as a file, such
// as an upload in its sandbox. A data: URL prefix is accepted. The image goes through the same
// checks and re-encoding as any upload, so anything that isn't a real image is refused.
func (s *Service) SetPhotoFromBase64(ctx context.Context, v Viewer, id int64, data string) (Recipe, error) {
	data = strings.TrimSpace(data)
	if i := strings.Index(data, ","); strings.HasPrefix(data, "data:") && i > 0 {
		data = data[i+1:]
	}
	data = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, data)
	if base64.StdEncoding.DecodedLen(len(data)) > media.MaxUploadBytes {
		return Recipe{}, invalid("invalid_image", "The image is larger than 20 MB")
	}
	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(data, "="))
	}
	if err != nil || len(raw) == 0 {
		return Recipe{}, invalid("invalid_image", "This isn't valid base64 image data. If the photo is only visible to you and not a file you hold, use create_photo_upload_link instead.")
	}
	return s.SetPhoto(ctx, v, id, bytes.NewReader(raw))
}

// CreatePhotoUploadLink makes a link that lets whoever opens it set this recipe's photo once,
// within 15 minutes, as the person who created it. An assistant hands it to the person, who
// picks or takes the photo on their phone.
func (s *Service) CreatePhotoUploadLink(ctx context.Context, v Viewer, id int64, base string) (PhotoUploadLink, error) {
	if v.User == nil {
		return PhotoUploadLink{}, ErrUnauthorized
	}
	if err := s.checkEdit(ctx, v, id); err != nil {
		return PhotoUploadLink{}, err
	}
	token, err := auth.NewToken()
	if err != nil {
		return PhotoUploadLink{}, err
	}
	expires := s.stamp(s.Now().Add(photoLinkLifetime))
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		if err := q.DeleteOldPhotoUploadLinks(ctx, s.stamp(s.Now().Add(-24*time.Hour))); err != nil {
			return err
		}
		return q.CreatePhotoUploadLink(ctx, sqlcgen.CreatePhotoUploadLinkParams{TokenHash: auth.HashToken(token), RecipeID: id, UserID: v.id(), ExpiresAt: expires})
	})
	if err != nil {
		return PhotoUploadLink{}, err
	}
	return PhotoUploadLink{URL: strings.TrimRight(base, "/") + "/upload/" + token, ExpiresAt: expires}, nil
}

// photoUploadLink finds a link that still works, and the person it acts as. Anything else,
// whether unknown, used, expired, or the person can no longer edit the recipe, is not found.
func (s *Service) photoUploadLink(ctx context.Context, token string) (sqlcgen.PhotoUploadLink, Viewer, error) {
	l, err := s.Store.Queries.GetPhotoUploadLink(ctx, auth.HashToken(token))
	if isNoRows(err) {
		return l, Viewer{}, ErrNotFound
	}
	if err != nil {
		return l, Viewer{}, err
	}
	if l.UsedAt.Valid || l.ExpiresAt <= s.stamp(s.Now()) {
		return l, Viewer{}, ErrNotFound
	}
	u, err := s.Store.Queries.GetUser(ctx, l.UserID)
	if isNoRows(err) || (err == nil && u.DisabledAt.Valid) {
		return l, Viewer{}, ErrNotFound
	}
	if err != nil {
		return l, Viewer{}, err
	}
	v := Viewer{User: &u, Locale: u.Locale}
	if err := s.checkEdit(ctx, v, l.RecipeID); err != nil {
		return l, Viewer{}, ErrNotFound
	}
	return l, v, nil
}

func (s *Service) PhotoUploadInfo(ctx context.Context, token string) (PhotoUpload, error) {
	l, v, err := s.photoUploadLink(ctx, token)
	if err != nil {
		return PhotoUpload{}, err
	}
	r, err := s.GetRecipe(ctx, v, l.RecipeID)
	if err != nil {
		return PhotoUpload{}, err
	}
	return PhotoUpload{RecipeName: r.Name}, nil
}

// UsePhotoUploadLink sets the photo and uses the link up. The link is claimed first, so it
// can't be used twice at once, and freed again if the photo is refused, so a wrong file can be
// replaced by the right one.
func (s *Service) UsePhotoUploadLink(ctx context.Context, token string, r io.Reader) error {
	l, v, err := s.photoUploadLink(ctx, token)
	if err != nil {
		return err
	}
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		n, err := q.ClaimPhotoUploadLink(ctx, sqlcgen.ClaimPhotoUploadLinkParams{UsedAt: nullStr(s.stamp(s.Now())), ID: l.ID})
		if err == nil && n == 0 {
			return ErrNotFound
		}
		return err
	})
	if err != nil {
		return err
	}
	if _, err := s.SetPhoto(ctx, v, l.RecipeID, r); err != nil {
		release := func(q *sqlcgen.Queries) error { return q.ReleasePhotoUploadLink(ctx, l.ID) }
		if rerr := s.Store.Tx(ctx, release); rerr != nil {
			s.Log.Warn("could not free a photo upload link", "error", rerr)
		}
		return err
	}
	return nil
}
