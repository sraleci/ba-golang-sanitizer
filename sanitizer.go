package main

import (
	"os"
	"image"
	"image/gif"
	"fmt"
	"strings"
	"image/jpeg"
	"image/png"
	"log"
	"errors"
)

func main() {
	files := []string{
		"example.gif",
		"example.jpg",
		"example.png",
	}

	for _, file := range files {
		img, err := openImage(file)
		if err != nil {
			log.Fatal(err)
		}

		x, y := imageResolution(img)
		fmt.Printf("Image %s resolution is (%d, %d)\n", file, x, y)
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

	fileParts := strings.Split(file, ".")
	ext := fileParts[len(fileParts) - 1]

	switch strings.ToLower(ext) {
	case "png":
		return png.Decode(reader)
	case "gif":
		return gif.Decode(reader)
	case "jpg", "jpeg", "jpe", "jif", "jfif", "jfi":
		return jpeg.Decode(reader)
	default:
		return nil, errors.New("Unrecognized image extension")
	}
}

