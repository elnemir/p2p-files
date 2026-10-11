-- Privileged export access remains bound to a current ADMIN web session.
ALTER TABLE artifact_credentials ADD COLUMN admin_access boolean NOT NULL DEFAULT false;
