package main

import (
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
	"log"
	"flag"
	"fmt"
)

const (
	naFormat format = iota
	gifFormat
	jpegFormat
	pngFormat
)

type format int

func main() {
	source := flag.String("source", "", "Define the source of existing media to be sanitized")
	target := flag.String("target", "", "Define the target for sanitized media to be created")
	flag.Parse()

	if *source == "" {
		log.Fatal("Existing media source is required")
	}

	if *target == "" {
		log.Fatal("Target for sanitized media is required")
	}

	fmt.Printf("Hooray, we got here and nothing exploded: source = %s, target = %s", *source, *target)
}

func writeMinimalImage(x, y int, file string) {
	writer, err := os.Create(file)
	if err != nil {
		log.Fatal(err)
	}

	image := image.NewGray(image.Rect(0,0,x,y))

	switch getFormat(file) {
	case pngFormat:
		png.Encode(writer, image)
	case gifFormat:
		options := &gif.Options{
			NumColors: 1,
			Quantizer: nil,
			Drawer: nil,
		}
		gif.Encode(writer, image, options)
	case jpegFormat:
		options := &jpeg.Options{
			Quality: 1,
		}
		jpeg.Encode(writer, image, options)
	}
}

func imageResolution(img image.Image) (int, int) {
	bounds := img.Bounds()
	min := bounds.Min
	max := bounds.Max
	return max.X - min.X, max.Y - min.Y
}

func openImage(file string) (image.Image, error) {
	reader, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	switch getFormat(file) {
	case pngFormat:
		return png.Decode(reader)
	case gifFormat:
		return gif.Decode(reader)
	case jpegFormat:
		return jpeg.Decode(reader)
	default:
		return nil, errors.New("Unrecognized image extension")
	}
}

func getFormat(file string) format {
	fileParts := strings.Split(file, ".")
	ext := fileParts[len(fileParts) - 1]

	switch strings.ToLower(ext) {
	case "png":
		return pngFormat
	case "gif":
		return gifFormat
	case "jpg", "jpeg", "jpe", "jif", "jfif", "jfi":
		return jpegFormat
	default:
		return naFormat
	}
}
