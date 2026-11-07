package media

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func jpegWithOrientation(t *testing.T, w, h, o int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	tiff := []byte{'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 0x01, 3, 0, 1, 0, 0, 0, byte(o), 0, 0, 0, 0, 0, 0, 0}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	size := len(payload) + 2
	app1 := append([]byte{0xFF, 0xE1, byte(size >> 8), byte(size)}, payload...)
	raw := b.Bytes()
	return append(append([]byte{0xFF, 0xD8}, app1...), raw[2:]...)
}

func TestPhonePhotosAreRotatedUpright(t *testing.T) {
	data := jpegWithOrientation(t, 40, 20, 6)
	if got := jpegOrientation(data); got != 6 {
		t.Fatalf("orientation: %d", got)
	}
	s := Store{Dir: t.TempDir()}
	name, err := s.Save(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(s.Dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, err := jpeg.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 20 || cfg.Height != 40 {
		t.Fatalf("not rotated: %dx%d", cfg.Width, cfg.Height)
	}
}

func TestLargePhotosAreScaledAndThumbnailed(t *testing.T) {
	var b bytes.Buffer
	if err := jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, 3000, 1500)), nil); err != nil {
		t.Fatal(err)
	}
	s := Store{Dir: t.TempDir()}
	name, err := s.Save(&b)
	if err != nil {
		t.Fatal(err)
	}
	for file, want := range map[string]int{name: fullSize, ThumbName(name): thumbSize} {
		f, err := os.Open(filepath.Join(s.Dir, file))
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := jpeg.DecodeConfig(f)
		f.Close()
		if err != nil || cfg.Width != want {
			t.Fatalf("%s: width %d, want %d (%v)", file, cfg.Width, want, err)
		}
	}
	if OriginalName(ThumbName(name)) != name || OriginalName(name) != name {
		t.Fatal("OriginalName must map both files to the stored name")
	}
	if err := s.Delete(name); err != nil {
		t.Fatal(err)
	}
	left, _ := os.ReadDir(s.Dir)
	if len(left) != 0 {
		t.Fatalf("left: %v", left)
	}
}

func TestOnlyRealImagesAreAccepted(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	for _, in := range []string{"", "<svg onload=alert(1)>", "GIF89a not really", "\xff\xd8\xff broken jpeg"} {
		if _, err := s.Save(strings.NewReader(in)); !errors.Is(err, ErrInvalidImage) {
			t.Errorf("%q: got %v", in, err)
		}
	}
	left, _ := os.ReadDir(s.Dir)
	if len(left) != 0 {
		t.Fatalf("rejected uploads left files: %v", left)
	}
}

func TestPathRejectsEscapes(t *testing.T) {
	s := Store{Dir: "/data/uploads"}
	for _, bad := range []string{"", "../wokko.db", "a/b.jpg", ".hidden"} {
		if _, ok := s.Path(bad); ok {
			t.Errorf("%q must be rejected", bad)
		}
	}
	if err := s.Delete("../x.jpg"); err == nil {
		t.Error("Delete must reject paths")
	}
}
