package httpapi

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func testImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 16), G: uint8(y * 16), B: 128, A: 255})
		}
	}
	return img
}

// jpegWithExif собирает валидный JPEG с APP1 EXIF-сегментом сразу после SOI —
// так реальный фотофайл несёт метаданные (включая GPS).
func jpegWithExif(t *testing.T) []byte {
	t.Helper()
	var orig bytes.Buffer
	if err := jpeg.Encode(&orig, testImage(), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	payload := append([]byte("Exif\x00\x00"), []byte("MM\x00\x2aFAKE-GPS-DATA")...)
	segLen := len(payload) + 2 // длина включает само поле длины
	seg := append([]byte{0xFF, 0xE1, byte(segLen >> 8), byte(segLen)}, payload...)
	src := orig.Bytes()
	out := make([]byte, 0, len(src)+len(seg))
	out = append(out, src[:2]...) // SOI
	out = append(out, seg...)
	out = append(out, src[2:]...)
	return out
}

// ТЗ: приватность — EXIF (в том числе геолокация) не должен попадать в хранилище.
func TestStripImageMetadataRemovesEXIF(t *testing.T) {
	in := jpegWithExif(t)
	if !bytes.Contains(in, []byte("Exif\x00\x00")) {
		t.Fatal("тестовый JPEG должен содержать EXIF-сегмент")
	}
	out := stripImageMetadata(in, "image/jpeg")
	if bytes.Contains(out, []byte("Exif\x00\x00")) {
		t.Error("EXIF-сегмент выжил после перекодирования")
	}
	if _, format, err := image.Decode(bytes.NewReader(out)); err != nil || format != "jpeg" {
		t.Errorf("после очистки JPEG должен декодироваться: format=%q err=%v", format, err)
	}
}

func TestStripImageMetadataPNG(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, testImage()); err != nil {
		t.Fatal(err)
	}
	out := stripImageMetadata(buf.Bytes(), "image/png")
	img, format, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil || format != "png" {
		t.Fatalf("после очистки PNG должен декодироваться: format=%q err=%v", format, err)
	}
	if img.Width != 16 || img.Height != 16 {
		t.Errorf("размер изменился: %dx%d, want 16x16", img.Width, img.Height)
	}
}

// Не-изображения и битые файлы сохраняются как есть.
func TestStripImageMetadataPassthrough(t *testing.T) {
	pdf := []byte("%PDF-1.4 fake document")
	if got := stripImageMetadata(pdf, "application/pdf"); !bytes.Equal(got, pdf) {
		t.Error("PDF должен проходить без изменений")
	}
	txt := []byte("просто текстовая заметка")
	if got := stripImageMetadata(txt, "text/plain"); !bytes.Equal(got, txt) {
		t.Error("текст должен проходить без изменений")
	}
	broken := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 'g', 'a', 'r', 'b', 'a', 'g', 'e'}
	if got := stripImageMetadata(broken, "image/jpeg"); !bytes.Equal(got, broken) {
		t.Error("недекодируемый JPEG должен сохраняться как есть")
	}
}

func TestAllowedContentTypes(t *testing.T) {
	allowed := []string{"image/jpeg", "image/png", "image/gif", "image/webp", "application/pdf", "text/plain"}
	for _, ct := range allowed {
		if !allowedContentTypes[ct] {
			t.Errorf("тип %s должен быть разрешён (ТЗ: фото, pdf, txt)", ct)
		}
	}
	for _, ct := range []string{"application/x-msdownload", "video/mp4", "application/zip"} {
		if allowedContentTypes[ct] {
			t.Errorf("тип %s не должен быть разрешён", ct)
		}
	}
}
