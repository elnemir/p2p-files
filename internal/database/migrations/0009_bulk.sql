ALTER TABLE bulk_jobs DROP CONSTRAINT bulk_jobs_state_check;
ALTER TABLE bulk_jobs ADD CONSTRAINT bulk_jobs_state_check CHECK(state IN ('paused','queued','running','completed','cancelled'));
ALTER TABLE bulk_jobs ADD COLUMN admin_scope boolean NOT NULL DEFAULT false;
ALTER TABLE bulk_jobs ADD COLUMN account_epoch bigint NOT NULL DEFAULT 1;
ALTER TABLE bulk_jobs ADD COLUMN reason text NOT NULL DEFAULT '';
CREATE TABLE draft_folders(id uuid PRIMARY KEY,transfer_id uuid NOT NULL REFERENCES transfers(id),path text NOT NULL,path_key text NOT NULL,UNIQUE(transfer_id,path_key));
CREATE INDEX draft_folders_transfer ON draft_folders(transfer_id);
