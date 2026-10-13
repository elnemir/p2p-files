package branding

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func testPNG(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.NRGBA{R: 128, A: 255})
	var b bytes.Buffer
	if e := png.Encode(&b, img); e != nil {
		t.Fatal(e)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}
func TestBrandingValidation(t *testing.T) {
	if _, e := Normalize(Default()); e != nil {
		t.Fatal("defaults", e)
	}
	for name, change := range map[string]func(*Config){
		"blank": func(c *Config) { c.ProductName = " " }, "long": func(c *Config) { c.ProductName = strings.Repeat("Ж", 81) },
		"header": func(c *Config) { c.ProductName = "Brand\r\nBcc: example" }, "bidi": func(c *Config) { c.ProductName = "Brand\u202e" },
		"css": func(c *Config) { c.Light.Accent = "red;display:none" }, "contrast": func(c *Config) { c.Dark.Foreground = c.Dark.Background },
		"border": func(c *Config) { c.Light.Border = c.Light.Surface }, "button": func(c *Config) { c.Light.OnAccent = c.Light.Accent },
		"theme": func(c *Config) { c.DefaultTheme = "external" }, "icons": func(c *Config) { c.IconStyle = "https://evil.test/icons" },
		"url": func(c *Config) { c.LogoLight = "https://127.0.0.1/private" }, "svg": func(c *Config) { c.Favicon = "data:image/svg+xml;base64,PHN2Zz4=" },
		"spoofed": func(c *Config) {
			c.LogoLight = "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("<script>alert(1)</script>"))
		},
		"oversized": func(c *Config) { c.LogoLight = "data:image/png;base64," + strings.Repeat("A", 200000) },
	} {
		t.Run(name, func(t *testing.T) {
			c := Default()
			change(&c)
			if _, e := Normalize(c); e == nil {
				t.Fatal("accepted invalid branding")
			}
		})
	}
}
func TestBrandingRasterBoundsAndNormalization(t *testing.T) {
	valid := testPNG(t, 32, 32)
	raw, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(valid, "data:image/png;base64,"))
	polluted := "data:image/png;base64," + base64.StdEncoding.EncodeToString(append(raw, []byte("<script>metadata</script>")...))
	normalized, e := NormalizeAsset(polluted, false)
	if e != nil {
		t.Fatal(e)
	}
	decoded, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(normalized, "data:image/png;base64,"))
	if bytes.Contains(decoded, []byte("<script>")) {
		t.Fatal("trailing bytes retained")
	}
	if _, e = NormalizeAsset(testPNG(t, 1025, 1), false); e == nil {
		t.Fatal("logo width budget bypass")
	}
	if _, e = NormalizeAsset(testPNG(t, 1, 513), false); e == nil {
		t.Fatal("logo height budget bypass")
	}
	if _, e = NormalizeAsset(testPNG(t, 257, 1), true); e == nil {
		t.Fatal("favicon width budget bypass")
	}
	if _, e = NormalizeAsset(strings.Replace(valid, "image/png", "image/jpeg", 1), false); e == nil {
		t.Fatal("media type spoof accepted")
	}
	if _, e = NormalizeAsset(valid[:len(valid)-8], false); e == nil {
		t.Fatal("truncation accepted")
	}
	if _, e = NormalizeAsset(valid, true); e != nil {
		t.Fatal(e)
	}
}
