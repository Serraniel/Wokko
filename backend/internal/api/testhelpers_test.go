package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/serraniel/wokko/backend/internal/api"
	"github.com/serraniel/wokko/backend/internal/app"
	"github.com/serraniel/wokko/backend/internal/config"
	"github.com/serraniel/wokko/backend/internal/store"
)

type env struct {
	t   *testing.T
	srv *httptest.Server
	svc *app.Service
	cfg config.Config
}

func newEnv(t *testing.T) *env {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Config{DataDir: filepath.Join(dir, "data"), BackupDir: filepath.Join(dir, "backups"), BackupInterval: 24 * time.Hour, BackupRetentionDays: 30, TrashRetentionDays: 30, Version: "test", PublicURL: "http://wokko.test"}
	mustDir(t, cfg.UploadsDir())
	st, err := store.Open(cfg.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := app.New(st, cfg, log)
	srv := httptest.NewServer(api.New(svc, log).Handler())
	t.Cleanup(srv.Close)
	return &env{t, srv, svc, cfg}
}

// client is one browser, with its own cookies.
type client struct {
	e    *env
	http *http.Client
}

func (e *env) anon() *client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		e.t.Fatal(err)
	}
	return &client{e, &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

type result struct {
	status int
	body   []byte
	header http.Header
}

func (r result) decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.body, v); err != nil {
		t.Fatalf("decode %q: %v", r.body, err)
	}
}

func (r result) code(t *testing.T) string {
	t.Helper()
	var e struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	r.decode(t, &e)
	return e.Error.Code
}

func (c *client) do(method, path string, body any) result {
	c.e.t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.e.t.Fatal(err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.e.srv.URL+path, rd)
	if err != nil {
		c.e.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.send(req)
}

func (c *client) send(req *http.Request) result {
	c.e.t.Helper()
	res, err := c.http.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		c.e.t.Fatal(err)
	}
	return result{res.StatusCode, b, res.Header}
}

func (c *client) ok(method, path string, body any, want int) result {
	c.e.t.Helper()
	r := c.do(method, path, body)
	if r.status != want {
		c.e.t.Fatalf("%s %s: status %d, want %d: %s", method, path, r.status, want, r.body)
	}
	return r
}

func (c *client) wantCode(method, path string, body any, status int, code string) {
	c.e.t.Helper()
	r := c.do(method, path, body)
	if r.status != status {
		c.e.t.Fatalf("%s %s: status %d, want %d: %s", method, path, r.status, status, r.body)
	}
	if code != "" && r.code(c.e.t) != code {
		c.e.t.Fatalf("%s %s: code %q, want %q", method, path, r.code(c.e.t), code)
	}
}

const v1 = "/api/v1"

func (e *env) admin() *client {
	e.t.Helper()
	c := e.anon()
	c.ok("POST", v1+"/setup", map[string]string{"handle": "admin", "password": "password123", "locale": "en"}, 201)
	return c
}

// newUser registers a user through a fresh invite from the admin.
func (e *env) newUser(admin *client, handle string) *client {
	e.t.Helper()
	var inv app.Invite
	admin.ok("POST", v1+"/admin/invites", map[string]any{"max_uses": 1}, 201).decode(e.t, &inv)
	c := e.anon()
	c.ok("POST", v1+"/auth/register", map[string]string{"handle": handle, "password": "password123", "invite": inv.Token}, 201)
	return c
}

func (c *client) createRecipe(in app.RecipeInput) app.Recipe {
	c.e.t.Helper()
	var r app.Recipe
	c.ok("POST", v1+"/recipes", in, 201).decode(c.e.t, &r)
	return r
}

func path(format string, args ...any) string { return v1 + fmt.Sprintf(format, args...) }

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	img.Set(1, 1, color.RGBA{200, 50, 50, 255})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func (c *client) upload(recipeID int64, content []byte) result {
	c.e.t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	f, err := w.CreateFormFile("photo", "photo.png")
	if err != nil {
		c.e.t.Fatal(err)
	}
	if _, err := f.Write(content); err != nil {
		c.e.t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		c.e.t.Fatal(err)
	}
	req, err := http.NewRequest("POST", c.e.srv.URL+path("/recipes/%d/photo", recipeID), &b)
	if err != nil {
		c.e.t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	return c.send(req)
}

func ptr[T any](v T) *T { return &v }

func simpleRecipe(name string) app.RecipeInput {
	return app.RecipeInput{
		Name: name, Servings: 4, Visibility: "private", Keywords: []string{"Quick"},
		Steps: []app.StepInput{
			{Instruction: "Chop", Minutes: 5, Ingredients: []app.IngredientInput{{FoodName: "onion", UnitName: "piece", Amount: ptr(2.0)}}},
			{Instruction: "Fry", Ingredients: []app.IngredientInput{{FoodName: "brand new food", Note: "fine"}}},
		},
	}
}
