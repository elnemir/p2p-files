ALTER TABLE storage_providers ADD COLUMN secret_key_id uuid REFERENCES key_versions(id);
ALTER TABLE storage_providers ADD COLUMN secret_envelope bytea;
CREATE TABLE storage_migrations(id uuid PRIMARY KEY,source_id uuid NOT NULL REFERENCES storage_providers(id),target_id uuid NOT NULL REFERENCES storage_providers(id),state text NOT NULL DEFAULT 'paused' CHECK(state IN ('paused','running','completed','cancelled','failed')),created_at timestamptz NOT NULL DEFAULT clock_timestamp(),moved_files int NOT NULL DEFAULT 0,CHECK(source_id<>target_id));
CREATE INDEX storage_migrations_source ON storage_migrations(source_id);
CREATE INDEX storage_migrations_target ON storage_migrations(target_id);
CREATE TABLE rotation_jobs(id uuid PRIMARY KEY,class text NOT NULL,target_key_id uuid NOT NULL REFERENCES key_versions(id),state text NOT NULL DEFAULT 'running',processed int NOT NULL DEFAULT 0,created_at timestamptz NOT NULL DEFAULT clock_timestamp());
