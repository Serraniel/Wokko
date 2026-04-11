package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestPickLang(t *testing.T) {
	tests := []struct {
		lang   string
		en     string
		de     string
		expect string
	}{
		{"en", "Flour", "Mehl", "Flour"},
		{"de", "Flour", "Mehl", "Mehl"},
		{"de", "Flour", "", "Flour"}, // fallback when DE empty
		{"de", "", "", ""},
	}
	for _, tt := range tests {
		got := pickLang(tt.lang, tt.en, tt.de)
		if got != tt.expect {
			t.Errorf("pickLang(%q, %q, %q) = %q, want %q", tt.lang, tt.en, tt.de, got, tt.expect)
		}
	}
}

func TestResolveLang(t *testing.T) {
	e := echo.New()
	tests := []struct {
		header string
		expect string
	}{
		{"de", "de"},
		{"de-DE", "de"},
		{"de-AT", "de"},
		{"en", "en"},
		{"en-US", "en"},
		{"", "en"},
		{"fr", "en"},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Language", tt.header)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		got := resolveLang(c)
		if got != tt.expect {
			t.Errorf("resolveLang(%q) = %q, want %q", tt.header, got, tt.expect)
		}
	}
}
