package api_test

import (
	"context"
	"strings"
	"testing"

	"github.com/serraniel/wokko/backend/internal/app"
)

// TestLinkPreviewsShowOnlyWhatAnyoneMayOpen: a share link previews its recipe, while a private
// recipe's address, a wrong token or a paused link only get the generic card.
func TestLinkPreviewsShowOnlyWhatAnyoneMayOpen(t *testing.T) {
	e := newEnv(t)
	admin := e.admin()
	ctx := context.Background()
	in := simpleRecipe("Secret soup")
	in.Description = "A warm soup for cold days."
	r := admin.createRecipe(in)
	if res := admin.upload(r.ID, testPNG(t)); res.status != 200 {
		t.Fatalf("upload: %s", res.body)
	}
	var link app.ShareLink
	admin.ok("POST", path("/recipes/%d/sharing/links", r.ID), map[string]any{}, 201).decode(t, &link)
	token := link.URL[strings.LastIndex(link.URL, "/")+1:]

	p := e.svc.LinkPreview(ctx, "/s/"+token, "en")
	if p.Title != "Secret soup" || p.Description != "A warm soup for cold days." || !strings.Contains(p.Image, "?share="+token) || !p.Article || !p.NoIndex {
		t.Fatalf("share link: %+v", p)
	}
	for _, page := range []string{path("/recipes/%d", r.ID), path("/gallery/%d", r.ID), "/s/wrong", "/shared/wrong/1", "/login", "/"} {
		if got := e.svc.LinkPreview(ctx, strings.TrimPrefix(page, v1), "en"); got.Title != "Wokko" || strings.Contains(got.Description, "soup") || got.Article {
			t.Errorf("%s: %+v", page, got)
		}
	}

	// The admin's own name, text and image, and recipes kept out of previews.
	admin.ok("PUT", v1+"/admin/link-preview", app.LinkPreviewSettings{SiteName: "Our kitchen", Description: "Family recipes", ShowRecipes: false}, 200)
	var set app.LinkPreviewSettings
	if res := admin.uploadTo(v1+"/admin/link-preview/image", testPNG(t)); res.status != 200 {
		t.Fatalf("image: %d %s", res.status, res.body)
	} else {
		res.decode(t, &set)
	}
	if !set.HasImage || set.SiteName != "Our kitchen" {
		t.Fatalf("settings: %+v", set)
	}
	p = e.svc.LinkPreview(ctx, "/s/"+token, "en")
	if p.Title != "Our kitchen" || p.Description != "Family recipes" || p.Image != "/api/v1/public/link-preview-image" || !p.NoIndex {
		t.Fatalf("recipes hidden: %+v", p)
	}
	if res := e.anon().do("GET", v1+"/public/link-preview-image", nil); res.status != 200 || res.header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("image served: %d", res.status)
	}
	admin.ok("DELETE", v1+"/admin/link-preview/image", nil, 200)
	e.anon().ok("GET", v1+"/public/link-preview-image", nil, 404)

	// A paused link (public recipes and links off) previews nothing of the recipe either.
	admin.ok("PUT", v1+"/admin/link-preview", app.LinkPreviewSettings{ShowRecipes: true}, 200)
	admin.ok("PUT", v1+"/admin/settings", app.Settings{AllowPublic: false}, 200)
	if p := e.svc.LinkPreview(ctx, "/s/"+token, "en"); p.Title != "Wokko" {
		t.Fatalf("paused link: %+v", p)
	}

	// Only admins change it.
	other := e.newUser(admin, "other")
	other.wantCode("PUT", v1+"/admin/link-preview", app.LinkPreviewSettings{SiteName: "x"}, 403, "forbidden")
	other.wantCode("GET", v1+"/admin/link-preview", nil, 403, "forbidden")
}
