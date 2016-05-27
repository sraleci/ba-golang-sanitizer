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
	"io"
)

const (
	naFormat format = iota
	gifFormat
	jpegFormat
	pngFormat
)

const fileMode = 0777

type format int

type imageMap map[format]map[int]map[int][]string // map[format,x,y] -> [fileNames, ...]

var (
	verbose bool
	targetSanitized string
)

func main() {
	source := flag.String("source", "", "Define the source of existing media to be sanitized")
	target := flag.String("target", "", "Define the target for sanitized media to be created")
	throttle := flag.Int("throttle", 0, "Define the buffer size for concurrent image reads/writes")
	verboseFlag := flag.Bool("verbose", false, "Define logging behavior")
	flag.Parse()
	verbose = *verboseFlag

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
		targetFile.Close()
		log.Fatal("Target already exists")
	}
	targetSanitized = *target + string(os.PathSeparator) + ".sanitized" + string(os.PathSeparator)
	os.MkdirAll(targetSanitized, fileMode)

	// Begin recursive read of source media
	c := make(chan int, *throttle)
	info := make(imageMap)

	// go readFile(*sourceFile, c, &info)
	readFile(*sourceFile, c, &info, *source, *target)

	// Write minimal images for all unique formats and resolutions
	for imageFormat, xMap := range info {
		for x, yMap := range xMap {
			for y, _ := range yMap {
				targetFile, err := getMinimalFileName(x, y, imageFormat)
				if err == nil {
					if verbose {
						fmt.Printf("Creating sanitized image file %s\n", targetFile)
					}
					writeMinimalImage(x, y, targetFile)
				}
			}
		}
	}

	// Link image files to the minimal images
	for imageFormat, xMap := range info {
		for x, yMap := range xMap {
			for y, fileList := range yMap {
				for _, file := range fileList {
					targetFile, err := getMinimalFileName(x, y, imageFormat)
					if err == nil {
						oldFile := strings.Replace(file, *source, *target, -1)
						if verbose {
							fmt.Printf("Establishing a hard link from %s to sanitized %s\n", oldFile, targetFile)
						}
						os.Link(oldFile, targetFile)
					}
				}
			}
		}
	}

	// <-c // causes deadlock - added in order to see go routines finish
}

func getMinimalFileName(x, y int, imageFormat format) (string, error) {
	switch imageFormat {
	case gifFormat:
		return fmt.Sprintf("%s%dx%d.gif", targetSanitized, x, y), nil
	case jpegFormat:
		return fmt.Sprintf("%s%dx%d.jpg", targetSanitized, x, y), nil
	case pngFormat:
		return fmt.Sprintf("%s%dx%d.png", targetSanitized, x, y), nil
	default:
		return "", errors.New("Unrecognized image extension")
	}
}

func readFile(file os.File, c chan int, info *imageMap, source, target string) {
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
		fmt.Printf("Children of %s are %v\n", fileName, children)
		for _, child := range children {
			func() {
				childFile, err := os.Open(fileName + string(os.PathSeparator) + child)
				if err != nil {
					log.Fatal(err)
				}
				defer childFile.Close()
				// go readFile(*childFile, c, info)
				readFile(*childFile, c, info, source, target)
			}()
		}
	} else {
		img, err := openImage(&file)
		if err != nil {
			// We can go ahead and copy the file if it's not a matching image format
			targetFile := strings.Replace(fileName, source, target, -1)
			splitTarget := strings.Split(targetFile, string(os.PathSeparator))
			targetFileParent := strings.Join(splitTarget[:len(splitTarget)-1], string(os.PathSeparator))
			_, err := os.Open(targetFileParent)
			if err != nil {
				os.MkdirAll(targetFileParent, fileMode)
			}

			file.Close()
			out, err := os.Create(targetFile)
			if err != nil {
				log.Fatal(err)
			}
			defer out.Close()
			in, err := os.Open(fileName)
			if err != nil {
				log.Fatal(err)
			}
			if _, err = io.Copy(out, in); err != nil {
				log.Fatal(err)
			} else if verbose {
				fmt.Printf("Copied non-image file from %s to %s\n", in.Name(), out.Name())
			}
			out.Sync()
		} else {
			fileFormat := getFormat(fileName)
			x, y := imageResolution(img)

			xMap := (*info)[fileFormat]
			if xMap == nil {
				xMap = make(map[int]map[int][]string)
				(*info)[fileFormat] = xMap
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
