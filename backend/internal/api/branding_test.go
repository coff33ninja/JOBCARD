package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"testing"
)

func TestBrandingImagesPersistAndRejectUnsupportedFormats(t *testing.T) {
	ts := testServer(t, "")
	base := ts.URL + "/api/v1"

	var pngData bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	if err := png.Encode(&pngData, img); err != nil {
		t.Fatal(err)
	}
	logo := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData.Bytes())
	signature := logo
	body := map[string]any{
		"businessName": "Example Business", "person": "Technician",
		"phone": "000", "email": "tech@example.test", "location": "City",
		"bankName": "Bank", "accountHolder": "Example", "accountNumber": "123", "branchCode": "000",
		"taxRate": 0, "defaultTermsDays": 7, "quoteValidityDays": 14,
		"companyLogo": logo, "technicianSignatureImage": signature,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	update := do(t, http.MethodPut, base+"/settings", string(payload), jsonHdr)
	if update.StatusCode != http.StatusOK {
		t.Fatalf("save branding images: status %d", update.StatusCode)
	}
	var saved map[string]any
	if err := json.NewDecoder(update.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved["companyLogo"] != logo || saved["technicianSignatureImage"] != signature {
		t.Fatalf("branding images were not persisted: %+v", saved)
	}

	body["companyLogo"] = "data:image/svg+xml;base64,PHN2Zy8+"
	payload, err = json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	invalid := do(t, http.MethodPut, base+"/settings", string(payload), jsonHdr)
	if invalid.StatusCode != http.StatusBadRequest {
		t.Fatalf("unsupported branding format should be rejected, status %d", invalid.StatusCode)
	}
}
