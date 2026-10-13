package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/transfer"
	"fmt"
	"github.com/jackc/pgx/v5"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"
)

const davMaxNodes = 10000
const davMaxPropertiesBytes = 32768
const davMaxDraftPropertiesBytes = 4 << 20

type davQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type davNode struct {
	ID, Path, State, Media string
	Transfer               string
	Size, Offset, Revision int64
	SHA                    *string
	Directory              bool
	Created, Modified      time.Time
	Properties             []davStoredProperty
	Locks                  []transfer.DAVLock
}

func (n davNode) contentType() string {
	if n.Directory {
		return "httpd/unix-directory"
	}
	if n.Media == "" {
		return "application/octet-stream"
	}
	return n.Media
}
func (n davNode) etag() string {
	if n.Directory {
		return fmt.Sprintf(`W/"%s:%d"`, n.ID, n.Revision)
	}
	return davFileETag(n.ID, n.State, n.Offset, n.SHA)
}

func (d davAccess) nodes(ctx context.Context, q davQuerier) (map[string]davNode, error) {
	var revision int64
	var created time.Time
	var rootProps []byte
	if e := q.QueryRow(ctx, "SELECT revision,created_at,dav_properties FROM transfers WHERE id=$1 AND owner_id=$2 AND state='draft' AND NOT frozen", d.transfer, d.owner).Scan(&revision, &created, &rootProps); e != nil {
		return nil, e
	}
	root := davNode{ID: d.transfer, Directory: true, Revision: revision, Created: created, Modified: created}
	if e := json.Unmarshal(rootProps, &root.Properties); e != nil {
		return nil, e
	}
	nodes := map[string]davNode{"": root}
	// Bound aggregate metadata before decoding. A per-resource limit alone lets
	// a large collection amplify a tiny PROPFIND into hundreds of megabytes.
	propertyBytes := max(0, len(rootProps)-2)
	rows, e := q.Query(ctx, `SELECT id::text,path,size,upload_offset,state,sha256,COALESCE(content_type,''),false,created_at,modified_at,dav_properties FROM files WHERE transfer_id=$1 AND staged_until IS NULL AND state NOT IN ('cancelled','deleted') UNION ALL SELECT id::text,path,0,0,'',NULL,'',true,created_at,modified_at,dav_properties FROM draft_folders WHERE transfer_id=$1 LIMIT 10001`, d.transfer)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		count++
		if count > davMaxNodes {
			return nil, davError(507)
		}
		n := davNode{Revision: revision}
		var props []byte
		if e = rows.Scan(&n.ID, &n.Path, &n.Size, &n.Offset, &n.State, &n.SHA, &n.Media, &n.Directory, &n.Created, &n.Modified, &props); e != nil {
			return nil, e
		}
		propertyBytes += max(0, len(props)-2)
		if propertyBytes > davMaxDraftPropertiesBytes {
			return nil, davError(507)
		}
		key, e := domain.PathKey(n.Path, d.settings.MaxPathBytes)
		if e != nil {
			return nil, e
		}
		if e = json.Unmarshal(props, &n.Properties); e != nil {
			return nil, e
		}
		nodes[key] = n
		if len(nodes) > davMaxNodes {
			return nil, davError(507)
		}
		for ancestor := path.Dir(n.Path); ancestor != "."; ancestor = path.Dir(ancestor) {
			key, e := domain.PathKey(ancestor, d.settings.MaxPathBytes)
			if e != nil {
				return nil, e
			}
			if _, ok := nodes[key]; !ok {
				dir := root
				dir.Path = ancestor
				dir.Properties = nil
				nodes[key] = dir
			}
			if len(nodes) > davMaxNodes {
				return nil, davError(507)
			}
		}
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	return nodes, nil
}

func (d davAccess) lookup(ctx context.Context, q davQuerier, name string) (davNode, error) {
	nodes, e := d.nodes(ctx, q)
	if e != nil {
		return davNode{}, e
	}
	key := ""
	if name != "" {
		key, e = domain.PathKey(name, d.settings.MaxPathBytes)
		if e != nil {
			return davNode{}, e
		}
	}
	n, ok := nodes[key]
	if !ok {
		return davNode{}, davError(404)
	}
	return n, nil
}

func (d davAccess) parentExists(ctx context.Context, q davQuerier, name string) error {
	return d.parentExistsExcept(ctx, q, name, "")
}

func (d davAccess) parentExistsExcept(ctx context.Context, q davQuerier, name, exclude string) error {
	parent := path.Dir(name)
	if parent == "." {
		return nil
	}
	key, e := domain.PathKey(parent, d.settings.MaxPathBytes)
	if e != nil {
		return e
	}
	var exists bool
	e = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM draft_folders WHERE transfer_id=$1 AND (path_key=$2 OR starts_with(path_key,$2||'/'))) OR EXISTS(SELECT 1 FROM files WHERE transfer_id=$1 AND id::text<>$3 AND starts_with(path_key,$2||'/') AND state NOT IN ('cancelled','deleted'))`, d.transfer, key, exclude).Scan(&exists)
	if e != nil {
		return e
	}
	if !exists {
		return davError(409)
	}
	return nil
}

func (d davAccess) lock(ctx context.Context, tx pgx.Tx) error {
	var id string
	if e := tx.QueryRow(ctx, "SELECT id::text FROM transfers WHERE id=$1 AND owner_id=$2 FOR UPDATE", d.transfer, d.owner).Scan(&id); e != nil {
		return e
	}
	return d.guard(ctx, tx)
}

func (d davAccess) mkdir(w http.ResponseWriter, r *http.Request, name string) error {
	if name == "" {
		return davError(405)
	}
	var b [1]byte
	if n, e := r.Body.Read(b[:]); n != 0 || e != io.EOF {
		return davError(415)
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
	keyCheck, e := domain.PathKey(name, d.settings.MaxPathBytes)
	if e != nil {
		return e
	}
	if e = transfer.CheckDAVMembership(r.Context(), tx, d.transfer, keyCheck); e != nil {
		return e
	}
	if e = transfer.CheckDAVLocks(r.Context(), tx, d.transfer, keyCheck, false); e != nil {
		return e
	}
	if _, e = d.lookup(r.Context(), tx, name); e == nil {
		return davError(405)
	} else if !errors.Is(e, davError(404)) {
		return e
	}
	if e = d.parentExists(r.Context(), tx, name); e != nil {
		return e
	}
	key, e := domain.PathKey(name, d.settings.MaxPathBytes)
	if e != nil {
		return e
	}
	// Bound explicit directories as well as derived parents, before insertion.
	nodes, e := d.nodes(r.Context(), tx)
	if e != nil {
		return e
	}
	if len(nodes) >= davMaxNodes {
		return davError(507)
	}
	if _, e = tx.Exec(r.Context(), "INSERT INTO draft_folders(id,transfer_id,path,path_key) VALUES($1,$2,$3,$4)", domain.NewID(), d.transfer, name, key); e != nil {
		return e
	}
	if _, e = tx.Exec(r.Context(), "UPDATE transfers SET revision=revision+1 WHERE id=$1", d.transfer); e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	w.WriteHeader(201)
	return nil
}

func (d davAccess) destination(r *http.Request) (string, error) {
	raw := r.Header.Get("Destination")
	if raw == "" {
		return "", davError(400)
	}
	u, e := url.Parse(raw)
	if e != nil {
		return "", davError(400)
	}
	if u.IsAbs() || u.Host != "" {
		origin, e := url.Parse(d.settings.PublicOrigin)
		if e != nil {
			return "", e
		}
		if u.Scheme != origin.Scheme || !strings.EqualFold(u.Host, origin.Host) {
			return "", davError(403)
		}
	}
	id, name, e := davPath(u, d.settings.MaxPathBytes)
	if e != nil {
		return "", e
	}
	if id != d.transfer {
		return "", davError(403)
	}
	if name == "" {
		return "", davError(403)
	}
	return name, nil
}

func (d davAccess) change(w http.ResponseWriter, r *http.Request, name string, principal auth.Principal) error {
	if name == "" {
		return davError(403)
	}
	if depth := r.Header.Get("Depth"); depth != "" && depth != "infinity" {
		return davError(400)
	}
	if value := r.Header.Get("Overwrite"); value != "" && value != "F" && value != "T" {
		return davError(400)
	}
	node, e := d.lookup(r.Context(), d.app.DB, name)
	if e != nil {
		return e
	}
	destination := ""
	if r.Method == "MOVE" {
		destination, e = d.destination(r)
		if e != nil {
			return e
		}
		from, err := domain.PathKey(node.Path, d.settings.MaxPathBytes)
		if err != nil {
			return err
		}
		to, err := domain.PathKey(destination, d.settings.MaxPathBytes)
		if err != nil {
			return err
		}
		if node.Directory && strings.HasPrefix(to, from+"/") {
			return davError(403)
		}
	}
	status := 201
	if r.Method == "DELETE" {
		status = 204
	}
	check := func(tx pgx.Tx) error {
		if e := d.guard(r.Context(), tx); e != nil {
			return e
		}
		if e := d.conditions(r.Context(), tx, r, name); e != nil {
			return e
		}
		from, _ := domain.PathKey(name, d.settings.MaxPathBytes)
		if e := transfer.CheckDAVMembership(r.Context(), tx, d.transfer, from); e != nil {
			return e
		}
		if e := transfer.CheckDAVLocks(r.Context(), tx, d.transfer, from, true); e != nil {
			return e
		}
		if destination != "" {
			to, _ := domain.PathKey(destination, d.settings.MaxPathBytes)
			if e := transfer.CheckDAVMembership(r.Context(), tx, d.transfer, to); e != nil {
				return e
			}
			if e := transfer.CheckDAVLocks(r.Context(), tx, d.transfer, to, true); e != nil {
				return e
			}
		}
		current, e := d.lookup(r.Context(), tx, name)
		if e != nil {
			return e
		}
		if current.ID != node.ID || current.Path != node.Path || current.Directory != node.Directory {
			return davError(409)
		}
		if destination != "" {
			if target, err := d.lookup(r.Context(), tx, destination); err == nil {
				if r.Header.Get("Overwrite") == "F" {
					return davError(412)
				}
				if target.Directory || node.Directory || target.ID == node.ID {
					return davError(409)
				}
				if e = d.cancelInTx(r.Context(), tx, target.ID); e != nil {
					return e
				}
				status = 204
			} else if !errors.Is(err, davError(404)) {
				return err
			}
			if e = d.parentExists(r.Context(), tx, destination); e != nil {
				return e
			}
		}
		// RFC 4918: locks rooted on the moved/deleted source do not follow it.
		_, e = tx.Exec(r.Context(), "DELETE FROM dav_locks WHERE transfer_id=$1 AND (path_key=$2 OR starts_with(path_key,$2||'/'))", d.transfer, from)
		return e
	}
	if !node.Directory && r.Method == "DELETE" {
		service := d.app.Files
		service.WriteGuard = func(_ context.Context, tx pgx.Tx) error { return check(tx) }
		if e = service.Cancel(r.Context(), d.owner, node.ID); e != nil {
			return e
		}
		w.WriteHeader(204)
		return nil
	}
	moves := []pathMove(nil)
	prefix := ""
	if node.Directory {
		prefix = node.Path
	} else {
		moves = []pathMove{{ID: node.ID, Path: destination}}
	}
	return d.app.repathChecked(davChangeResponse{w, &status}, r, principal, d.settings, moves, destination, r.Method == "DELETE", prefix, check, status)
}

func sortedDAVNodes(nodes map[string]davNode) []davNode {
	out := make([]davNode, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

type davChangeResponse struct {
	http.ResponseWriter
	status *int
}

func (w davChangeResponse) WriteHeader(status int) {
	if status == 201 || status == 204 {
		status = *w.status
	}
	w.ResponseWriter.WriteHeader(status)
}
