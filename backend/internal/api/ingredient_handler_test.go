package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestIngredientHandlers(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testCfg(t)
	ts := newIntegrationServer(t, pool, cfg)

	_, userToken := createTestUser(t, pool, cfg, "user")
	_, adminToken := createTestUser(t, pool, cfg, "admin")

	authHeader := func(token string) http.Header {
		h := http.Header{}
		h.Set("Authorization", "Bearer "+token)
		return h
	}

	doJSON := func(method, url string, body interface{}, headers http.Header) *http.Response {
		t.Helper()
		var req *http.Request
		if body != nil {
			b, _ := json.Marshal(body)
			r, _ := http.NewRequest(method, url, bytes.NewReader(b))
			r.Header.Set("Content-Type", "application/json")
			req = r
		} else {
			r, _ := http.NewRequest(method, url, nil)
			req = r
		}
		for k, vs := range headers {
			for _, v := range vs {
				req.Header.Set(k, v)
			}
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		return resp
	}

	t.Run("POST /api/ingredients - creates ingredient, returns 201", func(t *testing.T) {
		body := map[string]interface{}{
			"name_en":   "Onion",
			"plural_en": "Onions",
			"name_de":   "Zwiebel",
		}
		resp := doJSON(http.MethodPost, ts.URL+"/api/ingredients", body, authHeader(userToken))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.StatusCode)
		}
		var ing map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&ing); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if ing["name_en"] != "Onion" {
			t.Errorf("expected name_en 'Onion', got %v", ing["name_en"])
		}
		if ing["id"] == "" {
			t.Error("expected non-empty id")
		}
	})

	t.Run("POST /api/ingredients - unauthenticated returns 401", func(t *testing.T) {
		body := map[string]interface{}{"name_en": "Garlic"}
		resp := doJSON(http.MethodPost, ts.URL+"/api/ingredients", body, nil)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	// Create a few ingredients for search testing
	for _, name := range []string{"Carrot", "Parsnip", "Broccoli"} {
		resp := doJSON(http.MethodPost, ts.URL+"/api/ingredients",
			map[string]interface{}{"name_en": name},
			authHeader(userToken),
		)
		resp.Body.Close()
	}

	t.Run("GET /api/ingredients?search=on - returns matching ingredients (fuzzy)", func(t *testing.T) {
		resp := doJSON(http.MethodGet, ts.URL+"/api/ingredients?search=on", nil, authHeader(userToken))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var list []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
			t.Fatalf("decode: %v", err)
		}
		// "Onion" should match "on"
		found := false
		for _, item := range list {
			name, _ := item["name_en"].(string)
			if strings.EqualFold(name, "onion") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected 'Onion' in search results for 'on', got %v", list)
		}
	})

	t.Run("GET /api/ingredients - lists all ingredients", func(t *testing.T) {
		resp := doJSON(http.MethodGet, ts.URL+"/api/ingredients", nil, authHeader(userToken))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var list []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(list) == 0 {
			t.Fatal("expected at least one ingredient")
		}
	})

	t.Run("Admin route - GET /api/admin/users requires admin role", func(t *testing.T) {
		// Regular user should get 403
		resp := doJSON(http.MethodGet, ts.URL+"/api/admin/users", nil, authHeader(userToken))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 for regular user, got %d", resp.StatusCode)
		}
	})

	t.Run("Admin route - GET /api/admin/users accessible with admin token", func(t *testing.T) {
		resp := doJSON(http.MethodGet, ts.URL+"/api/admin/users", nil, authHeader(adminToken))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for admin, got %d", resp.StatusCode)
		}
	})
}
