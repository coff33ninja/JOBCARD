// Package export adds optional uploaded artwork as inline WordprocessingML images.
package export

import (
	"fmt"
	"math"
	"strings"

	"jobcard/internal/branding"
	"jobcard/internal/domain"
)

type embeddedImage struct {
	placeholder    string
	relationshipID string
	target         string
	contentType    string
	data           []byte
	width          int64
	height         int64
	docPrID        int
	name           string
}

func brandingImages(settings domain.Settings, existing map[string][]byte) ([]embeddedImage, error) {
	requests := []struct {
		value, placeholder, id, name string
		maxWidth, maxHeight          int64
	}{
		{settings.CompanyLogo, "business.logo", "rIdJobCardLogo", "jobcard-company-logo", 1_645_920, 640_080},
		{settings.TechnicianSignatureImage, "authorization.technician_signature_image", "rIdJobCardTechnicianSignature", "jobcard-technician-signature", 1_280_160, 457_200},
	}
	images := make([]embeddedImage, 0, len(requests))
	for i, request := range requests {
		decoded, err := branding.DecodeDataURL(request.value)
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", request.placeholder, err)
		}
		if decoded == nil {
			continue
		}
		targetName := request.name + "." + decoded.Format
		for suffix := 2; ; suffix++ {
			if _, exists := existing["word/media/"+targetName]; !exists {
				break
			}
			targetName = fmt.Sprintf("%s-%d.%s", request.name, suffix, decoded.Format)
		}
		id := request.id
		for suffix := 2; relationshipIDExists(existing["word/_rels/document.xml.rels"], id); suffix++ {
			id = fmt.Sprintf("%s%d", request.id, suffix)
		}
		width, height := fittedImageSize(decoded.Width, decoded.Height, request.maxWidth, request.maxHeight)
		images = append(images, embeddedImage{
			placeholder: request.placeholder, relationshipID: id, target: "media/" + targetName,
			contentType: decoded.ContentType, data: decoded.Data, width: width, height: height,
			docPrID: 1001 + i, name: strings.ReplaceAll(request.placeholder, ".", "-"),
		})
	}
	return images, nil
}

func fittedImageSize(width, height int, maxWidth, maxHeight int64) (int64, int64) {
	if int64(width)*maxHeight > int64(height)*maxWidth {
		return maxWidth, int64(math.Round(float64(maxWidth) * float64(height) / float64(width)))
	}
	return int64(math.Round(float64(maxHeight) * float64(width) / float64(height))), maxHeight
}

func relationshipIDExists(data []byte, id string) bool {
	return strings.Contains(string(data), `Id="`+id+`"`)
}

func replaceImagePlaceholders(paragraph string, images []embeddedImage) string {
	return reRun.ReplaceAllStringFunc(paragraph, func(run string) string {
		text, ok := singleText(run)
		if !ok {
			return run
		}
		for _, image := range images {
			token := "{{" + image.placeholder + "}}"
			index := strings.Index(text, token)
			if index < 0 {
				continue
			}
			var replacement strings.Builder
			if before := text[:index]; before != "" {
				replacement.WriteString(setRunText(run, before))
			}
			replacement.WriteString(image.drawingRun())
			if after := text[index+len(token):]; after != "" {
				replacement.WriteString(setRunText(run, after))
			}
			return replacement.String()
		}
		return run
	})
}

func (img embeddedImage) drawingRun() string {
	return fmt.Sprintf(
		`<w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"><wp:extent cx="%d" cy="%d"/><wp:docPr id="%d" name="%s"/><wp:cNvGraphicFramePr><a:graphicFrameLocks xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" noChangeAspect="1"/></wp:cNvGraphicFramePr><a:graphic xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:nvPicPr><pic:cNvPr id="%d" name="%s"/><pic:cNvPicPr/></pic:nvPicPr><pic:blipFill><a:blip r:embed="%s" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill><pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r>`,
		img.width, img.height, img.docPrID, img.name, img.docPrID, img.name,
		img.relationshipID, img.width, img.height,
	)
}

func imageRelationship(img embeddedImage) string {
	return fmt.Sprintf(`<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="%s"/>`,
		img.relationshipID, img.target)
}

func addImageRelationships(data []byte, images []embeddedImage) ([]byte, error) {
	if len(data) == 0 {
		data = []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`)
	}
	if !strings.Contains(string(data), "</Relationships>") {
		return nil, fmt.Errorf("document relationships are malformed")
	}
	var entries strings.Builder
	for _, img := range images {
		entries.WriteString(imageRelationship(img))
	}
	return []byte(strings.Replace(string(data), "</Relationships>", entries.String()+"</Relationships>", 1)), nil
}

func addImageContentTypes(data []byte, images []embeddedImage) ([]byte, error) {
	text := string(data)
	if !strings.Contains(text, "</Types>") {
		return nil, fmt.Errorf("document content types are malformed")
	}
	extensions := map[string]string{
		"image/png":  "png",
		"image/jpeg": "jpeg",
	}
	var entries strings.Builder
	for contentType, extension := range extensions {
		needed := false
		for _, img := range images {
			if img.contentType == contentType {
				needed = true
				break
			}
		}
		if !needed || strings.Contains(strings.ToLower(text), `extension="`+extension+`"`) {
			continue
		}
		entries.WriteString(fmt.Sprintf(`<Default Extension="%s" ContentType="%s"/>`, extension, contentType))
	}
	return []byte(strings.Replace(text, "</Types>", entries.String()+"</Types>", 1)), nil
}
