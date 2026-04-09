package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/serraniel/wokko/backend/internal/config"
)

func newTestServer(t *testing.T) (*Server, *pgxpool.Pool) {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	cfg := &config.Config{
		JWTSecret:        "test-secret-key-for-testing",
		OpenRegistration: false,
	}
	s := NewServer(pool, cfg)
	return s, pool
}

func doRequest(t *testing.T, e *echo.Echo, method, path string, body interface{}, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func doRequestWithToken(t *testing.T, e *echo.Echo, method, path string, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func setupEcho(s *Server) *echo.Echo {
	e := echo.New()
	e.GET("/api/health", s.HealthCheck)
	authGroup := e.Group("/api/auth")
	authGroup.POST("/setup", s.SetupAdmin)
	authGroup.POST("/register", s.Register)
	authGroup.POST("/login", s.Login)
	authGroup.POST("/refresh", s.RefreshToken)
	authGroup.POST("/logout", s.Logout)
	authGroup.GET("/me", s.GetCurrentUser, s.AuthMiddleware)
	return e
}

func TestSetupAdmin(t *testing.T) {
	s, pool := newTestServer(t)
	defer pool.Close()
	e := setupEcho(s)

	// Clean slate
	_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE username = 'testadmin'")
	_, _ = pool.Exec(context.Background(), "UPDATE system_settings SET value = 'false' WHERE key = 'setup_complete'")
	_, _ = pool.Exec(context.Background(), "DELETE FROM system_settings WHERE key = 'setup_complete'")

	t.Run("success", func(t *testing.T) {
		rec := doRequest(t, e, http.MethodPost, "/api/auth/setup", map[string]string{
			"username": "testadmin",
			"password": "adminpassword123",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp tokenResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil || resp.AccessToken == "" {
			t.Fatal("expected access_token in response")
		}
	})

	t.Run("already set up returns 400", func(t *testing.T) {
		rec := doRequest(t, e, http.MethodPost, "/api/auth/setup", map[string]string{
			"username": "testadmin2",
			"password": "adminpassword123",
		})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// Cleanup
	_, _ = pool.Exec(context.Background(), "DELETE FROM refresh_tokens WHERE user_id = (SELECT id FROM users WHERE username = 'testadmin')")
	_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE username = 'testadmin'")
	_, _ = pool.Exec(context.Background(), "DELETE FROM system_settings WHERE key = 'setup_complete'")
}

func TestLogin(t *testing.T) {
	s, pool := newTestServer(t)
	defer pool.Close()
	e := setupEcho(s)

	// Ensure setup is done first
	_, _ = pool.Exec(context.Background(), "DELETE FROM system_settings WHERE key = 'setup_complete'")
	_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE username = 'loginuser'")
	setupRec := doRequest(t, e, http.MethodPost, "/api/auth/setup", map[string]string{
		"username": "loginuser",
		"password": "loginpassword123",
	})
	if setupRec.Code != http.StatusOK {
		t.Fatalf("setup failed: %d %s", setupRec.Code, setupRec.Body.String())
	}

	t.Run("success", func(t *testing.T) {
		rec := doRequest(t, e, http.MethodPost, "/api/auth/login", map[string]string{
			"username": "loginuser",
			"password": "loginpassword123",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("wrong password returns 401", func(t *testing.T) {
		rec := doRequest(t, e, http.MethodPost, "/api/auth/login", map[string]string{
			"username": "loginuser",
			"password": "wrongpassword",
		})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("unknown user returns 401", func(t *testing.T) {
		rec := doRequest(t, e, http.MethodPost, "/api/auth/login", map[string]string{
			"username": "nosuchuser",
			"password": "somepassword",
		})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	// Cleanup
	_, _ = pool.Exec(context.Background(), "DELETE FROM refresh_tokens WHERE user_id = (SELECT id FROM users WHERE username = 'loginuser')")
	_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE username = 'loginuser'")
	_, _ = pool.Exec(context.Background(), "DELETE FROM system_settings WHERE key = 'setup_complete'")
}

func TestRegisterWithoutInvite(t *testing.T) {
	s, pool := newTestServer(t)
	defer pool.Close()
	e := setupEcho(s)

	// Setup required first
	_, _ = pool.Exec(context.Background(), "DELETE FROM system_settings WHERE key = 'setup_complete'")
	_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE username = 'adminforregtest'")
	setupRec := doRequest(t, e, http.MethodPost, "/api/auth/setup", map[string]string{
		"username": "adminforregtest",
		"password": "adminpassword123",
	})
	if setupRec.Code != http.StatusOK {
		t.Fatalf("setup failed: %d", setupRec.Code)
	}

	t.Run("no invite on closed registration returns 403", func(t *testing.T) {
		rec := doRequest(t, e, http.MethodPost, "/api/auth/register", map[string]string{
			"username": "newuser",
			"password": "newpassword123",
		})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// Cleanup
	_, _ = pool.Exec(context.Background(), "DELETE FROM refresh_tokens WHERE user_id = (SELECT id FROM users WHERE username = 'adminforregtest')")
	_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE username = 'adminforregtest'")
	_, _ = pool.Exec(context.Background(), "DELETE FROM system_settings WHERE key = 'setup_complete'")
}

func TestRefreshToken(t *testing.T) {
	s, pool := newTestServer(t)
	defer pool.Close()
	e := setupEcho(s)

	// Setup
	_, _ = pool.Exec(context.Background(), "DELETE FROM system_settings WHERE key = 'setup_complete'")
	_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE username = 'refreshuser'")
	setupRec := doRequest(t, e, http.MethodPost, "/api/auth/setup", map[string]string{
		"username": "refreshuser",
		"password": "refreshpassword123",
	})
	if setupRec.Code != http.StatusOK {
		t.Fatalf("setup failed: %d", setupRec.Code)
	}

	var refreshCookie *http.Cookie
	for _, c := range setupRec.Result().Cookies() {
		if c.Name == "refresh_token" {
			refreshCookie = c
			break
		}
	}
	if refreshCookie == nil {
		t.Fatal("no refresh_token cookie set")
	}

	t.Run("valid cookie returns new token", func(t *testing.T) {
		rec := doRequest(t, e, http.MethodPost, "/api/auth/refresh", nil, refreshCookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp tokenResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil || resp.AccessToken == "" {
			t.Fatal("expected access_token in response")
		}
	})

	t.Run("invalid cookie returns 401", func(t *testing.T) {
		badCookie := &http.Cookie{Name: "refresh_token", Value: "invalid-token-value"}
		rec := doRequest(t, e, http.MethodPost, "/api/auth/refresh", nil, badCookie)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	// Cleanup
	_, _ = pool.Exec(context.Background(), "DELETE FROM refresh_tokens WHERE user_id = (SELECT id FROM users WHERE username = 'refreshuser')")
	_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE username = 'refreshuser'")
	_, _ = pool.Exec(context.Background(), "DELETE FROM system_settings WHERE key = 'setup_complete'")
}

func TestGetCurrentUser(t *testing.T) {
	s, pool := newTestServer(t)
	defer pool.Close()
	e := setupEcho(s)

	// Setup
	_, _ = pool.Exec(context.Background(), "DELETE FROM system_settings WHERE key = 'setup_complete'")
	_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE username = 'meuser'")
	setupRec := doRequest(t, e, http.MethodPost, "/api/auth/setup", map[string]string{
		"username": "meuser",
		"password": "mepassword123",
	})
	if setupRec.Code != http.StatusOK {
		t.Fatalf("setup failed: %d", setupRec.Code)
	}
	var tokenResp tokenResponse
	_ = json.NewDecoder(setupRec.Body).Decode(&tokenResp)

	t.Run("with valid token returns user", func(t *testing.T) {
		rec := doRequestWithToken(t, e, http.MethodGet, "/api/auth/me", tokenResp.AccessToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp userResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatal("failed to decode response")
		}
		if resp.Username != "meuser" {
			t.Fatalf("expected username 'meuser', got '%s'", resp.Username)
		}
	})

	t.Run("without token returns 401", func(t *testing.T) {
		rec := doRequest(t, e, http.MethodGet, "/api/auth/me", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	// Cleanup
	_, _ = pool.Exec(context.Background(), "DELETE FROM refresh_tokens WHERE user_id = (SELECT id FROM users WHERE username = 'meuser')")
	_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE username = 'meuser'")
	_, _ = pool.Exec(context.Background(), "DELETE FROM system_settings WHERE key = 'setup_complete'")
}
