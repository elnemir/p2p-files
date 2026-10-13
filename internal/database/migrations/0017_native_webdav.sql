-- Existing resources receive a documented migration-time metadata baseline.
ALTER TABLE files ADD COLUMN created_at timestamptz NOT NULL DEFAULT clock_timestamp();
ALTER TABLE files ADD COLUMN modified_at timestamptz NOT NULL DEFAULT clock_timestamp();
ALTER TABLE files ADD COLUMN dav_properties jsonb NOT NULL DEFAULT '[]';
ALTER TABLE files ADD COLUMN staged_until timestamptz;
CREATE INDEX files_staged_expiry ON files(staged_until,transfer_id) WHERE staged_until IS NOT NULL AND state NOT IN ('cancelled','deleted');
ALTER TABLE draft_folders ADD COLUMN created_at timestamptz NOT NULL DEFAULT clock_timestamp();
ALTER TABLE draft_folders ADD COLUMN modified_at timestamptz NOT NULL DEFAULT clock_timestamp();
ALTER TABLE draft_folders ADD COLUMN dav_properties jsonb NOT NULL DEFAULT '[]';
ALTER TABLE transfers ADD COLUMN dav_properties jsonb NOT NULL DEFAULT '[]';
CREATE TABLE dav_locks (
 token uuid PRIMARY KEY,
 transfer_id uuid NOT NULL REFERENCES transfers(id),
 credential_id uuid NOT NULL REFERENCES api_tokens(id),
 path text NOT NULL,
 path_key text NOT NULL,
 recursive boolean NOT NULL,
 owner_xml text NOT NULL DEFAULT '',
 expires_at timestamptz NOT NULL,
 CHECK(octet_length(owner_xml)<=4096)
);
CREATE INDEX dav_locks_transfer ON dav_locks(transfer_id,expires_at);
CREATE INDEX dav_locks_credential ON dav_locks(credential_id);
ALTER TABLE files ADD CONSTRAINT dav_properties_bounded CHECK(octet_length(dav_properties::text)<=65536);
ALTER TABLE draft_folders ADD CONSTRAINT dav_properties_bounded CHECK(octet_length(dav_properties::text)<=65536);
ALTER TABLE transfers ADD CONSTRAINT dav_properties_bounded CHECK(octet_length(dav_properties::text)<=65536);
