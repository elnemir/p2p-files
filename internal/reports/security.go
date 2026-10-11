package reports

import (
	"bytes"
	"encoding/json"
	"github.com/go-pdf/fpdf"
	"io"
)

// SecurityPDF renders the immutable JSON revision; its hash identifies the
// source JSON, not the separate PDF byte representation.
func SecurityPDF(w io.Writer, raw []byte, digest string) error {
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
	p.MultiCell(0, 9, "FileX · Отчёт об инциденте / Security report", "", "L", false)
	p.SetFont("Body", "", 8)
	p.MultiCell(0, 5, "SHA-256 (canonical JSON):\n"+digest, "", "L", false)
	p.Ln(5)
	p.SetFont("Body", "", 9)
	p.MultiCell(0, 5, pretty.String(), "", "L", false)
	return p.Output(w)
}
