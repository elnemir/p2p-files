package files

import (
	"context"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/transfer"
	"github.com/jackc/pgx/v5"
)

// Cancel fences writers and records cleanup before releasing the draft name.
func (s Service) Cancel(ctx context.Context, owner, id string) error {
	f, err := s.Get(ctx, owner, id)
	if err != nil {
		return err
	}
	return s.mutate(ctx, f.TransferID, f.Path, func(tx pgx.Tx) error {
		key, e := domain.PathKey(f.Path, 4096)
		if e != nil {
			return e
		}
		if e = transfer.CheckDAVMembership(ctx, tx, f.TransferID, key); e != nil {
			return e
		}
		var frozen bool
		if err := tx.QueryRow(ctx, "SELECT frozen FROM files WHERE id=$1 FOR UPDATE", id).Scan(&frozen); err != nil {
			return err
		}
		if frozen {
			return transfer.ErrConflict
		}
		if _, err := tx.Exec(ctx, "UPDATE files SET state='cancelled',upload_epoch=upload_epoch+1,path_key='#cancelled:'||id::text WHERE id=$1 AND state<>'cancelled'", id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "INSERT INTO deletion_jobs(provider_id,object_key,file_id) SELECT $1,object_key,file_id FROM file_chunks WHERE file_id=$2 ON CONFLICT(object_key) DO NOTHING", f.ProviderID, id); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, "UPDATE transfers SET revision=revision+1 WHERE id=$1", f.TransferID)
		return err
	})
}
