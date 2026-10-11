package httpapi

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/reports"
	"filex.local/filex/internal/transfer"
	"github.com/skip2/go-qrcode"
	"io"
	"net/http"
	"time"
)

// The revision check closes the race between rendering and link rotation.
func (a *App) qrBundle(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	id := r.PathValue("id")
	var revision int64
	if e := a.DB.QueryRow(r.Context(), "SELECT links_revision FROM transfers WHERE id::text=$1 AND owner_id=$2 AND state='active' AND absolute_expires_at>clock_timestamp() AND (frozen OR expires_at>clock_timestamp())", id, p.ID).Scan(&revision); e != nil {
		return e
	}
	links, e := a.ownerLinks(r, id, p.ID, c)
	if e != nil {
		return e
	}
	if len(links) == 0 {
		return transfer.ErrNotFound
	}
	artifact, e := a.artifactStore().Create(r.Context(), p.ID, "qr-links.zip", "application/zip", func(w io.Writer) error {
		z := zip.NewWriter(w)
		for _, link := range links {
			png, e := qrcode.Encode(link["url"].(string), qrcode.Medium, 512)
			if e != nil {
				return e
			}
			part, e := z.Create(link["id"].(string) + ".png")
			if e != nil {
				return e
			}
			if _, e = part.Write(png); e != nil {
				return e
			}
		}
		mapping, e := z.Create("recipients.csv")
		if e != nil {
			return e
		}
		csvWriter := csv.NewWriter(mapping)
		csvWriter.UseCRLF = true
		if e = csvWriter.Write([]string{"linkId", "email", "url", "qrFile"}); e != nil {
			return e
		}
		for _, link := range links {
			email := ""
			if v, ok := link["email"].(*string); ok && v != nil {
				email = *v
			}
			lid := link["id"].(string)
			if e = csvWriter.Write([]string{lid, reports.SafeCSV(email), link["url"].(string), lid + ".png"}); e != nil {
				return e
			}
		}
		csvWriter.Flush()
		if e = csvWriter.Error(); e != nil {
			return e
		}
		return z.Close()
	})
	if e != nil {
		return e
	}
	attached := false
	defer func() {
		if !attached {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, _ = a.DB.Exec(ctx, "UPDATE artifacts SET state='deleting' WHERE id=$1", artifact)
		}
	}()
	tx, e := a.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var current int64
	if e = tx.QueryRow(r.Context(), "SELECT links_revision FROM transfers WHERE id::text=$1 AND owner_id=$2 AND state='active' AND absolute_expires_at>clock_timestamp() AND (frozen OR expires_at>clock_timestamp()) FOR UPDATE", id, p.ID).Scan(&current); e != nil {
		return e
	}
	if current != revision {
		return transfer.ErrConflict
	}
	if _, e = tx.Exec(r.Context(), "INSERT INTO qr_bundles(artifact_id,transfer_id,links_revision) VALUES($1,$2,$3)", artifact, id, revision); e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	attached = true
	return reply(w, 201, map[string]string{"artifactId": artifact})
}
