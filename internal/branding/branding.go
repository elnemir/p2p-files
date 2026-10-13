// Package branding owns public, nonsecret installation presentation.
package branding

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxAssetBytes = 128 << 10
const MaxRequestBytes = 600 << 10

//go:embed defaults.json
var defaults []byte

var ErrInvalid = errors.New("invalid branding")
var ErrConflict = errors.New("branding revision conflict")
var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type Palette struct {
	Background    string `json:"background"`
	Foreground    string `json:"foreground"`
	Muted         string `json:"muted"`
	Surface       string `json:"surface"`
	Border        string `json:"border"`
	Accent        string `json:"accent"`
	OnAccent      string `json:"onAccent"`
	Danger        string `json:"danger"`
	DangerSurface string `json:"dangerSurface"`
}
type Config struct {
	ProductName  string  `json:"productName"`
	TaglineRU    string  `json:"taglineRu"`
	TaglineEN    string  `json:"taglineEn"`
	FooterText   string  `json:"footerText"`
	DefaultTheme string  `json:"defaultTheme"`
	IconStyle    string  `json:"iconStyle"`
	Light        Palette `json:"light"`
	Dark         Palette `json:"dark"`
	LogoLight    string  `json:"logoLight"`
	LogoDark     string  `json:"logoDark"`
	Favicon      string  `json:"favicon"`
}

func Default() Config {
	var c Config
	// The embedded source is validated by tests and cannot depend on runtime input.
	if e := json.Unmarshal(defaults, &c); e != nil {
		panic(e)
	}
	return c
}
func textValid(s string, limit int) bool {
	if !utf8.ValidString(s) || utf8.RuneCountInString(s) > limit {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
	}
	return true
}
func luminance(s string) float64 {
	n, _ := strconv.ParseUint(s[1:], 16, 32)
	linear := func(v uint64) float64 {
		f := float64(v) / 255
		if f <= 0.04045 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	return 0.2126*linear((n>>16)&255) + 0.7152*linear((n>>8)&255) + 0.0722*linear(n&255)
}
func Contrast(a, b string) float64 {
	x, y := luminance(a), luminance(b)
	return (math.Max(x, y) + 0.05) / (math.Min(x, y) + 0.05)
}
func (p Palette) Validate() error {
	for _, c := range []string{p.Background, p.Foreground, p.Muted, p.Surface, p.Border, p.Accent, p.OnAccent, p.Danger, p.DangerSurface} {
		if !colorPattern.MatchString(c) {
			return ErrInvalid
		}
	}
	for _, bg := range []string{p.Background, p.Surface} {
		if Contrast(p.Foreground, bg) < 4.5 || Contrast(p.Muted, bg) < 4.5 || Contrast(p.Accent, bg) < 4.5 || Contrast(p.Border, bg) < 3 {
			return ErrInvalid
		}
	}
	if Contrast(p.OnAccent, p.Accent) < 4.5 || Contrast(p.Danger, p.DangerSurface) < 4.5 {
		return ErrInvalid
	}
	return nil
}

// NormalizeAsset bounds work before decoding and discards all original metadata.
func NormalizeAsset(value string, favicon bool) (string, error) {
	if value == "" {
		return "", nil
	}
	if len(value) > base64.StdEncoding.EncodedLen(MaxAssetBytes)+32 {
		return "", ErrInvalid
	}
	var format, payload string
	for _, kind := range []string{"png", "jpeg"} {
		if s, ok := strings.CutPrefix(value, "data:image/"+kind+";base64,"); ok {
			format, payload = kind, s
			break
		}
	}
	if format == "" || strings.ContainsAny(payload, "\r\n") {
		return "", ErrInvalid
	}
	raw, e := base64.StdEncoding.Strict().DecodeString(payload)
	if e != nil || len(raw) > MaxAssetBytes {
		return "", ErrInvalid
	}
	cfg, kind, e := image.DecodeConfig(bytes.NewReader(raw))
	maxW, maxH := 1024, 512
	if favicon {
		maxW, maxH = 256, 256
	}
	if e != nil || kind != format || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > maxW || cfg.Height > maxH {
		return "", ErrInvalid
	}
	img, _, e := image.Decode(bytes.NewReader(raw))
	if e != nil {
		return "", ErrInvalid
	}
	var out bytes.Buffer
	normalized := image.NewNRGBA(img.Bounds())
	draw.Draw(normalized, normalized.Bounds(), img, img.Bounds().Min, draw.Src)
	if e = png.Encode(&out, normalized); e != nil {
		return "", e
	}
	if out.Len() > MaxAssetBytes {
		return "", ErrInvalid
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(out.Bytes()), nil
}
func Normalize(c Config) (Config, error) {
	if strings.TrimSpace(c.ProductName) != c.ProductName || c.ProductName == "" || !textValid(c.ProductName, 80) || !textValid(c.TaglineRU, 160) || !textValid(c.TaglineEN, 160) || !textValid(c.FooterText, 160) {
		return c, ErrInvalid
	}
	if c.DefaultTheme != "system" && c.DefaultTheme != "light" && c.DefaultTheme != "dark" {
		return c, ErrInvalid
	}
	if c.IconStyle != "outline" && c.IconStyle != "rounded" && c.IconStyle != "duotone" {
		return c, ErrInvalid
	}
	if e := c.Light.Validate(); e != nil {
		return c, e
	}
	if e := c.Dark.Validate(); e != nil {
		return c, e
	}
	for _, asset := range []struct {
		v    *string
		icon bool
	}{{&c.LogoLight, false}, {&c.LogoDark, false}, {&c.Favicon, true}} {
		v, e := NormalizeAsset(*asset.v, asset.icon)
		if e != nil {
			return c, e
		}
		*asset.v = v
	}
	return c, nil
}

// UnmarshalJSON keeps omitted or null fields from silently diverging from the contract.
func (c *Config) UnmarshalJSON(raw []byte) error {
	type plain Config
	var fields map[string]json.RawMessage
	if e := json.Unmarshal(raw, &fields); e != nil {
		return e
	}
	for _, name := range []string{"productName", "taglineRu", "taglineEn", "footerText", "defaultTheme", "iconStyle", "light", "dark", "logoLight", "logoDark", "favicon"} {
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return ErrInvalid
		}
	}
	var next plain
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(&next); e != nil {
		return e
	}
	*c = Config(next)
	return nil
}
