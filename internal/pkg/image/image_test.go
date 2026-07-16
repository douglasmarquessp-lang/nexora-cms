package image

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func createTestImage(t *testing.T, path string) {
	t.Helper()
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)

	img := image.NewRGBA(image.Rect(0, 0, 800, 600))
	for x := 0; x < 800; x++ {
		for y := 0; y < 600; y++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 128, 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if _, err := f.Write(createMinimalJPEG()); err != nil {
		t.Fatal(err)
	}
}

func createMinimalJPEG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	f, _ := os.CreateTemp("", "test-*.jpg")
	defer f.Close()
	defer os.Remove(f.Name())
	jpeg.Encode(f, img, &jpeg.Options{Quality: 50})
	data, _ := os.ReadFile(f.Name())
	return data
}

func TestSupportedFormats(t *testing.T) {
	formats := SupportedFormats()
	if len(formats) == 0 {
		t.Error("expected non-empty formats")
	}
}

func TestGetFormat(t *testing.T) {
	tests := []struct {
		mime string
		want string
	}{
		{"image/jpeg", "jpeg"},
		{"image/png", "png"},
		{"image/gif", "gif"},
		{"image/webp", "webp"},
		{"unknown", "jpeg"},
	}
	for _, tt := range tests {
		got := GetFormat(tt.mime)
		if got != tt.want {
			t.Errorf("GetFormat(%q) = %q, want %q", tt.mime, got, tt.want)
		}
	}
}

func TestVariantSizes(t *testing.T) {
	if len(VariantSizes) != 5 {
		t.Errorf("expected 5 variant sizes, got %d", len(VariantSizes))
	}

	thumb := VariantSizes[VariantThumbnail]
	if thumb.MaxWidth != 150 || thumb.MaxHeight != 150 {
		t.Errorf("thumbnail dimensions wrong: %+v", thumb)
	}
}

func TestResizeImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))
	resized := resizeImage(img, 100, 100)
	bounds := resized.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 75 {
		t.Errorf("resized to %dx%d, want 100x75", bounds.Dx(), bounds.Dy())
	}
}

func TestCenterCrop(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))
	cropped := centerCrop(img, 100, 100)
	bounds := cropped.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Errorf("cropped to %dx%d, want 100x100", bounds.Dx(), bounds.Dy())
	}
}

func TestGetImageInfo(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "test.jpg")

	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	f, _ := os.Create(imgPath)
	jpeg.Encode(f, img, &jpeg.Options{Quality: 50})
	f.Close()

	info, err := GetImageInfo(imgPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Width != 200 {
		t.Errorf("width = %d, want 200", info.Width)
	}
	if info.Height != 100 {
		t.Errorf("height = %d, want 100", info.Height)
	}
}

func TestGenerateVariant(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "src.jpg")
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))
	f, _ := os.Create(srcPath)
	jpeg.Encode(f, img, &jpeg.Options{Quality: 50})
	f.Close()

	dstPath := filepath.Join(tmpDir, "thumb.jpg")
	err := GenerateThumbnail(srcPath, dstPath)
	if err != nil {
		t.Fatal(err)
	}

	info, err := GetImageInfo(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Width > 150 || info.Height > 150 {
		t.Errorf("thumb too big: %dx%d", info.Width, info.Height)
	}
}
