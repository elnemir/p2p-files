package reports

import (
	"bytes"
	"encoding/json"
	"filex.local/filex/internal/branding"
	"github.com/go-pdf/fpdf"
	"io"
)

// SecurityPDF renders the immutable JSON revision; its hash identifies the
// source JSON, not the separate PDF byte representation.
func SecurityPDF(w io.Writer, raw []byte, digest string) error {
	return SecurityPDFBranded(w, raw, digest, nil)
}
func SecurityPDFBranded(w io.Writer, raw []byte, digest string, brand *branding.Config) error {
	var pretty bytes.Buffer
	if e := json.Indent(&pretty, raw, "", "  "); e != nil {
		return e
	}
	p := fpdf.New("P", "mm", "A4", "")
	p.SetMargins(15, 15, 15)
	p.SetAutoPageBreak(true, 18)
	p.AddUTF8FontFromBytes("Body", "", font)
	p.AddPage()
	p.SetFont("Body", "", 16)
	name := "FileX"
	if brand != nil {
		name = brand.ProductName
		if e := brandHeading(p, *brand); e != nil {
			return e
		}
		p.SetFont("Body", "", 16)
	}
	if brand != nil {
		p.SetTitle(name+" security report", true)
		p.SetAuthor(name, true)
	}
	p.MultiCell(0, 9, name+" · Отчёт об инциденте / Security report", "", "L", false)
	p.SetFont("Body", "", 8)
	p.MultiCell(0, 5, "SHA-256 (canonical JSON):\n"+digest, "", "L", false)
	p.Ln(5)
	p.SetFont("Body", "", 9)
	p.MultiCell(0, 5, pretty.String(), "", "L", false)
	return p.Output(w)
}
