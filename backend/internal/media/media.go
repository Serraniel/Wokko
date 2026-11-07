// Package media validates and stores recipe photos. Uploads are decoded and re-encoded, so
// only real images reach the disk, and phone photos are rotated upright.
package media

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoding
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoding
)

const (
	MaxUploadBytes = 20 << 20
	maxPixels      = 80_000_000
	fullSize       = 1920
	thumbSize      = 640
)

var ErrInvalidImage = errors.New("not a supported image (JPEG, PNG or WebP)")

type Store struct{ Dir string }

// Save stores an uploaded image and its thumbnail and returns the file name to keep. New
// files are complete on disk before the caller points a recipe at them.
func (s Store) Save(r io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return "", fmt.Errorf("read upload: %w", err)
	}
	if len(data) > MaxUploadBytes {
		return "", fmt.Errorf("%w: file is larger than %d MB", ErrInvalidImage, MaxUploadBytes>>20)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", ErrInvalidImage
	}
	if cfg.Width*cfg.Height > maxPixels {
		return "", fmt.Errorf("%w: image has too many pixels", ErrInvalidImage)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", ErrInvalidImage
	}
	img = orient(img, jpegOrientation(data))

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate name: %w", err)
	}
	name := hex.EncodeToString(b) + ".jpg"
	if err := os.MkdirAll(s.Dir, 0o750); err != nil {
		return "", fmt.Errorf("create uploads folder: %w", err)
	}
	if err := s.write(name, img, fullSize); err != nil {
		return "", err
	}
	if err := s.write(ThumbName(name), img, thumbSize); err != nil {
		_ = os.Remove(filepath.Join(s.Dir, name))
		return "", err
	}
	return name, nil
}

// ThumbName returns the thumbnail's file name for a photo.
func ThumbName(name string) string { return strings.TrimSuffix(name, ".jpg") + ".thumb.jpg" }

// OriginalName maps a photo or thumbnail file back to the name stored on the recipe.
func OriginalName(file string) string {
	return strings.TrimSuffix(file, ".thumb.jpg") + strings.Repeat(".jpg", b2i(strings.HasSuffix(file, ".thumb.jpg")))
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Delete removes a photo and its thumbnail.
func (s Store) Delete(name string) error {
	if name == "" || name != filepath.Base(name) {
		return fmt.Errorf("invalid photo name %q", name)
	}
	var errs []error
	for _, n := range []string{name, ThumbName(name)} {
		if err := os.Remove(filepath.Join(s.Dir, n)); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Path resolves a served file name inside the uploads folder.
func (s Store) Path(file string) (string, bool) {
	if file == "" || file != filepath.Base(file) || strings.HasPrefix(file, ".") {
		return "", false
	}
	return filepath.Join(s.Dir, file), true
}

func (s Store) write(name string, img image.Image, max int) error {
	scaled := fit(img, max)
	tmp, err := os.CreateTemp(s.Dir, ".upload-*")
	if err != nil {
		return fmt.Errorf("create photo file: %w", err)
	}
	defer os.Remove(tmp.Name()) // no-op after the rename below
	if err := jpeg.Encode(tmp, scaled, &jpeg.Options{Quality: 85}); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("encode photo: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("flush photo: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close photo: %w", err)
	}
	if err := os.Chmod(tmp.Name(), 0o640); err != nil {
		return fmt.Errorf("set photo permissions: %w", err)
	}
	if err := os.Rename(tmp.Name(), filepath.Join(s.Dir, name)); err != nil {
		return fmt.Errorf("finish photo: %w", err)
	}
	return nil
}

func fit(img image.Image, max int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= max && h <= max {
		return img
	}
	if w >= h {
		h, w = h*max/w, max
	} else {
		w, h = w*max/h, max
	}
	dst := image.NewRGBA(image.Rect(0, 0, max1(w), max1(h)))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	return dst
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

// jpegOrientation reads the EXIF orientation tag, returning 1 (upright) if there is none.
func jpegOrientation(d []byte) int {
	if len(d) < 4 || d[0] != 0xFF || d[1] != 0xD8 {
		return 1
	}
	for i := 2; i+4 <= len(d); {
		if d[i] != 0xFF {
			return 1
		}
		marker := d[i+1]
		size := int(binary.BigEndian.Uint16(d[i+2:]))
		if marker == 0xDA || size < 2 || i+2+size > len(d) {
			return 1
		}
		if marker == 0xE1 && size > 14 && string(d[i+4:i+10]) == "Exif\x00\x00" {
			return exifOrientation(d[i+10 : i+2+size])
		}
		i += 2 + size
	}
	return 1
}

func exifOrientation(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 1
	}
	off := int(bo.Uint32(t[4:]))
	if off < 0 || off+2 > len(t) {
		return 1
	}
	n := int(bo.Uint16(t[off:]))
	for i := 0; i < n; i++ {
		e := off + 2 + i*12
		if e+12 > len(t) {
			return 1
		}
		if bo.Uint16(t[e:]) == 0x0112 {
			if v := int(bo.Uint16(t[e+8:])); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

func orient(img image.Image, o int) image.Image {
	if o <= 1 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch o {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
