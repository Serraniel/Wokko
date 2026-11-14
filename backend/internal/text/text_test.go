package text

import (
	"sync"
	"testing"
)

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

// TestNormalizeIsSafeForConcurrentUse: requests normalize text at the same time. A shared
// transformer used to corrupt itself and panic with "slice bounds out of range".
func TestNormalizeIsSafeForConcurrentUse(t *testing.T) {
	words := []string{"Käse", "Hühnerbrühe", "Crème fraîche", "Weißwein-Essig", "onion", "Paprikapulver edelsüß"}
	want := map[string]string{}
	for _, w := range words {
		want[w] = Normalize(w)
	}
	var wg sync.WaitGroup
	errs := make(chan string, 100)
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 2000; i++ {
				w := words[(g+i)%len(words)]
				if got := Normalize(w); got != want[w] {
					select {
					case errs <- w + " → " + got:
					default:
					}
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}
