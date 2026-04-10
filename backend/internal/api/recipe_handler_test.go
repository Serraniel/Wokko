package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestRecipeHandlers(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := testCfg(t)
	ts := newIntegrationServer(t, pool, cfg)

	userID, userToken := createTestUser(t, pool, cfg, "user")
	_, otherToken := createTestUser(t, pool, cfg, "user")

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

	t.Run("GET /api/recipes - empty list for new user", func(t *testing.T) {
		resp := doJSON(http.MethodGet, ts.URL+"/api/recipes", nil, authHeader(userToken))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var list []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(list) != 0 {
			t.Fatalf("expected empty list, got %d items", len(list))
		}
	})

	t.Run("GET /api/recipes - unauthenticated returns 401", func(t *testing.T) {
		resp := doJSON(http.MethodGet, ts.URL+"/api/recipes", nil, nil)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	var createdID string

	t.Run("POST /api/recipes - creates recipe, returns 201", func(t *testing.T) {
		body := map[string]interface{}{
			"name":        "Test Recipe",
			"description": "A test recipe description",
			"servings":    4,
			"is_public":   false,
		}
		resp := doJSON(http.MethodPost, ts.URL+"/api/recipes", body, authHeader(userToken))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.StatusCode)
		}
		var recipe map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&recipe); err != nil {
			t.Fatalf("decode: %v", err)
		}
		id, ok := recipe["id"].(string)
		if !ok || id == "" {
			t.Fatalf("expected id in response, got %v", recipe)
		}
		createdID = id

		// Verify author_id matches our user
		authorID, _ := recipe["author_id"].(string)
		if authorID != userID {
			t.Errorf("expected author_id %s, got %s", userID, authorID)
		}
	})

	t.Run("POST /api/recipes - unauthenticated returns 401", func(t *testing.T) {
		body := map[string]interface{}{"name": "Unauthenticated Recipe"}
		resp := doJSON(http.MethodPost, ts.URL+"/api/recipes", body, nil)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("GET /api/recipes/:id - returns created recipe", func(t *testing.T) {
		if createdID == "" {
			t.Skip("no recipe created")
		}
		resp := doJSON(http.MethodGet, ts.URL+"/api/recipes/"+createdID, nil, authHeader(userToken))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var recipe map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&recipe); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if recipe["name"] != "Test Recipe" {
			t.Errorf("expected name 'Test Recipe', got %v", recipe["name"])
		}
	})

	t.Run("GET /api/recipes/:id - other user cannot access private recipe", func(t *testing.T) {
		if createdID == "" {
			t.Skip("no recipe created")
		}
		resp := doJSON(http.MethodGet, ts.URL+"/api/recipes/"+createdID, nil, authHeader(otherToken))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.StatusCode)
		}
	})

	t.Run("PUT /api/recipes/:id - updates recipe", func(t *testing.T) {
		if createdID == "" {
			t.Skip("no recipe created")
		}
		newName := "Updated Test Recipe"
		body := map[string]interface{}{"name": newName}
		resp := doJSON(http.MethodPut, ts.URL+"/api/recipes/"+createdID, body, authHeader(userToken))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var recipe map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&recipe); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if recipe["name"] != newName {
			t.Errorf("expected name %q, got %v", newName, recipe["name"])
		}
	})

	t.Run("PUT /api/recipes/:id - other user cannot update", func(t *testing.T) {
		if createdID == "" {
			t.Skip("no recipe created")
		}
		body := map[string]interface{}{"name": "Hijacked Name"}
		resp := doJSON(http.MethodPut, ts.URL+"/api/recipes/"+createdID, body, authHeader(otherToken))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.StatusCode)
		}
	})

	t.Run("DELETE /api/recipes/:id - other user cannot delete", func(t *testing.T) {
		if createdID == "" {
			t.Skip("no recipe created")
		}
		resp := doJSON(http.MethodDelete, ts.URL+"/api/recipes/"+createdID, nil, authHeader(otherToken))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.StatusCode)
		}
	})

	t.Run("DELETE /api/recipes/:id - deletes recipe, subsequent GET returns 404", func(t *testing.T) {
		if createdID == "" {
			t.Skip("no recipe created")
		}
		// Delete
		resp := doJSON(http.MethodDelete, ts.URL+"/api/recipes/"+createdID, nil, authHeader(userToken))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", resp.StatusCode)
		}

		// Subsequent GET should return 404
		resp2 := doJSON(http.MethodGet, ts.URL+"/api/recipes/"+createdID, nil, authHeader(userToken))
		defer resp2.Body.Close()
		if resp2.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 after delete, got %d", resp2.StatusCode)
		}
	})

	t.Run("GET /api/recipes list only returns own recipes", func(t *testing.T) {
		// Create one recipe as user
		body := map[string]interface{}{"name": fmt.Sprintf("List Test Recipe")}
		resp := doJSON(http.MethodPost, ts.URL+"/api/recipes", body, authHeader(userToken))
		resp.Body.Close()

		// List for user should see it
		listResp := doJSON(http.MethodGet, ts.URL+"/api/recipes", nil, authHeader(userToken))
		defer listResp.Body.Close()
		var list []map[string]interface{}
		json.NewDecoder(listResp.Body).Decode(&list)
		if len(list) == 0 {
			t.Fatal("expected at least one recipe for user")
		}

		// List for other user should be empty (they have no recipes)
		listResp2 := doJSON(http.MethodGet, ts.URL+"/api/recipes", nil, authHeader(otherToken))
		defer listResp2.Body.Close()
		var list2 []map[string]interface{}
		json.NewDecoder(listResp2.Body).Decode(&list2)
		if len(list2) != 0 {
			t.Fatalf("expected 0 recipes for other user, got %d", len(list2))
		}
	})
}
