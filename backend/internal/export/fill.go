package export

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	reParagraph = regexp.MustCompile(`(?s)<w:p(?:\s[^>]*)?>.*?</w:p>`)
	reRun       = regexp.MustCompile(`(?s)<w:r(?:\s[^>]*)?>.*?</w:r>`)
	reText      = regexp.MustCompile(`(?s)<w:t(?:\s[^>]*)?>(.*?)</w:t>`)
	reRow       = regexp.MustCompile(`(?s)<w:tr(?:\s[^>]*)?>.*?</w:tr>`)
	rePlace     = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.]+)\s*\}\}`)
	rePart      = regexp.MustCompile(`^word/(document|header\d*|footer\d*)\.xml$`)
	reColor     = regexp.MustCompile(`<w:color[^>]*/>`)
	reStrike    = regexp.MustCompile(`<w:strike\s*/>`)
)

var unesc = strings.NewReplacer("&lt;", "<", "&gt;", ">", "&quot;", `"`, "&apos;", "'", "&amp;", "&")

// singleText returns the text of a run that has at most one <w:t> element.
func singleText(run string) (string, bool) {
	m := reText.FindAllStringSubmatch(run, -1)
	switch len(m) {
	case 0:
		return "", true
	case 1:
		return unesc.Replace(m[0][1]), true
	}
	return "", false
}

func setRunText(run, text string) string {
	if !reText.MatchString(run) {
		return run
	}
	return reText.ReplaceAllStringFunc(run, func(string) string {
		return `<w:t xml:space="preserve">` + runTextXML(text) + `</w:t>`
	})
}

// runTextXML escapes text; newlines become line breaks.
func runTextXML(text string) string {
	parts := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for i := range parts {
		parts[i] = esc(parts[i])
	}
	return strings.Join(parts, `</w:t><w:br/><w:t xml:space="preserve">`)
}

// normalizeParagraph merges placeholders that Word split across runs ("{{cli", "ent.na", "me}}")
// into the run where the placeholder starts, so later replacement keeps that run's formatting.
func normalizeParagraph(p string) string {
	idx := reRun.FindAllStringIndex(p, -1)
	if len(idx) == 0 || !strings.Contains(p, "{{") {
		return p
	}
	runs := make([]string, len(idx))
	texts := make([]string, len(idx))
	for i, ix := range idx {
		runs[i] = p[ix[0]:ix[1]]
		t, ok := singleText(runs[i])
		if !ok {
			t = "" // opaque run: leave as is
		}
		texts[i] = t
	}
	s := strings.Join(texts, "")
	matches := rePlace.FindAllStringSubmatchIndex(s, -1)
	if len(matches) == 0 {
		return p
	}
	var out strings.Builder
	last := 0
	off := 0
	for i, r := range runs {
		out.WriteString(p[last:idx[i][0]])
		rs, re := off, off+len(texts[i])
		off = re
		if _, ok := singleText(r); !ok || len(texts[i]) == 0 && !reText.MatchString(r) {
			out.WriteString(r)
			last = idx[i][1]
			continue
		}
		var nt strings.Builder
		cursor := rs
		for _, m := range matches {
			ms, me := m[0], m[1]
			if me <= rs || ms >= re {
				continue
			}
			if ms >= rs {
				nt.WriteString(s[cursor:ms])
				nt.WriteString("{{" + s[m[2]:m[3]] + "}}")
			}
			if e := minInt(me, re); e > cursor {
				cursor = e
			}
		}
		nt.WriteString(s[cursor:re])
		out.WriteString(setRunText(r, nt.String()))
		last = idx[i][1]
	}
	out.WriteString(p[last:])
	return out.String()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// fillRun replaces placeholders inside one run. Unknown keys are kept and recorded.
func fillRun(run string, vals map[string]string, unknown map[string]bool) string {
	t, ok := singleText(run)
	if !ok || !strings.Contains(t, "{{") {
		return run
	}
	hasStamp := false
	nt := rePlace.ReplaceAllStringFunc(t, func(tok string) string {
		key := rePlace.FindStringSubmatch(tok)[1]
		if v, found := vals[key]; found {
			if key == "stamp" {
				hasStamp = true
			}
			return v
		}
		unknown[key] = true
		return tok
	})
	out := setRunText(run, nt)
	if hasStamp {
		out = styleStamp(out, vals["stamp.color"], vals["stamp.strike"] != "")
	}
	return out
}

// styleStamp colours (and optionally strikes through) a run, keeping rPr children in schema order.
func styleStamp(run, colour string, strike bool) string {
	if colour == "" && !strike {
		return run
	}
	run = reColor.ReplaceAllString(run, "")
	run = reStrike.ReplaceAllString(run, "")
	add := ""
	if strike {
		add += "<w:strike/>"
	}
	if colour != "" {
		add += `<w:color w:val="` + colour + `"/>`
	}
	if i := strings.Index(run, "<w:sz "); i >= 0 && strings.Contains(run, "<w:rPr>") {
		return run[:i] + add + run[i:]
	}
	if strings.Contains(run, "</w:rPr>") {
		return strings.Replace(run, "</w:rPr>", add+"</w:rPr>", 1)
	}
	if strings.Contains(run, "<w:rPr/>") {
		return strings.Replace(run, "<w:rPr/>", "<w:rPr>"+add+"</w:rPr>", 1)
	}
	j := strings.Index(run, ">")
	return run[:j+1] + "<w:rPr>" + add + "</w:rPr>" + run[j+1:]
}

func fillAll(x string, vals map[string]string, unknown map[string]bool) string {
	return reRun.ReplaceAllStringFunc(x, func(r string) string { return fillRun(r, vals, unknown) })
}

func renderPart(name, x string, c Ctx, vals map[string]string, unknown map[string]bool, images []embeddedImage) string {
	x = reParagraph.ReplaceAllStringFunc(x, normalizeParagraph)
	ignore := map[string]bool{} // keys left over in repeated fragments are resolved by the final pass
	if name == "word/document.xml" {
		x = reParagraph.ReplaceAllStringFunc(x, func(paragraph string) string {
			return replaceImagePlaceholders(paragraph, images)
		})
		// one table row per line item
		x = reRow.ReplaceAllStringFunc(x, func(row string) string {
			if !strings.Contains(row, "{{line.") {
				return row
			}
			var b strings.Builder
			for _, l := range c.Doc.Lines {
				b.WriteString(fillAll(row, c.lineValues(l), ignore))
			}
			return b.String()
		})
		// one paragraph per work item / related document
		x = reParagraph.ReplaceAllStringFunc(x, func(p string) string {
			switch {
			case strings.Contains(p, "{{item."):
				items := c.Doc.WorkItems
				if len(items) == 0 {
					return fillAll(p, map[string]string{"item.text": "-", "item.title": "-", "item.detail": ""}, ignore)
				}
				var b strings.Builder
				for _, w := range items {
					b.WriteString(fillAll(p, map[string]string{"item.text": itemText(w), "item.title": w.Title, "item.detail": w.Detail}, ignore))
				}
				return b.String()
			case strings.Contains(p, "{{intake_item."):
				items := c.Doc.IntakeItems
				if len(items) == 0 {
					return fillAll(p, map[string]string{
						"intake_item.label": "No intake details recorded",
						"intake_item.value": "",
					}, ignore)
				}
				var b strings.Builder
				for _, item := range items {
					b.WriteString(fillAll(p, map[string]string{
						"intake_item.label": item.Label,
						"intake_item.value": item.Value,
					}, ignore))
				}
				return b.String()
			case strings.Contains(p, "{{diagnostic_section."):
				sections := c.Doc.DiagnosticSections
				if len(sections) == 0 {
					return fillAll(p, map[string]string{"diagnostic_section.text": "No diagnostic details recorded."}, ignore)
				}
				var b strings.Builder
				for _, section := range sections {
					b.WriteString(fillAll(p, map[string]string{"diagnostic_section.text": diagnosticText(section)}, ignore))
				}
				return b.String()
			case strings.Contains(p, "{{related}}"):
				rel := c.Related
				if len(rel) == 0 {
					rel = []string{"-"}
				}
				var b strings.Builder
				for _, r := range rel {
					b.WriteString(fillAll(p, map[string]string{"related": r}, ignore))
				}
				return b.String()
			}
			return p
		})
	}
	return fillAll(x, vals, unknown)
}

// Render fills a .docx template with the document's values. It returns the new .docx and the
// names of any placeholders it did not recognise (left visible in the output).
func Render(template []byte, c Ctx) ([]byte, []string, error) {
	zr, err := zip.NewReader(bytes.NewReader(template), int64(len(template)))
	if err != nil {
		return nil, nil, fmt.Errorf("template is not a valid .docx: %w", err)
	}
	parts := make(map[string][]byte, len(zr.File)+2)
	order := make([]string, 0, len(zr.File)+2)
	for _, file := range zr.File {
		reader, err := file.Open()
		if err != nil {
			return nil, nil, err
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			return nil, nil, err
		}
		parts[file.Name] = data
		order = append(order, file.Name)
	}
	images, err := brandingImages(c.Settings, parts)
	if err != nil {
		return nil, nil, err
	}
	if len(images) > 0 {
		const relationshipsPath = "word/_rels/document.xml.rels"
		const contentTypesPath = "[Content_Types].xml"
		relationships, err := addImageRelationships(parts[relationshipsPath], images)
		if err != nil {
			return nil, nil, err
		}
		parts[relationshipsPath] = relationships
		if _, exists := parts[relationshipsPath]; exists {
			found := false
			for _, name := range order {
				if name == relationshipsPath {
					found = true
					break
				}
			}
			if !found {
				order = append(order, relationshipsPath)
			}
		}
		contentTypes, ok := parts[contentTypesPath]
		if !ok {
			return nil, nil, fmt.Errorf("template has no [Content_Types].xml")
		}
		parts[contentTypesPath], err = addImageContentTypes(contentTypes, images)
		if err != nil {
			return nil, nil, err
		}
		for _, image := range images {
			path := "word/" + image.target
			parts[path] = image.data
			order = append(order, path)
		}
	}
	vals := c.Values()
	unknown := map[string]bool{}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	hasDoc := false
	for _, name := range order {
		data := parts[name]
		if rePart.MatchString(name) {
			if name == "word/document.xml" {
				hasDoc = true
			}
			data = []byte(renderPart(name, string(data), c, vals, unknown, images))
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate, Modified: time.Now()})
		if err != nil {
			return nil, nil, err
		}
		if _, err := w.Write(data); err != nil {
			return nil, nil, err
		}
	}
	if !hasDoc {
		return nil, nil, fmt.Errorf("template has no word/document.xml")
	}
	if err := zw.Close(); err != nil {
		return nil, nil, err
	}
	var warn []string
	for k := range unknown {
		warn = append(warn, k)
	}
	sort.Strings(warn)
	return buf.Bytes(), warn, nil
}
