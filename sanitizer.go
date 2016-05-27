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
//	"fmt"
)

const (
	naFormat format = iota
	gifFormat
	jpegFormat
	pngFormat
)

type format int

type mediaInfo struct {
	imageMap      map[format]map[int]map[int][]string // map[format,x,y] -> [fileNames, ...]
	nonMediaPaths []string
}


func main() {
	source := flag.String("source", "", "Define the source of existing media to be sanitized")
	target := flag.String("target", "", "Define the target for sanitized media to be created")
	throttle := flag.Int("throttle", 0, "Define the buffer size for concurrent image reads/writes")
	flag.Parse()

	if *source == "" {
		log.Fatal("Existing media source is required")
	}

	if *target == "" {
		log.Fatal("Target for sanitized media is required")
	}

	// Check if source directory exists, want it to be a directory
	sourceInfo, err := os.Stat(*source)
	if err != nil {
		log.Fatal("Ensure that source exists")
	}

	if !sourceInfo.IsDir() {
		log.Fatal("Ensure that source is a directory")
	}

	sourceFile, err := os.Open(*source)
	if err != nil {
		log.Fatal(err)
	}
	defer sourceFile.Close()

	// Check if target directory already exists, fatal if it already exists
	targetFile, _ := os.Open(*target)
	if targetFile != nil {
		log.Fatal("Target already exists")
	}

	// Begin recursive read of source media
	c := make(chan int, *throttle)
	info := mediaInfo{
		imageMap: make(map[format]map[int]map[int][]string),
		nonMediaPaths: make([]string, 0),
	}

	// go readFile(*sourceFile, c, &info)
	readFile(*sourceFile, c, &info)

	// <-c // causes deadlock - added in order to see go routines finish
}

func readFile(file os.File, c chan int, info *mediaInfo) {
	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}

	fileName := file.Name()
	if fileInfo.IsDir() {
		children, err := file.Readdirnames(0)
		if err != nil {
			log.Fatal(err)
		}
		for _, child := range children {
			childFile, err := os.Open(fileName + string(os.PathSeparator) + child)
			if err != nil {
				log.Fatal(err)
			}
			// go readFile(*childFile, c, info)
			readFile(*childFile, c, info)
		}
	} else {
		img, err := openImage(&file)
		if err != nil {
			if (*info).nonMediaPaths == nil {
				(*info).nonMediaPaths = make([]string, 0)
			}
			(*info).nonMediaPaths = append((*info).nonMediaPaths, fileName)
		} else {
			fileFormat := getFormat(fileName)
			x, y := imageResolution(img)

			formatMap := (*info).imageMap
			if formatMap == nil {
				formatMap = make(map[format]map[int]map[int][]string)
				(*info).imageMap = formatMap
			}

			xMap := formatMap[fileFormat]
			if xMap == nil {
				xMap = make(map[int]map[int][]string)
				formatMap[fileFormat] = xMap
			}

			yMap := xMap[x]
			if yMap == nil {
				yMap = make(map[int][]string)
				xMap[x] = yMap
			}

			fileList := yMap[y]
			if fileList == nil {
				fileList = make([]string, 0)
			}

			yMap[y] = append(fileList, fileName)
		}
	}
}

func writeMinimalImage(x, y int, file string) {
	writer, err := os.Create(file)
	if err != nil {
		log.Fatal(err)
	}

	image := image.NewGray(image.Rect(0, 0, x, y))

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

func openImage(reader *os.File) (image.Image, error) {
	switch getFormat(reader.Name()) {
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
