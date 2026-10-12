package httpapi

import (
	"context"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/transfer"
	"github.com/jackc/pgx/v5"
	"net/http"
	"sort"
	"strings"
)

type pathMove struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

func (a *App) treeRoutes(m *http.ServeMux) {
	m.HandleFunc("PATCH /api/v1/transfers/{id}/files", a.protect("transfers:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Moves []pathMove `json:"moves"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if len(v.Moves) == 0 || len(v.Moves) > 1000 {
			return auth.ErrInvalid
		}
		return a.repath(w, r, p, c, v.Moves, "", false)
	}))
	m.HandleFunc("PATCH /api/v1/transfers/{id}/folders/{folder}", a.protect("transfers:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Path string `json:"path"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		return a.repath(w, r, p, c, nil, v.Path, false)
	}))
	m.HandleFunc("DELETE /api/v1/transfers/{id}/folders/{folder}", a.protect("transfers:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		return a.repath(w, r, p, c, nil, "", true)
	}))
}
func (a *App) repath(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings, moves []pathMove, folderPath string, remove bool) error {
	return a.repathChecked(w, r, p, c, moves, folderPath, remove, "", nil, 204)
}

func (a *App) repathChecked(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings, moves []pathMove, folderPath string, remove bool, prefix string, check func(pgx.Tx) error, status int) error {
	ctx := r.Context()
	tx, e := a.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var parent string
	if e = tx.QueryRow(ctx, "SELECT id::text FROM transfers WHERE id::text=$1 AND owner_id=$2 AND state='draft' AND NOT frozen FOR UPDATE", r.PathValue("id"), p.ID).Scan(&parent); e != nil {
		return e
	}
	if check != nil {
		if e = check(tx); e != nil {
			return e
		}
	}
	folderID := r.PathValue("folder")
	if folderID != "" {
		if e = tx.QueryRow(ctx, "SELECT path FROM draft_folders WHERE id::text=$1 AND transfer_id=$2", folderID, parent).Scan(&prefix); e != nil {
			return e
		}
	}
	prefixKey := ""
	prefixSegments := 0
	if prefix != "" {
		prefixKey, e = domain.PathKey(prefix, c.MaxPathBytes)
		if e != nil {
			return e
		}
		prefixSegments = len(strings.Split(prefix, "/"))
		if !remove {
			var targetKey string
			if targetKey, e = domain.PathKey(folderPath, c.MaxPathBytes); e != nil {
				return e
			}
			if strings.HasPrefix(targetKey, prefixKey+"/") {
				return auth.ErrInvalid
			}
		}
	}
	if prefix != "" {
		if e = transfer.CheckDAVMembership(ctx, tx, parent, prefixKey); e != nil {
			return e
		}
		if e = transfer.CheckDAVLocks(ctx, tx, parent, prefixKey, true); e != nil {
			return e
		}
	}
	if folderPath != "" {
		key, err := domain.PathKey(folderPath, c.MaxPathBytes)
		if err != nil {
			return err
		}
		if e = transfer.CheckDAVLocks(ctx, tx, parent, key, true); e != nil {
			return e
		}
	}
	type item struct{ id, path, key, table string }
	items := []item{}
	for _, table := range []string{"files", "draft_folders"} {
		predicate := ""
		if table == "files" {
			predicate = " AND staged_until IS NULL AND state NOT IN ('cancelled','deleted')"
		}
		rows, e := tx.Query(ctx, "SELECT id::text,path,path_key FROM "+table+" WHERE transfer_id=$1"+predicate, parent)
		if e != nil {
			return e
		}
		for rows.Next() {
			v := item{table: table}
			if e = rows.Scan(&v.id, &v.path, &v.key); e != nil {
				rows.Close()
				return e
			}
			items = append(items, v)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
	}
	replacements := map[string]string{}
	for _, v := range moves {
		if _, ok := replacements[v.ID]; ok {
			return auth.ErrInvalid
		}
		replacements[v.ID] = v.Path
	}
	changed := map[string]bool{}
	keys := map[string]string{}
	fileKeys := []string{}
	for i := range items {
		v := &items[i]
		target := v.path
		if prefixKey != "" && (v.key == prefixKey || strings.HasPrefix(v.key, prefixKey+"/")) {
			if remove {
				changed[v.id] = true
				continue
			}
			// Match the canonical namespace, then retain the original suffix. Unicode
			// case folding can change byte length, so byte-prefix slicing is unsafe.
			target = folderPath
			if suffix := strings.Join(strings.Split(v.path, "/")[prefixSegments:], "/"); suffix != "" {
				target += "/" + suffix
			}
		}
		if value, ok := replacements[v.id]; ok && v.table == "files" {
			target = value
			delete(replacements, v.id)
		}
		key, e := domain.PathKey(target, c.MaxPathBytes)
		if e != nil {
			return e
		}
		if previous, ok := keys[key]; ok && previous != v.id {
			return transfer.ErrConflict
		}
		keys[key] = v.id
		if target != v.path {
			changed[v.id] = true
		}
		if changed[v.id] {
			if e = transfer.CheckDAVMembership(ctx, tx, parent, v.key); e != nil {
				return e
			}
			if e = transfer.CheckDAVMembership(ctx, tx, parent, key); e != nil {
				return e
			}
			if e = transfer.CheckDAVLocks(ctx, tx, parent, v.key, true); e != nil {
				return e
			}
			if e = transfer.CheckDAVLocks(ctx, tx, parent, key, true); e != nil {
				return e
			}
		}
		v.path = target
		v.key = key
		if v.table == "files" {
			fileKeys = append(fileKeys, key)
		}
	}
	if len(replacements) > 0 {
		return transfer.ErrNotFound
	}
	// A file can never be the ancestor of another file or virtual directory.
	sort.Strings(fileKeys)
	allKeys := make([]string, 0, len(keys))
	for key := range keys {
		allKeys = append(allKeys, key)
	}
	sort.Strings(allKeys)
	for _, key := range fileKeys {
		index := sort.SearchStrings(allKeys, key+"/")
		if index < len(allKeys) && strings.HasPrefix(allKeys[index], key+"/") {
			return transfer.ErrConflict
		}
	}
	// Free old unique keys before swaps; no intermediate paths escape the transaction.
	for _, v := range items {
		if changed[v.id] && !remove {
			if _, e = tx.Exec(ctx, "UPDATE "+v.table+" SET path_key=$2 WHERE id=$1", v.id, "#move:"+v.id); e != nil {
				return e
			}
		}
	}
	for _, v := range items {
		if !changed[v.id] {
			continue
		}
		if v.table == "files" {
			var frozen bool
			if e = tx.QueryRow(ctx, "SELECT frozen FROM files WHERE id=$1 FOR UPDATE", v.id).Scan(&frozen); e != nil {
				return e
			}
			if frozen {
				return transfer.ErrConflict
			}
		}
		if remove {
			if v.table == "files" {
				if _, e = tx.Exec(ctx, "UPDATE files SET state='cancelled',upload_epoch=upload_epoch+1,path_key='#cancelled:'||id::text WHERE id=$1", v.id); e != nil {
					return e
				}
				if _, e = tx.Exec(ctx, "INSERT INTO deletion_jobs(provider_id,object_key,file_id) SELECT f.provider_id,ch.object_key,f.id FROM files f JOIN file_chunks ch ON ch.file_id=f.id WHERE f.id=$1 ON CONFLICT(object_key) DO NOTHING", v.id); e != nil {
					return e
				}
			} else {
				if _, e = tx.Exec(ctx, "DELETE FROM draft_folders WHERE id=$1", v.id); e != nil {
					return e
				}
			}
		} else {
			if _, e = tx.Exec(ctx, "UPDATE "+v.table+" SET path=$2,path_key=$3 WHERE id=$1", v.id, v.path, v.key); e != nil {
				return e
			}
		}
	}
	if _, e = tx.Exec(ctx, "UPDATE transfers SET revision=revision+1 WHERE id=$1", parent); e != nil {
		return e
	}
	if e = tx.Commit(ctx); e != nil {
		return e
	}
	w.WriteHeader(status)
	return nil
}
