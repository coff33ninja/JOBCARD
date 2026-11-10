package export

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

// A template edited in Word can carry a logo (word/media/*), a header with its own placeholders
// and drawing runs. The renderer must fill the placeholders and leave images untouched.
func TestHeaderPlaceholdersAndMediaArePreserved(t *testing.T) {
	body := `<w:p><w:r><w:drawing><wp:inline xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"><wp:docPr id="1" name="Logo"/></wp:inline></w:drawing></w:r>` +
		`<w:r><w:t>{{business.name}}</w:t></w:r></w:p>`
	base, err := assembleDocx(body)
	if err != nil {
		t.Fatal(err)
	}
	png := []byte{0x89, 'P', 'N', 'G', 0, 1, 2, 3, 250, 251, 252}
	header := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:hdr ` + wns + `><w:p><w:r><w:drawing><wp:inline xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"><wp:docPr id="2" name="HeaderLogo"/></wp:inline></w:drawing></w:r>` +
		`<w:r><w:t>{{business.name}} - {{doc.number}}</w:t></w:r></w:p></w:hdr>`

	zr, _ := zip.NewReader(bytes.NewReader(base), int64(len(base)))
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range zr.File {
		rc, _ := f.Open()
		data, _ := io.ReadAll(rc)
		rc.Close()
		w, _ := zw.Create(f.Name)
		w.Write(data)
	}
	for name, data := range map[string][]byte{"word/header1.xml": []byte(header), "word/media/image1.png": png} {
		w, _ := zw.Create(name)
		w.Write(data)
	}
	zw.Close()

	out, warn, err := Render(buf.Bytes(), sample("invoice"))
	if err != nil || len(warn) != 0 {
		t.Fatalf("%v %v", err, warn)
	}
	h := part(t, out, "word/header1.xml")
	wellFormed(t, h)
	if !strings.Contains(h, "Example IT - INV-2026-001") || !strings.Contains(h, "<w:drawing>") || strings.Contains(h, "{{") {
		t.Fatalf("header not filled or drawing lost:\n%s", h)
	}
	b := part(t, out, "word/document.xml")
	if !strings.Contains(b, "<w:drawing>") || !strings.Contains(b, "Example IT") {
		t.Fatalf("body drawing lost or placeholder not filled:\n%s", b)
	}
	if got := part(t, out, "word/media/image1.png"); got != string(png) {
		t.Fatal("image bytes changed")
	}
}
