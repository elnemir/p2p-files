package httpapi

import (
	"bytes"
	"encoding/xml"
	"filex.local/filex/internal/branding"
	"filex.local/filex/internal/transfer"
	"io"
	"net/http"
	"path"
	"strconv"
	"time"

	"filex.local/filex/internal/domain"
)

type davXMLNode struct {
	Name     xml.Name
	Children []*davXMLNode
	Attr     []xml.Attr
	Content  []any
}

// Parse bounded XML without a DTD/entity resolver or unbounded recursion.
func davXML(w http.ResponseWriter, r *http.Request) (*davXMLNode, error) {
	raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 65536))
	if e != nil {
		return nil, davError(413)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	d := xml.NewDecoder(bytes.NewReader(raw))
	var root *davXMLNode
	stack := []*davXMLNode{}
	for {
		token, e := d.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, davError(400)
		}
		switch t := token.(type) {
		case xml.StartElement:
			if len(stack) >= 16 {
				return nil, davError(400)
			}
			n := &davXMLNode{Name: t.Name}
			for _, a := range t.Attr {
				if a.Name.Space != "xmlns" && a.Name.Local != "xmlns" {
					n.Attr = append(n.Attr, a)
				}
			}
			if len(stack) == 0 {
				if root != nil {
					return nil, davError(400)
				}
				root = n
			} else {
				p := stack[len(stack)-1]
				p.Children = append(p.Children, n)
				p.Content = append(p.Content, n)
			}
			stack = append(stack, n)
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, davError(400)
			}
			stack = stack[:len(stack)-1]
		case xml.Directive:
			return nil, davError(400)
		case xml.ProcInst:
			if t.Target != "xml" || root != nil {
				return nil, davError(400)
			}
		case xml.CharData:
			if len(stack) > 0 {
				n := stack[len(stack)-1]
				n.Content = append(n.Content, string(t))
			}
			if len(stack) == 0 && len(bytes.TrimSpace(t)) != 0 {
				return nil, davError(400)
			}
		}
	}
	if root == nil || len(stack) != 0 {
		return nil, davError(400)
	}
	return root, nil
}

var davLiveProperties = []string{"displayname", "resourcetype", "getcontentlength", "getcontenttype", "getetag", "supportedlock", "lockdiscovery", "creationdate", "getlastmodified"}

type davProperty struct {
	XMLName    xml.Name
	Namespace  *string    `xml:"xmlns,attr,omitempty"`
	Value      string     `xml:",chardata"`
	Inner      string     `xml:",innerxml"`
	Attr       []xml.Attr `xml:",any,attr"`
	Collection *struct{}  `xml:"DAV: collection,omitempty"`
}
type davPropStat struct {
	Properties []davProperty `xml:"DAV: prop>property"`
	Status     string        `xml:"DAV: status"`
}
type davXMLResponse struct {
	XMLName xml.Name      `xml:"DAV: response"`
	Href    string        `xml:"DAV: href"`
	Stats   []davPropStat `xml:"DAV: propstat"`
}
type davMultistatus struct {
	XMLName   xml.Name         `xml:"DAV: multistatus"`
	Responses []davXMLResponse `xml:"DAV: response"`
}

func davPropertyValue(name xml.Name, n davNode, namesOnly bool) (davProperty, bool) {
	p := davNamedProperty(name)
	for _, stored := range n.Properties {
		if stored.Space == name.Space && stored.Local == name.Local {
			if !namesOnly {
				p.Inner = stored.Inner
				p.Attr = stored.Attr
			}
			return p, true
		}
	}
	if name.Space != "DAV:" {
		return p, false
	}
	switch name.Local {
	case "displayname":
		p.Value = path.Base(n.Path)
		if n.Path == "" {
			p.Value = "Files"
		}
	case "resourcetype":
		if n.Directory {
			p.Collection = &struct{}{}
		}
	case "getcontentlength":
		p.Value = strconv.FormatInt(n.Size, 10)
	case "getcontenttype":
		p.Value = n.contentType()
	case "getetag":
		p.Value = n.etag()
	case "creationdate":
		p.Value = n.Created.UTC().Format(time.RFC3339)
	case "getlastmodified":
		p.Value = n.Modified.UTC().Format(http.TimeFormat)
	case "supportedlock":
		p.Inner = davSupportedLock
	case "lockdiscovery":
		for _, l := range n.Locks {
			p.Inner += davLockXML(l, n.Transfer)
		}
	default:
		return p, false
	}
	if namesOnly {
		p.Value = ""
		p.Collection = nil
		p.Inner = ""
	}
	return p, true
}

func davNamedProperty(name xml.Name) davProperty {
	p := davProperty{XMLName: name}
	// Unqualified requested properties must not inherit DAV: from their parent.
	if name.Space == "" {
		empty := ""
		p.Namespace = &empty
	}
	return p
}

func davRequested(root *davXMLNode) ([]xml.Name, bool, error) {
	namesOnly := false
	if root != nil {
		if root.Name != (xml.Name{Space: "DAV:", Local: "propfind"}) || len(root.Children) != 1 {
			return nil, false, davError(400)
		}
		selection := root.Children[0]
		if selection.Name.Space != "DAV:" {
			return nil, false, davError(400)
		}
		switch selection.Name.Local {
		case "prop":
			if len(selection.Children) == 0 || len(selection.Children) > 256 {
				return nil, false, davError(400)
			}
			props := []xml.Name{}
			for _, n := range selection.Children {
				props = append(props, n.Name)
			}
			return props, false, nil
		case "propname":
			namesOnly = true
		case "allprop":
		default:
			return nil, false, davError(400)
		}
	}
	props := []xml.Name{}
	for _, n := range davLiveProperties {
		props = append(props, xml.Name{Space: "DAV:", Local: n})
	}
	return props, namesOnly, nil
}

const davMaxResponseBytes = 8 << 20

type davXMLBuffer struct{ bytes.Buffer }

func (b *davXMLBuffer) Write(p []byte) (int, error) {
	if len(p) > davMaxResponseBytes-b.Len() {
		return 0, davError(507)
	}
	return b.Buffer.Write(p)
}

// Buffer a bounded document before sending 207, so an amplified property query
// fails atomically instead of allocating the entire tree's repeated properties.
func davEncodeXML(w http.ResponseWriter, responses func(*xml.Encoder) error) error {
	b := &davXMLBuffer{}
	if _, e := b.Write([]byte(xml.Header)); e != nil {
		return e
	}
	encoder := xml.NewEncoder(b)
	start := xml.StartElement{Name: xml.Name{Space: "DAV:", Local: "multistatus"}}
	if e := encoder.EncodeToken(start); e != nil {
		return e
	}
	if e := responses(encoder); e != nil {
		return e
	}
	if e := encoder.EncodeToken(start.End()); e != nil {
		return e
	}
	if e := encoder.Flush(); e != nil {
		return e
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(207)
	_, e := w.Write(b.Bytes())
	return e
}

func davWriteXML(w http.ResponseWriter, value davMultistatus) error {
	return davEncodeXML(w, func(encoder *xml.Encoder) error {
		for _, response := range value.Responses {
			if e := encoder.Encode(response); e != nil {
				return e
			}
		}
		return nil
	})
}

func (d davAccess) propfind(w http.ResponseWriter, r *http.Request, name string) error {
	depth := r.Header.Get("Depth")
	if depth == "" || depth == "infinity" {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(403)
		_, e := io.WriteString(w, `<D:error xmlns:D="DAV:"><D:propfind-finite-depth/></D:error>`)
		return e
	}
	if depth != "0" && depth != "1" {
		return davError(400)
	}
	root, e := davXML(w, r)
	if e != nil {
		return e
	}
	props, namesOnly, e := davRequested(root)
	if e != nil {
		return e
	}
	nodes, e := d.nodes(r.Context(), d.app.DB)
	if e != nil {
		return e
	}
	key := ""
	if name != "" {
		key, e = domain.PathKey(name, d.settings.MaxPathBytes)
		if e != nil {
			return e
		}
	}
	n, ok := nodes[key]
	if !ok {
		return davError(404)
	}
	selected := map[string]davNode{key: n}
	if depth == "1" && n.Directory {
		for k, item := range nodes {
			if k == key || item.Path == "" {
				continue
			}
			parent := path.Dir(k)
			if parent == "." {
				parent = ""
			}
			if parent == key {
				selected[k] = item
			}
		}
	}
	locks, e := transfer.DAVLocks(r.Context(), d.app.DB, d.transfer)
	if e != nil {
		return e
	}
	brand, e := branding.Current(r.Context(), d.app.DB)
	if e != nil {
		return e
	}
	all := root == nil || len(root.Children) == 1 && root.Children[0].Name.Local != "prop"
	return davEncodeXML(w, func(encoder *xml.Encoder) error {
		for _, n := range sortedDAVNodes(selected) {
			n.Transfer = d.transfer
			key := ""
			if n.Path != "" {
				key, _ = domain.PathKey(n.Path, d.settings.MaxPathBytes)
			}
			for _, l := range locks {
				if transfer.DAVCovers(l.Key, key, l.Recursive) {
					n.Locks = append(n.Locks, l)
				}
			}
			requested := append([]xml.Name{}, props...)
			if all {
				for _, p := range n.Properties {
					requested = append(requested, xml.Name{Space: p.Space, Local: p.Local})
				}
			}
			response := davXMLResponse{Href: davHref(d.transfer, n.Path, n.Directory)}
			found, missing := []davProperty{}, []davProperty{}
			for _, name := range requested {
				p, ok := davPropertyValue(name, n, namesOnly)
				if !namesOnly && n.Path == "" && name.Space == "DAV:" && name.Local == "displayname" {
					p.Value = brand.Value.ProductName
				}
				if ok {
					found = append(found, p)
				} else {
					missing = append(missing, p)
				}
			}
			if len(found) > 0 {
				response.Stats = append(response.Stats, davPropStat{Properties: found, Status: "HTTP/1.1 200 OK"})
			}
			if len(missing) > 0 {
				response.Stats = append(response.Stats, davPropStat{Properties: missing, Status: "HTTP/1.1 404 Not Found"})
			}
			if e := encoder.Encode(response); e != nil {
				return e
			}
		}
		return nil
	})
}
