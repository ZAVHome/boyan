package cover

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"

	"golang.org/x/image/draw"
)

// ProcessCover принимает сырые байты обложки любого формата и масштабирует ее до максимального разрешения maxDim.
func ProcessCover(data []byte, maxDim int) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty cover data")
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()

	if origW == 0 || origH == 0 {
		return nil, fmt.Errorf("invalid image dimensions: %dx%d", origW, origH)
	}

	// Если изображение меньше или равно лимиту, можно просто перекодировать в JPEG
	if maxDim <= 0 || (origW <= maxDim && origH <= maxDim) {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
			return nil, fmt.Errorf("encode jpeg: %w", err)
		}
		return buf.Bytes(), nil
	}

	// Рассчитываем пропорциональное масштабирование
	var newW, newH int
	if origW > origH {
		newW = maxDim
		newH = origH * maxDim / origW
	} else {
		newH = maxDim
		newW = origW * maxDim / origH
	}
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	// Масштабирование средствами чистого Go без CGO
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		return nil, fmt.Errorf("encode resized jpeg: %w", err)
	}

	return buf.Bytes(), nil
}
