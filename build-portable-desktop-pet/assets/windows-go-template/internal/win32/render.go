package win32

import (
	"image"
	"image/color"

	"portable-desktop-pet/internal/model"
)

const hitAlphaThreshold = 16

type Command uint16

const (
	CommandToggleTopmost Command = 1001
	CommandTogglePaused  Command = 1002
	CommandResetSize     Command = 1003
	CommandExit          Command = 1004
)

type preparedFrame struct {
	Pixels []byte
	Width  int
	Height int
	Stride int
	Mask   alphaMask
}

type alphaMask struct {
	Pixels []byte
	Width  int
	Height int
	Stride int
}

func (m alphaMask) Hit(x, y int) bool {
	if x < 0 || y < 0 || x >= m.Width || y >= m.Height {
		return false
	}
	index := y*m.Stride + x
	return index >= 0 && index < len(m.Pixels) && m.Pixels[index] >= hitAlphaThreshold
}

func prepareFrame(src image.Image) preparedFrame {
	if src == nil {
		return preparedFrame{}
	}

	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	stride := width * 4
	pixels := make([]byte, stride*height)
	alphaStride := width
	alpha := make([]byte, alphaStride*height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := color.NRGBAModel.Convert(src.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.NRGBA)
			a := uint16(c.A)
			offset := y*stride + x*4
			pixels[offset+0] = byte((uint16(c.B)*a + 127) / 255)
			pixels[offset+1] = byte((uint16(c.G)*a + 127) / 255)
			pixels[offset+2] = byte((uint16(c.R)*a + 127) / 255)
			pixels[offset+3] = c.A
			alpha[y*alphaStride+x] = c.A
		}
	}

	return preparedFrame{
		Pixels: pixels,
		Width:  width,
		Height: height,
		Stride: stride,
		Mask: alphaMask{
			Pixels: alpha,
			Width:  width,
			Height: height,
			Stride: alphaStride,
		},
	}
}

func scaleFrame(src image.Image, target model.Size) *image.NRGBA {
	if src == nil || src.Bounds().Empty() {
		return image.NewNRGBA(image.Rectangle{})
	}
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	targetWidth, targetHeight := target.W, target.H
	if targetWidth <= 0 {
		targetWidth = width
	}
	if targetHeight <= 0 {
		targetHeight = height
	}

	scaledWidth, scaledHeight := targetWidth, targetHeight
	if width*targetHeight > height*targetWidth {
		scaledHeight = max(1, (height*targetWidth+width/2)/width)
	} else {
		scaledWidth = max(1, (width*targetHeight+height/2)/height)
	}

	dst := image.NewNRGBA(image.Rect(0, 0, scaledWidth, scaledHeight))
	for y := 0; y < scaledHeight; y++ {
		sourceY := bounds.Min.Y + y*height/scaledHeight
		for x := 0; x < scaledWidth; x++ {
			sourceX := bounds.Min.X + x*width/scaledWidth
			dst.Set(x, y, src.At(sourceX, sourceY))
		}
	}
	return dst
}
