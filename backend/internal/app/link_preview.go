package app

import (
	"context"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/serraniel/wokko/backend/internal/store/sqlcgen"
)

// Preview is what a link preview shows for one page. Image is a path on this server.
type Preview struct {
	SiteName    string
	Title       string
	Description string
	Image       string
	Article     bool
	NoIndex     bool
}

func (s *Service) settingString(ctx context.Context, key, fallback string) (string, error) {
	v, err := s.Store.Queries.GetSetting(ctx, key)
	if isNoRows(err) {
		return fallback, nil
	}
	return v, err
}

// GetLinkPreviewSettings returns how link previews look. Anyone may read it: it's shown to
// anybody who gets a link anyway.
func (s *Service) GetLinkPreviewSettings(ctx context.Context) (LinkPreviewSettings, error) {
	var out LinkPreviewSettings
	var err error
	if out.SiteName, err = s.settingString(ctx, "preview_site_name", ""); err != nil {
		return out, err
	}
	if out.Description, err = s.settingString(ctx, "preview_description", ""); err != nil {
		return out, err
	}
	recipes, err := s.settingString(ctx, "preview_recipes", "true")
	if err != nil {
		return out, err
	}
	out.ShowRecipes = recipes == "true"
	image, err := s.settingString(ctx, "preview_image", "")
	if err != nil {
		return out, err
	}
	out.HasImage = image != ""
	return out, nil
}

// AdminLinkPreviewSettings is the settings for the admin page.
func (s *Service) AdminLinkPreviewSettings(ctx context.Context, v Viewer) (LinkPreviewSettings, error) {
	if !v.isAdmin() {
		return LinkPreviewSettings{}, ErrForbidden
	}
	return s.GetLinkPreviewSettings(ctx)
}

func (s *Service) UpdateLinkPreviewSettings(ctx context.Context, v Viewer, in LinkPreviewSettings) (LinkPreviewSettings, error) {
	if !v.isAdmin() {
		return LinkPreviewSettings{}, ErrForbidden
	}
	in.SiteName = strings.TrimSpace(in.SiteName)
	in.Description = strings.TrimSpace(in.Description)
	if utf8.RuneCountInString(in.SiteName) > 60 || utf8.RuneCountInString(in.Description) > 300 {
		return LinkPreviewSettings{}, invalid("too_long", "The name can be 60 characters long and the description 300")
	}
	err := s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		for key, value := range map[string]string{"preview_site_name": in.SiteName, "preview_description": in.Description, "preview_recipes": strconv.FormatBool(in.ShowRecipes)} {
			if err := q.SetSetting(ctx, sqlcgen.SetSettingParams{Key: key, Value: value}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return LinkPreviewSettings{}, err
	}
	return s.GetLinkPreviewSettings(ctx)
}

// SetLinkPreviewImage stores the image shown for pages without a recipe of their own. It's
// re-encoded like any photo. With no image, Wokko's own is used.
func (s *Service) SetLinkPreviewImage(ctx context.Context, v Viewer, r io.Reader) (LinkPreviewSettings, error) {
	if !v.isAdmin() {
		return LinkPreviewSettings{}, ErrForbidden
	}
	var name string
	if r != nil {
		saved, err := s.Media.Save(r)
		if err != nil {
			return LinkPreviewSettings{}, invalid("invalid_image", err.Error())
		}
		name = saved
	}
	old, err := s.settingString(ctx, "preview_image", "")
	if err != nil {
		return LinkPreviewSettings{}, err
	}
	err = s.Store.Tx(ctx, func(q *sqlcgen.Queries) error {
		return q.SetSetting(ctx, sqlcgen.SetSettingParams{Key: "preview_image", Value: name})
	})
	if err != nil {
		if name != "" {
			s.removePhotos([]string{name})
		}
		return LinkPreviewSettings{}, err
	}
	if old != "" {
		s.removePhotos([]string{old})
	}
	return s.GetLinkPreviewSettings(ctx)
}

// LinkPreviewImage is the file of the admin's preview image, if one is set.
func (s *Service) LinkPreviewImage(ctx context.Context) (string, error) {
	name, err := s.settingString(ctx, "preview_image", "")
	if err != nil || name == "" {
		return "", orNotFound(err)
	}
	path, ok := s.Media.Path(name)
	if !ok {
		return "", ErrNotFound
	}
	return path, nil
}

// LinkPreview decides the preview of a page. Only pages anyone may open anyway show a recipe,
// checked exactly as opening them is. Everything else, including a private recipe's address, an
// expired link or a wrong token, gets the generic preview, which says nothing about what exists.
func (s *Service) LinkPreview(ctx context.Context, path, locale string) Preview {
	set, err := s.GetLinkPreviewSettings(ctx)
	if err != nil {
		s.Log.Warn("could not read the link preview settings", "error", err)
	}
	name := set.SiteName
	if name == "" {
		name = "Wokko"
	}
	generic := Preview{SiteName: name, Title: name, Description: set.Description, Image: "/og-image.png"}
	if set.HasImage {
		generic.Image = "/api/v1/public/link-preview-image"
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	noIndex := parts[0] == "s" || parts[0] == "shared" || parts[0] == "upload"
	generic.NoIndex = noIndex
	if !set.ShowRecipes {
		return generic
	}
	var r Recipe
	switch {
	case len(parts) == 2 && parts[0] == "s":
		r, err = s.GetSharedRecipe(ctx, locale, parts[1])
	case len(parts) == 3 && parts[0] == "shared":
		var id int64
		if id, err = strconv.ParseInt(parts[2], 10, 64); err == nil {
			r, err = s.GetCollectionRecipe(ctx, locale, parts[1], id)
		}
	case len(parts) == 2 && parts[0] == "gallery":
		var id int64
		if id, err = strconv.ParseInt(parts[1], 10, 64); err == nil {
			r, err = s.GetRecipe(ctx, Viewer{Locale: locale}, id)
		}
	default:
		return generic
	}
	if err != nil {
		return generic
	}
	out := Preview{SiteName: name, Title: r.Name, Description: recipeSummary(r), Article: true, NoIndex: noIndex, Image: generic.Image}
	if r.PhotoURL != "" {
		out.Image = r.PhotoURL
	}
	return out
}

// recipeSummary is a recipe's description, or else its ingredients, cut to about 200 characters.
func recipeSummary(r Recipe) string {
	text := strings.Join(strings.Fields(r.Description), " ")
	if text == "" {
		var names []string
		seen := map[string]bool{}
		for _, st := range r.Steps {
			for _, ing := range st.Ingredients {
				if n := ing.Food.Name; n != "" && !seen[n] {
					seen[n] = true
					names = append(names, n)
				}
			}
		}
		text = strings.Join(names, ", ")
	}
	if utf8.RuneCountInString(text) <= 200 {
		return text
	}
	cut := []rune(text)[:200]
	if i := strings.LastIndex(string(cut), " "); i > 100 {
		return string(cut)[:i] + "…"
	}
	return string(cut) + "…"
}
