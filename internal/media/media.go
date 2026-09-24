package media

import (
	"bytes"
	"encoding/binary"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
)

func Placeholder(seed string, size int) ([]byte, error) {
	h := fnv.New32a()
	h.Write([]byte(seed))
	v := h.Sum32()

	top := color.RGBA{
		R: uint8(70 + v%110),
		G: uint8(80 + (v>>8)%110),
		B: uint8(110 + (v>>16)%120),
		A: 255,
	}
	bottom := color.RGBA{
		R: uint8(top.R/2 + 30),
		G: uint8(top.G/2 + 30),
		B: uint8(top.B/2 + 40),
		A: 255,
	}

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		t := float64(y) / float64(size-1)
		row := color.RGBA{
			R: blend(top.R, bottom.R, t),
			G: blend(top.G, bottom.G, t),
			B: blend(top.B, bottom.B, t),
			A: 255,
		}
		for x := 0; x < size; x++ {
			img.SetRGBA(x, y, row)
		}
	}
	band := size / 6
	for y := size/2 - band/2; y < size/2+band/2; y++ {
		for x := 0; x < size; x++ {
			c := img.RGBAAt(x, y)
			img.SetRGBA(x, y, color.RGBA{R: lighten(c.R), G: lighten(c.G), B: lighten(c.B), A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func SilenceWAV(seconds int) []byte {
	const (
		rate     = 8000
		channels = 1
		bits     = 16
	)
	if seconds <= 0 {
		seconds = 30
	}
	blockAlign := channels * bits / 8
	byteRate := rate * blockAlign
	dataLen := rate * seconds * blockAlign
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+dataLen))
	buf.WriteString("WAVEfmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(channels))
	binary.Write(&buf, binary.LittleEndian, uint32(rate))
	binary.Write(&buf, binary.LittleEndian, uint32(byteRate))
	binary.Write(&buf, binary.LittleEndian, uint16(blockAlign))
	binary.Write(&buf, binary.LittleEndian, uint16(bits))
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(dataLen))
	buf.Write(make([]byte, dataLen))
	return buf.Bytes()
}

func blend(a, b uint8, t float64) uint8 {
	return uint8(float64(a)*(1-t) + float64(b)*t)
}

func lighten(v uint8) uint8 {
	if v > 205 {
		return 255
	}
	return v + 50
}
