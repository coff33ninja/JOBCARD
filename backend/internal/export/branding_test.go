package export

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"io"
	"strings"
	"testing"

	"jobcard/internal/domain"
)

func TestDefaultTemplatesEmbedBrandingImages(t *testing.T) {
	var pngData bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	if err := png.Encode(&pngData, img); err != nil {
		t.Fatal(err)
	}
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData.Bytes())

	for _, docType := range []string{"invoice", "quotation", "jobcard"} {
		t.Run(docType, func(t *testing.T) {
			ctx := sample(docType)
			ctx.Settings.CompanyLogo = dataURL
			ctx.Settings.TechnicianSignatureImage = dataURL
			ctx.Doc.Authorization.TechnicianSignature = "Technician Name"
			ctx.Doc.Authorization.TechnicianDate = "2026-11-08"
			template, err := DefaultTemplate(docType)
			if err != nil {
				t.Fatal(err)
			}
			rendered, warnings, err := Render(template, ctx)
			if err != nil || len(warnings) != 0 {
				t.Fatalf("render: %v, warnings: %v", err, warnings)
			}
			document := part(t, rendered, "word/document.xml")
			if strings.Contains(document, dataURL) || strings.Count(document, "<w:drawing>") != 2 {
				t.Fatalf("expected two embedded images in document XML")
			}
			wellFormed(t, document)
			if !strings.Contains(document, "Technician Name") || !strings.Contains(document, "8 November 2026") {
				t.Fatal("technician name or date missing from document")
			}
			relationships := part(t, rendered, "word/_rels/document.xml.rels")
			wellFormed(t, relationships)
			if !strings.Contains(relationships, "rIdJobCardLogo") || !strings.Contains(relationships, "rIdJobCardTechnicianSignature") {
				t.Fatalf("image relationships missing: %s", relationships)
			}
			wellFormed(t, part(t, rendered, "[Content_Types].xml"))
			zipReader, err := zip.NewReader(bytes.NewReader(rendered), int64(len(rendered)))
			if err != nil {
				t.Fatal(err)
			}
			mediaCount := 0
			for _, file := range zipReader.File {
				if strings.HasPrefix(file.Name, "word/media/jobcard-") {
					reader, err := file.Open()
					if err != nil {
						t.Fatal(err)
					}
					media, err := io.ReadAll(reader)
					reader.Close()
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(media, pngData.Bytes()) {
						t.Fatalf("embedded image %s differs from upload", file.Name)
					}
					mediaCount++
				}
			}
			if mediaCount != 2 {
				t.Fatalf("want 2 embedded media files, got %d", mediaCount)
			}
		})
	}
}

func TestRenderWithoutBrandingRemovesImagePlaceholders(t *testing.T) {
	template, err := DefaultTemplate("jobcard")
	if err != nil {
		t.Fatal(err)
	}
	ctx := sample("jobcard")
	ctx.Settings = domain.Settings{BusinessName: "Example"}
	rendered, warnings, err := Render(template, ctx)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("render without branding: %v, warnings: %v", err, warnings)
	}
	document := part(t, rendered, "word/document.xml")
	if strings.Contains(document, "{{business.logo}}") || strings.Contains(document, "{{authorization.technician_signature_image}}") {
		t.Fatal("empty branding image placeholders should be removed")
	}
}
