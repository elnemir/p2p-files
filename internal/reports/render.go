package reports

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"filex.local/filex/internal/branding"
	"fmt"
	"github.com/go-pdf/fpdf"
	"io"
	"strconv"
	"strings"
	"time"
)

//go:embed fonts/DejaVuSansCondensed.ttf
var font []byte

func SafeCSV(v string) string {
	trimmed := strings.TrimLeft(v, " \t\r\n")
	if strings.HasPrefix(trimmed, "=") || strings.HasPrefix(trimmed, "+") || strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "@") || strings.ContainsAny(v[:min(1, len(v))], "\t\r\n") {
		return "'" + v
	}
	return v
}
func Render(w io.Writer, raw []byte, hash, format string) error {
	return RenderBranded(w, raw, hash, format, nil)
}
func RenderBranded(w io.Writer, raw []byte, hash, format string, brand *branding.Config) error {
	if format == "json" {
		_, e := w.Write(raw)
		return e
	}
	var s struct {
		ID          string `json:"reportId"`
		ContentHash string `json:"documentContentHash"`
		Revision    int    `json:"revision"`
		Title       string `json:"title"`
		State       string `json:"state"`
		Created     string `json:"createdAt"`
		Locale      string `json:"locale"`
		Timezone    string `json:"timezone"`
		Files       []struct {
			Path  string  `json:"path"`
			Size  int64   `json:"size"`
			SHA   *string `json:"sha256"`
			State string  `json:"state"`
		} `json:"files"`
		Downloads []struct {
			At     string `json:"occurredAt"`
			Count  int64  `json:"fileCount"`
			Bytes  int64  `json:"totalBytes"`
			Status string `json:"status"`
		} `json:"downloads"`
		Admin []map[string]any `json:"administrativeEvents"`
	}
	if e := json.Unmarshal(raw, &s); e != nil {
		return e
	}
	if format == "csv" {
		c := csv.NewWriter(w)
		c.UseCRLF = true
		if e := c.Write([]string{"recordType", "pathOrTime", "bytes", "checksumOrCount", "status", "canonicalHash", "documentContentHash"}); e != nil {
			return e
		}
		for _, f := range s.Files {
			sha := ""
			if f.SHA != nil {
				sha = *f.SHA
			}
			if e := c.Write([]string{"file", SafeCSV(f.Path), fmt.Sprint(f.Size), sha, f.State, hash, s.ContentHash}); e != nil {
				return e
			}
		}
		for _, d := range s.Downloads {
			if e := c.Write([]string{"download", d.At, fmt.Sprint(d.Bytes), fmt.Sprint(d.Count), d.Status, hash, s.ContentHash}); e != nil {
				return e
			}
		}
		for _, event := range s.Admin {
			b, e := json.Marshal(event)
			if e != nil {
				return e
			}
			if e = c.Write([]string{"admin", SafeCSV(string(b)), "", "", "", hash, s.ContentHash}); e != nil {
				return e
			}
		}
		c.Flush()
		return c.Error()
	}
	if format != "pdf" {
		return fmt.Errorf("unsupported report format")
	}
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(18, 18, 18)
	pdf.SetAutoPageBreak(true, 18)
	pdf.SetCompression(true)
	pdf.AddUTF8FontFromBytes("Body", "", font)
	pdf.SetFont("Body", "", 10)
	name := "FileX"
	if brand != nil {
		name = brand.ProductName
	}
	pdf.SetTitle(name+" report", true)
	pdf.SetAuthor(name, brand != nil)
	pdf.AddPage()
	if brand != nil {
		if e := brandHeading(pdf, *brand); e != nil {
			return e
		}
	}
	title := "Отчёт о передаче"
	filesTitle := "Файлы"
	journalTitle := "Журнал скачиваний"
	if s.Locale == "en" {
		title = "Transfer report"
		filesTitle = "Files"
		journalTitle = "Download journal"
	}
	pdf.SetFont("Body", "", 20)
	pdf.MultiCell(0, 10, title, "", "L", false)
	pdf.SetFont("Body", "", 10)
	loc, e := time.LoadLocation(s.Timezone)
	if e != nil {
		return e
	}
	at, e := time.Parse(time.RFC3339Nano, s.Created)
	if e != nil {
		return e
	}
	pdf.SetCreationDate(at)
	pdf.SetModificationDate(at)
	bytesLabel := "байт"
	countLabel := "Файлов"
	pageLabel := "Страница"
	if s.Locale == "en" {
		bytesLabel = "bytes"
		countLabel = "Files"
		pageLabel = "Page"
	}
	label := func(v string) string {
		if s.Locale == "en" {
			return v
		}
		labels := map[string]string{"ready": "готово", "leased": "скачивается", "deleting": "удаляется", "deleted": "удалено", "completed": "завершено", "expired": "срок истёк", "revoked": "отозвано", "integrity_failed": "ошибка целостности", "storage_missing": "нет в хранилище"}
		if text, ok := labels[v]; ok {
			return text
		}
		return v
	}
	pdf.SetFooterFunc(func() {
		pdf.SetY(-13)
		pdf.SetFont("Body", "", 8)
		if brand == nil {
			pdf.CellFormat(0, 5, fmt.Sprintf("FileX · %s %d", pageLabel, pdf.PageNo()), "", 0, "R", false, 0, "")
		} else {
			pdf.MultiCell(0, 4, fmt.Sprintf("%s · %s %d", name, pageLabel, pdf.PageNo()), "", "R", false)
		}
	})
	pdf.MultiCell(0, 6, fmt.Sprintf("%s · #%d\n%s\n%s (%s)", s.ID, s.Revision, s.Title, at.In(loc).Format("2006-01-02 15:04:05"), s.Timezone), "", "L", false)
	pdf.Ln(4)
	pdf.SetFont("Body", "", 8)
	pdf.MultiCell(0, 5, "SHA-256 (canonical JSON):\n"+hash+"\nSHA-256 (document content):\n"+s.ContentHash, "", "L", false)
	pdf.Ln(6)
	pdf.SetFont("Body", "", 14)
	pdf.MultiCell(0, 8, filesTitle, "", "L", false)
	pdf.SetFont("Body", "", 10)
	for _, f := range s.Files {
		pdf.MultiCell(0, 6, fmt.Sprintf("%s\n%d %s · %s", f.Path, f.Size, bytesLabel, label(f.State)), "", "L", false)
		if f.SHA != nil {
			pdf.SetFont("Body", "", 8)
			pdf.MultiCell(0, 5, *f.SHA, "", "L", false)
			pdf.SetFont("Body", "", 10)
		}
		pdf.Ln(3)
	}
	pdf.SetFont("Body", "", 14)
	pdf.MultiCell(0, 8, journalTitle, "", "L", false)
	pdf.SetFont("Body", "", 10)
	for _, d := range s.Downloads {
		when, e := time.Parse(time.RFC3339Nano, d.At)
		if e != nil {
			return e
		}
		pdf.MultiCell(0, 6, fmt.Sprintf("%s · %s: %d · %d %s · %s", when.In(loc).Format("2006-01-02 15:04:05"), countLabel, d.Count, d.Bytes, bytesLabel, label(d.Status)), "", "L", false)
	}
	for _, event := range s.Admin {
		b, e := json.Marshal(event)
		if e != nil {
			return e
		}
		pdf.MultiCell(0, 5, string(b), "", "L", false)
	}
	var out bytes.Buffer
	if e = pdf.Output(&out); e != nil {
		return e
	}
	_, e = w.Write(out.Bytes())
	return e
}

// brandHeading uses only the immutable, normalized public presentation snapshot.
func brandHeading(pdf *fpdf.Fpdf, c branding.Config) error {
	if c.LogoLight != "" {
		raw, e := base64.StdEncoding.Strict().DecodeString(strings.TrimPrefix(c.LogoLight, "data:image/png;base64,"))
		if e != nil {
			return e
		}
		info := pdf.RegisterImageOptionsReader("branding", fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(raw))
		if info == nil {
			return pdf.Error()
		}
		w, h := info.Width(), info.Height()
		scale := min(50/w, 16/h)
		pdf.ImageOptions("branding", pdf.GetX(), pdf.GetY(), w*scale, h*scale, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
		pdf.Ln(h*scale + 3)
	}
	// PDFs use white paper even when the configured light palette is dark.
	accent := c.Light.Accent
	if branding.Contrast(accent, "#ffffff") < 4.5 {
		accent = "#000000"
	}
	color, e := strconv.ParseUint(strings.TrimPrefix(accent, "#"), 16, 32)
	if e != nil {
		return e
	}
	pdf.SetTextColor(int((color>>16)&255), int((color>>8)&255), int(color&255))
	pdf.SetFont("Body", "", 14)
	pdf.MultiCell(0, 7, c.ProductName, "", "L", false)
	pdf.Ln(3)
	pdf.SetTextColor(0, 0, 0)
	return pdf.Error()
}
