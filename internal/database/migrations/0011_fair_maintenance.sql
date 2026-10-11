ALTER TABLE storage_migrations ADD COLUMN checked_at timestamptz NOT NULL DEFAULT '-infinity';
ALTER TABLE rotation_jobs ADD COLUMN checked_at timestamptz NOT NULL DEFAULT '-infinity';
