CREATE TABLE ops_license (
 singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton),
 installation_id uuid NOT NULL DEFAULT gen_random_uuid(),
 epoch bigint NOT NULL DEFAULT 0 CHECK(epoch>=0),
 envelope text NOT NULL DEFAULT '', pending_envelope text NOT NULL DEFAULT '',
 entitlement_id text NOT NULL DEFAULT '', high_revision bigint NOT NULL DEFAULT 0,
 max_observed_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
INSERT INTO ops_license(singleton) VALUES(true);
ALTER TABLE files ADD COLUMN license_bytes bigint NOT NULL DEFAULT 0 CHECK(license_bytes>=0);
ALTER TABLE files ADD COLUMN license_path_key text NOT NULL DEFAULT '';
UPDATE files SET license_bytes=size,license_path_key=path_key;
-- Follow visible renames so later reservations cannot reuse an obsolete path key.
CREATE FUNCTION filex_license_path() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.staged_until IS NULL AND NEW.path_key NOT LIKE '#staged:%' THEN
  NEW.license_path_key=NEW.path_key;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER license_path BEFORE UPDATE OF path_key,staged_until ON files FOR EACH ROW EXECUTE FUNCTION filex_license_path();
