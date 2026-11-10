// Package branding validates and decodes uploaded client-facing image assets.
package branding

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	// Register only the supported raster formats with DecodeConfig.
	_ "image/jpeg"
	_ "image/png"
	"strings"
)

// MaxImageBytes caps each uploaded branding asset.
const MaxImageBytes = 256 * 1024

// Image contains a validated raster image and its decoded dimensions.
type Image struct {
	Data        []byte
	Format      string
	ContentType string
	Width       int
	Height      int
}

// DecodeDataURL validates a PNG or JPEG data URL and decodes its metadata.
func DecodeDataURL(value string) (*Image, error) {
	if value == "" {
		return nil, nil
	}
	var contentType string
	switch {
	case strings.HasPrefix(value, "data:image/png;base64,"):
		contentType = "image/png"
	case strings.HasPrefix(value, "data:image/jpeg;base64,"):
		contentType = "image/jpeg"
	default:
		return nil, fmt.Errorf("image must be a base64 PNG or JPEG data URL")
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, "data:"+contentType+";base64,"))
	if err != nil {
		return nil, fmt.Errorf("image data is not valid base64")
	}
	if len(data) == 0 || len(data) > MaxImageBytes {
		return nil, fmt.Errorf("image must be between 1 byte and %d bytes", MaxImageBytes)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (contentType == "image/png" && format != "png") || (contentType == "image/jpeg" && format != "jpeg") {
		return nil, fmt.Errorf("image data does not match its PNG or JPEG type")
	}
	if config.Width < 1 || config.Height < 1 || config.Width > 4096 || config.Height > 4096 ||
		config.Width*config.Height > 16_000_000 {
		return nil, fmt.Errorf("image dimensions must be at most 4096 by 4096 pixels")
	}
	return &Image{
		Data: data, Format: format, ContentType: contentType, Width: config.Width, Height: config.Height,
	}, nil
}
