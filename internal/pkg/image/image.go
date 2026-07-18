package image //nolint:goconst

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

type VariantSize string

const (
	VariantThumbnail VariantSize = "thumbnail"
	VariantSmall     VariantSize = "small"
	VariantMedium    VariantSize = "medium"
	VariantLarge     VariantSize = "large"
	VariantOriginal  VariantSize = "original"
)

type VariantDimensions struct {
	MaxWidth  int
	MaxHeight int
}

var VariantSizes = map[VariantSize]VariantDimensions{
	VariantThumbnail: {MaxWidth: 150, MaxHeight: 150},
	VariantSmall:     {MaxWidth: 320, MaxHeight: 240},
	VariantMedium:    {MaxWidth: 800, MaxHeight: 600},
	VariantLarge:     {MaxWidth: 1920, MaxHeight: 1080},
	VariantOriginal:  {MaxWidth: 0, MaxHeight: 0},
}

type ImageInfo struct {
	Width       int
	Height      int
	Format      string
	Orientation int
}

func Decode(r io.Reader) (*ImageInfo, image.Image, error) {
	src, format, err := image.Decode(r)
	if err != nil {
		return nil, nil, fmt.Errorf("image: failed to decode: %w", err)
	}

	bounds := src.Bounds()
	return &ImageInfo{
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
		Format: format,
	}, src, nil
}

func DecodeImage(path string) (image.Image, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("image: failed to open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	src, format, err := image.Decode(f)
	if err != nil {
		return nil, "", fmt.Errorf("image: failed to decode %s: %w", path, err)
	}

	return src, format, nil
}

func DecodeImageInto(path string) (*ImageInfo, image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("image: failed to open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	info, img, err := Decode(f)
	if err != nil {
		return nil, nil, fmt.Errorf("image: failed to decode %s: %w", path, err)
	}

	return info, img, nil
}

func GetImageInfo(path string) (*ImageInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("image: failed to open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	info, _, err := Decode(f)
	return info, err
}

func GenerateVariant(inputPath, outputPath string, size VariantSize) error {
	dims, ok := VariantSizes[size]
	if !ok {
		return fmt.Errorf("image: unknown variant size %q", size)
	}

	if size == VariantOriginal {
		return copyFile(inputPath, outputPath)
	}

	src, format, err := DecodeImage(inputPath)
	if err != nil {
		return err
	}

	resized := resizeImage(src, dims.MaxWidth, dims.MaxHeight)

	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("image: failed to create directory: %w", err)
	}

	return saveImage(resized, outputPath, format)
}

func GenerateThumbnail(inputPath, outputPath string) error {
	return GenerateVariant(inputPath, outputPath, VariantThumbnail)
}

func GenerateSmall(inputPath, outputPath string) error {
	return GenerateVariant(inputPath, outputPath, VariantSmall)
}

func GenerateMedium(inputPath, outputPath string) error {
	return GenerateVariant(inputPath, outputPath, VariantMedium)
}

func GenerateLarge(inputPath, outputPath string) error {
	return GenerateVariant(inputPath, outputPath, VariantLarge)
}

func ConvertToWebP(inputPath, outputPath string, quality int) error {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("image: failed to create directory: %w", err)
	}

	if cmd, err := exec.LookPath("cwebp"); err == nil {
		args := []string{
			"-q", fmt.Sprintf("%d", quality),
			inputPath,
			"-o", outputPath,
		}
		out, err := exec.Command(cmd, args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("image: cwebp failed: %w, output: %s", err, string(out))
		}
		return nil
	}

	src, format, err := DecodeImage(inputPath)
	if err != nil {
		return err
	}

	return saveImage(src, outputPath, format)
}

func ConvertToAVIF(inputPath, outputPath string, quality int) error {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("image: failed to create directory: %w", err)
	}

	if cmd, err := exec.LookPath("avifenc"); err == nil {
		args := []string{
			"-q", fmt.Sprintf("%d", quality),
			inputPath,
			"-o", outputPath,
		}
		out, err := exec.Command(cmd, args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("image: avifenc failed: %w, output: %s", err, string(out))
		}
		return nil
	}

	src, format, err := DecodeImage(inputPath)
	if err != nil {
		return err
	}

	return saveImage(src, outputPath, format)
}

func ExtractEXIF(path string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	result["orientation"] = 1

	return result, nil
}

func resizeImage(src image.Image, maxWidth, maxHeight int) image.Image {
	if maxWidth <= 0 || maxHeight <= 0 {
		return src
	}

	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w <= maxWidth && h <= maxHeight {
		return src
	}

	ratio := float64(w) / float64(h)
	targetRatio := float64(maxWidth) / float64(maxHeight)

	var newW, newH int
	if ratio > targetRatio {
		newW = maxWidth
		newH = int(float64(maxWidth) / ratio)
	} else {
		newH = maxHeight
		newW = int(float64(maxHeight) * ratio)
	}

	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	return dst
}

func centerCrop(src image.Image, width, height int) image.Image {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	srcRatio := float64(srcW) / float64(srcH)
	targetRatio := float64(width) / float64(height)

	var cropW, cropH int
	if srcRatio > targetRatio {
		cropH = srcH
		cropW = int(float64(srcH) * targetRatio)
	} else {
		cropW = srcW
		cropH = int(float64(srcW) / targetRatio)
	}

	if cropW > srcW {
		cropW = srcW
	}
	if cropH > srcH {
		cropH = srcH
	}

	x := (srcW - cropW) / 2
	y := (srcH - cropH) / 2

	cropped := image.NewRGBA(image.Rect(0, 0, cropW, cropH))
	draw.Draw(cropped, cropped.Bounds(), src, image.Point{X: x, Y: y}, draw.Over)

	resized := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.ApproxBiLinear.Scale(resized, resized.Bounds(), cropped, cropped.Bounds(), draw.Over, nil)

	return resized
}

func saveImage(img image.Image, path, format string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("image: failed to create file %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		return jpeg.Encode(f, img, &jpeg.Options{Quality: 85})
	case "png":
		return png.Encode(f, img)
	case "gif":
		return gif.Encode(f, img, nil)
	default:
		return jpeg.Encode(f, img, &jpeg.Options{Quality: 85})
	}
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("image: failed to open source: %w", err)
	}
	defer func() { _ = srcFile.Close() }()

	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("image: failed to create directory: %w", err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("image: failed to create destination: %w", err)
	}
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("image: failed to copy: %w", err)
	}

	return nil
}

func SupportedFormats() []string {
	return []string{"jpeg", "jpg", "png", "gif", "webp"}
}

func GetFormat(mimeType string) string {
	switch strings.ToLower(mimeType) {
	case "image/jpeg":
		return "jpeg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	case "image/avif":
		return "avif"
	default:
		return "jpeg"
	}
}

var JPEGQuality = 85
