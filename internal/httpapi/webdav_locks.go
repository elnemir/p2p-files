package httpapi

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/transfer"
)

const davSupportedLock = `<lockentry xmlns="DAV:"><lockscope><exclusive/></lockscope><locktype><write/></locktype></lockentry>`
const davMaxLocks = 256
const davMaxLockOwnerBytes = 4096

func davLockXML(l transfer.DAVLock, id string) string {
	type href struct {
		Href string `xml:"DAV: href"`
	}
	type active struct {
		XMLName xml.Name `xml:"DAV: activelock"`
		Scope   string   `xml:",innerxml"`
		Depth   string   `xml:"DAV: depth"`
		Timeout string   `xml:"DAV: timeout"`
		Token   href     `xml:"DAV: locktoken"`
		Root    href     `xml:"DAV: lockroot"`
	}
	depth := "0"
	if l.Recursive {
		depth = "infinity"
	}
	a := active{Scope: `<lockscope xmlns="DAV:"><exclusive/></lockscope><locktype xmlns="DAV:"><write/></locktype>` + l.OwnerXML, Depth: depth, Timeout: "Second-" + strconv.Itoa(max(1, int(time.Until(l.Expires).Seconds()))), Token: href{"urn:uuid:" + l.Token}, Root: href{davHref(id, l.Path, false)}}
	b, e := xml.Marshal(a)
	if e != nil {
		return ""
	}
	return string(b)
}
func davTimeout(value string) (int, error) {
	if value == "" {
		return 3600, nil
	}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "Infinite" {
			return 3600, nil
		}
		if strings.HasPrefix(part, "Second-") {
			n, e := strconv.ParseUint(strings.TrimPrefix(part, "Second-"), 10, 32)
			if e == nil && n > 0 {
				if n > 3600 {
					return 3600, nil
				}
				return int(n), nil
			}
		}
	}
	return 0, davError(400)
}

func (d davAccess) lockResource(w http.ResponseWriter, r *http.Request, name string) error {
	seconds, e := davTimeout(r.Header.Get("Timeout"))
	if e != nil {
		return e
	}
	root, e := davXML(w, r)
	if e != nil {
		return e
	}
	recursive := true
	if v := r.Header.Get("Depth"); v != "" {
		if v != "0" && v != "infinity" {
			return davError(400)
		}
		recursive = v == "infinity"
	}
	owner := ""
	if root != nil {
		if root.Name != (xml.Name{Space: "DAV:", Local: "lockinfo"}) {
			return davError(400)
		}
		scope, kind := false, false
		for _, c := range root.Children {
			switch c.Name {
			case xml.Name{Space: "DAV:", Local: "lockscope"}:
				scope = len(c.Children) == 1 && c.Children[0].Name == (xml.Name{Space: "DAV:", Local: "exclusive"})
			case xml.Name{Space: "DAV:", Local: "locktype"}:
				kind = len(c.Children) == 1 && c.Children[0].Name == (xml.Name{Space: "DAV:", Local: "write"})
			case xml.Name{Space: "DAV:", Local: "owner"}:
				owner = c.outerXML()
			default:
				return davError(400)
			}
		}
		if !scope || !kind {
			return davError(400)
		}
		if len(owner) > davMaxLockOwnerBytes {
			return davError(413)
		}
	}
	key := ""
	if name != "" {
		key, e = domain.PathKey(name, d.settings.MaxPathBytes)
		if e != nil {
			return e
		}
	}
	status := 200
	if root != nil {
		if _, e = d.lookup(r.Context(), d.app.DB, name); errors.Is(e, davError(404)) {
			// An unmapped URL becomes an ordinary validated zero-byte resource.
			// If-None-Match protects a concurrent creator; it is never overwritten here.
			create := r.Clone(r.Context())
			create.Method = "PUT"
			create.Body = io.NopCloser(bytes.NewReader(nil))
			create.ContentLength = 0
			// Preserve the parent's lock conditions when creating a new member.
			create.Header.Del("X-Expected-Entity-Length")
			create.Header.Set("If-None-Match", "*")
			if e = d.put(&davDiscardResponse{header: http.Header{}}, create, name); e != nil {
				return e
			}
			status = 201
		} else if e != nil {
			return e
		}
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
	locks, e := transfer.DAVLocks(r.Context(), tx, d.transfer)
	if e != nil {
		return e
	}
	l := transfer.DAVLock{Token: domain.NewID(), Credential: d.credential, Path: name, Key: key, Recursive: recursive, OwnerXML: owner}
	if root == nil {
		found := false
		for _, active := range locks {
			if active.Key == key && active.Credential == d.credential && d.tokens[active.Token] {
				l = active
				found = true
				break
			}
		}
		if !found {
			return davError(412)
		}
		e = tx.QueryRow(r.Context(), "UPDATE dav_locks SET expires_at=clock_timestamp()+make_interval(secs=>$2) WHERE token=$1 RETURNING expires_at", l.Token, seconds).Scan(&l.Expires)
	} else {
		for _, active := range locks {
			if transfer.DAVCovers(active.Key, key, active.Recursive) || recursive && transfer.DAVCovers(key, active.Key, true) {
				return transfer.ErrLocked
			}
		}
		if len(locks) >= davMaxLocks {
			return davError(507)
		}
		// Expired rows carry no authoritative state and need not grow forever.
		if _, e = tx.Exec(r.Context(), "DELETE FROM dav_locks WHERE transfer_id=$1 AND expires_at<=clock_timestamp()", d.transfer); e != nil {
			return e
		}
		e = tx.QueryRow(r.Context(), "INSERT INTO dav_locks(token,transfer_id,credential_id,path,path_key,recursive,owner_xml,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,clock_timestamp()+make_interval(secs=>$8)) RETURNING expires_at", l.Token, d.transfer, d.credential, name, key, recursive, owner, seconds).Scan(&l.Expires)
	}
	if e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Lock-Token", "<urn:uuid:"+l.Token+">")
	w.WriteHeader(status)
	_, e = io.WriteString(w, xml.Header+`<prop xmlns="DAV:"><lockdiscovery>`+davLockXML(l, d.transfer)+`</lockdiscovery></prop>`)
	return e
}

func (d davAccess) unlock(w http.ResponseWriter, r *http.Request, name string) error {
	token := r.Header.Get("Lock-Token")
	if !strings.HasPrefix(token, "<urn:uuid:") || !strings.HasSuffix(token, ">") {
		return davError(400)
	}
	token = strings.TrimSuffix(strings.TrimPrefix(token, "<urn:uuid:"), ">")
	if !domain.ValidID(token) {
		return davError(400)
	}
	key := ""
	var e error
	if name != "" {
		key, e = domain.PathKey(name, d.settings.MaxPathBytes)
		if e != nil {
			return e
		}
	}
	tx, e := d.app.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(r.Context())
	if e = d.lock(r.Context(), tx); e != nil {
		return e
	}
	tag, e := tx.Exec(r.Context(), "DELETE FROM dav_locks WHERE token=$1 AND transfer_id=$2 AND credential_id=$3 AND path_key=$4 AND expires_at>clock_timestamp()", token, d.transfer, d.credential, key)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return davError(409)
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	w.WriteHeader(204)
	return nil
}

type davDiscardResponse struct{ header http.Header }

func (w *davDiscardResponse) Header() http.Header         { return w.header }
func (w *davDiscardResponse) WriteHeader(int)             {}
func (w *davDiscardResponse) Write(p []byte) (int, error) { return len(p), nil }
