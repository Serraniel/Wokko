package text

import "testing"

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{"Käse": "kase", "Tomatensoße": "tomatensosse", "  CRÈME  fraîche ": "creme fraiche", "Огурец": "огурец", "ÄPFEL": "apfel"} {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPick(t *testing.T) {
	items := []Translation[string]{{"fr", "oignon"}, {"en", "onion"}, {"de", "Zwiebel"}}
	for _, c := range []struct{ loc, want string }{{"de", "Zwiebel"}, {"en", "onion"}, {"es", "onion"}} {
		if got, _ := Pick(items, c.loc); got != c.want {
			t.Errorf("Pick(%q) = %q, want %q", c.loc, got, c.want)
		}
	}
	if got, ok := Pick([]Translation[string]{{"fr", "oignon"}}, "de"); !ok || got != "oignon" {
		t.Errorf("should fall back to any language, got %q", got)
	}
	if _, ok := Pick[string](nil, "de"); ok {
		t.Error("empty must report not found")
	}
}

func TestPickLocale(t *testing.T) {
	for in, want := range map[string]string{"de-DE,de;q=0.9,en;q=0.8": "de", "": "en", "*": "en", "fr": "fr"} {
		if got := PickLocale(in, "en"); got != want {
			t.Errorf("PickLocale(%q) = %q, want %q", in, got, want)
		}
	}
}
