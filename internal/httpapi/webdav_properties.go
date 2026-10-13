package httpapi

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/transfer"
)

type davStoredProperty struct {
	Space, Local, Inner string
	Attr                []xml.Attr
}

func (n *davXMLNode) encode(e *xml.Encoder) error {
	start := xml.StartElement{Name: n.Name, Attr: n.Attr}
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	if err := n.encodeContent(e); err != nil {
		return err
	}
	return e.EncodeToken(start.End())
}
func (n *davXMLNode) encodeContent(e *xml.Encoder) error {
	for _, p := range n.Content {
		switch v := p.(type) {
		case string:
			if err := e.EncodeToken(xml.CharData(v)); err != nil {
				return err
			}
		case *davXMLNode:
			if err := v.encode(e); err != nil {
				return err
			}
		}
	}
	return nil
}
func (n *davXMLNode) outerXML() string {
	var b bytes.Buffer
	e := xml.NewEncoder(&b)
	if n.encode(e) != nil || e.Flush() != nil {
		return ""
	}
	return b.String()
}
func (n *davXMLNode) property() davStoredProperty {
	var b bytes.Buffer
	e := xml.NewEncoder(&b)
	if n.encodeContent(e) != nil || e.Flush() != nil {
		return davStoredProperty{}
	}
	return davStoredProperty{n.Name.Space, n.Name.Local, b.String(), n.Attr}
}
func (n *davXMLNode) text() string {
	var b strings.Builder
	for _, p := range n.Content {
		if v, ok := p.(string); ok {
			b.WriteString(v)
		}
	}
	return b.String()
}

func (d davAccess) proppatch(w http.ResponseWriter, r *http.Request, name string) error {
	root, e := davXML(w, r)
	if e != nil {
		return e
	}
	if root == nil || root.Name != (xml.Name{Space: "DAV:", Local: "propertyupdate"}) || len(root.Children) == 0 {
		return davError(400)
	}
	type change struct {
		Node   *davXMLNode
		Remove bool
		Status int
	}
	changes := []change{}
	for _, action := range root.Children {
		if action.Name.Space != "DAV:" || (action.Name.Local != "set" && action.Name.Local != "remove") || len(action.Children) != 1 {
			return davError(400)
		}
		prop := action.Children[0]
		if prop.Name != (xml.Name{Space: "DAV:", Local: "prop"}) || len(prop.Children) == 0 {
			return davError(400)
		}
		for _, p := range prop.Children {
			changes = append(changes, change{p, action.Name.Local == "remove", 200})
		}
	}
	if len(changes) > 256 {
		return davError(400)
	}
	tx, e := d.app.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(r.Context())
	if e = d.lock(r.Context(), tx); e != nil {
		return e
	}
	if e = d.conditions(r.Context(), tx, r, name); e != nil {
		return e
	}
	key := ""
	if name != "" {
		key, e = domain.PathKey(name, d.settings.MaxPathBytes)
		if e != nil {
			return e
		}
	}
	if e = transfer.CheckDAVLocks(r.Context(), tx, d.transfer, key, false); e != nil {
		return e
	}
	n, e := d.lookup(r.Context(), tx, name)
	if e != nil {
		return e
	}
	props := append([]davStoredProperty{}, n.Properties...)
	created, modified := n.Created, n.Modified
	failed := false
	for i, c := range changes {
		p := c.Node
		isTime := (p.Name.Space == "DAV:" && (p.Name.Local == "getlastmodified" || p.Name.Local == "creationdate")) || (p.Name.Space == "urn:schemas-microsoft-com:" && (p.Name.Local == "Win32CreationTime" || p.Name.Local == "Win32LastModifiedTime" || p.Name.Local == "Win32LastAccessTime"))
		if isTime {
			if name == "" {
				changes[i].Status = 403
				failed = true
				continue
			}
			date, err := http.ParseTime(p.text())
			if err != nil {
				date, err = time.Parse(time.RFC3339, p.text())
			}
			if c.Remove || err != nil || len(p.Children) > 0 {
				changes[i].Status = 409
				failed = true
				continue
			}
			if p.Name.Local == "creationdate" || p.Name.Local == "Win32CreationTime" {
				created = date
			} else if p.Name.Local != "Win32LastAccessTime" {
				modified = date
			}
			if p.Name.Space == "DAV:" {
				continue
			}
		} else if p.Name.Space == "DAV:" || p.Name.Space == "" {
			changes[i].Status = 403
			failed = true
			continue
		}
		next := []davStoredProperty{}
		for _, v := range props {
			if v.Space != p.Name.Space || v.Local != p.Name.Local {
				next = append(next, v)
			}
		}
		if !c.Remove {
			next = append(next, p.property())
		}
		props = next
	}
	raw, e := json.Marshal(props)
	if e != nil {
		return e
	}
	if len(raw) > davMaxPropertiesBytes || len(props) > 256 {
		return davError(507)
	}
	if failed {
		for i := range changes {
			if changes[i].Status == 200 {
				changes[i].Status = 424
			}
		}
	} else {
		table := "files"
		id := n.ID
		if n.Directory {
			table = "draft_folders"
			if name == "" {
				table = "transfers"
			} else if n.ID == d.transfer {
				id = domain.NewID()
				if _, e = tx.Exec(r.Context(), "INSERT INTO draft_folders(id,transfer_id,path,path_key) VALUES($1,$2,$3,$4)", id, d.transfer, n.Path, key); e != nil {
					return e
				}
			}
		}
		if table == "transfers" {
			_, e = tx.Exec(r.Context(), "UPDATE transfers SET dav_properties=$2 WHERE id=$1", id, raw)
		} else {
			_, e = tx.Exec(r.Context(), "UPDATE "+table+" SET dav_properties=$2,created_at=$3,modified_at=$4 WHERE id=$1", id, raw, created, modified)
		}
		if e != nil {
			return e
		}
		// The transfer row serializes all property writers. Measure PostgreSQL's
		// JSON representation, whose whitespace can exceed the incoming JSON.
		var propertyBytes int64
		e = tx.QueryRow(r.Context(), `SELECT sum(greatest(octet_length(dav_properties::text)-2,0)) FROM (
		 SELECT dav_properties FROM transfers WHERE id=$1
		 UNION ALL SELECT dav_properties FROM files WHERE transfer_id=$1 AND staged_until IS NULL AND state NOT IN ('cancelled','deleted')
		 UNION ALL SELECT dav_properties FROM draft_folders WHERE transfer_id=$1
		) properties`, d.transfer).Scan(&propertyBytes)
		if e != nil {
			return e
		}
		if propertyBytes > davMaxDraftPropertiesBytes {
			return davError(507)
		}
		if _, e = tx.Exec(r.Context(), "UPDATE transfers SET revision=revision+1 WHERE id=$1", d.transfer); e != nil {
			return e
		}
		if e = tx.Commit(r.Context()); e != nil {
			return e
		}
	}
	stats := []davPropStat{}
	for _, c := range changes {
		stats = append(stats, davPropStat{Properties: []davProperty{davNamedProperty(c.Node.Name)}, Status: "HTTP/1.1 " + httpStatus(c.Status)})
	}
	return davWriteXML(w, davMultistatus{Responses: []davXMLResponse{{Href: davHref(d.transfer, n.Path, n.Directory), Stats: stats}}})
}
func httpStatus(code int) string {
	switch code {
	case 200:
		return "200 OK"
	case 403:
		return "403 Forbidden"
	case 409:
		return "409 Conflict"
	default:
		return "424 Failed Dependency"
	}
}
