package reports

import (
	"bytes"
	"encoding/base64"
	"filex.local/filex/internal/branding"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBrandedPDFRasterAndLegacyJSON(t *testing.T) {
	img := image.NewNRGBA64(image.Rect(0, 0, 20, 12))
	img.Set(1, 1, color.NRGBA64{R: 50000, G: 30000, A: 40000})
	var source bytes.Buffer
	if e := png.Encode(&source, img); e != nil {
		t.Fatal(e)
	}
	b := branding.Default()
	b.ProductName = strings.Repeat("Длинный бренд ", 5)
	b.LogoLight = "data:image/png;base64," + base64.StdEncoding.EncodeToString(source.Bytes())
	normalized, e := branding.NormalizeAsset(b.LogoLight, false)
	if e != nil {
		t.Fatal(e)
	}
	b.LogoLight = normalized
	raw := []byte(`{"reportId":"fixture","revision":1,"createdAt":"2026-10-13T10:00:00Z","timezone":"UTC","locale":"ru","files":[],"downloads":[]}`)
	var pdf bytes.Buffer
	if e = RenderBranded(&pdf, raw, strings.Repeat("a", 64), "pdf", &b); e != nil {
		t.Fatal("normalized 16-bit source not printable", e)
	}
	if dir := os.Getenv("FILEX_EVIDENCE_DIR"); dir != "" {
		if e = os.WriteFile(filepath.Join(dir, "branding-report.pdf"), pdf.Bytes(), 0600); e != nil {
			t.Fatal(e)
		}
	}
	if !bytes.HasPrefix(pdf.Bytes(), []byte("%PDF-")) {
		t.Fatal("PDF missing")
	}
	var json bytes.Buffer
	if e = RenderBranded(&json, raw, "hash", "json", &b); e != nil || !bytes.Equal(json.Bytes(), raw) {
		t.Fatal("strict JSON changed", e)
	}
	var legacy bytes.Buffer
	if e = Render(&legacy, raw, "hash", "pdf"); e != nil {
		t.Fatal("legacy renderer", e)
	}
	var security bytes.Buffer
	if e = SecurityPDFBranded(&security, []byte(`{"id":"fixture","evidence":{}}`), "hash", &b); e != nil {
		t.Fatal("security renderer", e)
	}
}
