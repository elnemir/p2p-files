CREATE TABLE users (
 id uuid PRIMARY KEY,
 email text NOT NULL UNIQUE,
 role text NOT NULL CHECK (role IN ('user','admin')),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE app_settings (
 revision bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 active boolean NOT NULL DEFAULT false,
 value jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE UNIQUE INDEX app_settings_one_active ON app_settings(active) WHERE active;
INSERT INTO app_settings(active,value) VALUES(true,'{"version":1,"transferMaxTtlSeconds":604800,"maxFiles":10000,"maxPathBytes":1024}');
CREATE TABLE transfers (
 id uuid PRIMARY KEY,
 owner_id uuid NOT NULL REFERENCES users(id),
 state text NOT NULL DEFAULT 'draft' CHECK (state IN ('draft','active','completed','expired','revoked')),
 revision bigint NOT NULL DEFAULT 1 CHECK (revision>0),
 published_revision bigint,
 published_ttl_seconds bigint,
 published_at timestamptz,
 expires_at timestamptz,
 absolute_expires_at timestamptz,
 manifest bytea,
 manifest_hash text CHECK (manifest_hash ~ '^[0-9a-f]{64}$'),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 CHECK ((published_at IS NULL AND manifest IS NULL AND manifest_hash IS NULL AND published_revision IS NULL AND published_ttl_seconds IS NULL AND expires_at IS NULL AND absolute_expires_at IS NULL) OR (published_at IS NOT NULL AND manifest IS NOT NULL AND manifest_hash IS NOT NULL AND published_revision IS NOT NULL AND published_ttl_seconds>0 AND expires_at IS NOT NULL AND absolute_expires_at IS NOT NULL)),
 CHECK (state NOT IN ('active','completed','expired') OR published_at IS NOT NULL),
 CHECK (expires_at <= absolute_expires_at)
);
CREATE INDEX transfers_owner_created ON transfers(owner_id,created_at,id);
CREATE TABLE files (
 id uuid PRIMARY KEY,
 transfer_id uuid NOT NULL REFERENCES transfers(id),
 path text NOT NULL,
 path_key text NOT NULL,
 size bigint NOT NULL CHECK (size>=0),
 object_key text NOT NULL UNIQUE,
 sha256 text CHECK (sha256 ~ '^[0-9a-f]{64}$'),
 state text NOT NULL DEFAULT 'uploading' CHECK (state IN ('uploading','finalizing','ready','leased','deleting','deleted','cancelled','rejected','integrity_failed','storage_missing')),
 UNIQUE(transfer_id,path_key),
 CHECK (state NOT IN ('ready','leased') OR sha256 IS NOT NULL)
);
CREATE TABLE outbox (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 aggregate_id uuid NOT NULL REFERENCES transfers(id),
 event_type text NOT NULL,
 deduplication_key text NOT NULL UNIQUE,
 payload jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 delivered_at timestamptz
);
CREATE INDEX outbox_pending ON outbox(id) WHERE delivered_at IS NULL;
CREATE INDEX outbox_aggregate ON outbox(aggregate_id);
CREATE FUNCTION protect_file_identity() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE parent_state text;
BEGIN
 IF TG_OP='UPDATE' AND NEW.transfer_id<>OLD.transfer_id THEN
  RAISE EXCEPTION 'file reparenting is forbidden' USING ERRCODE='23514';
 END IF;
 IF TG_OP='DELETE' THEN
  SELECT state INTO parent_state FROM transfers WHERE id=OLD.transfer_id FOR UPDATE;
 ELSE
  SELECT state INTO parent_state FROM transfers WHERE id=NEW.transfer_id FOR UPDATE;
 END IF;
 IF parent_state<>'draft' THEN
  IF TG_OP IN ('INSERT','DELETE') THEN
   RAISE EXCEPTION 'published file set is immutable' USING ERRCODE='23514';
  ELSIF ROW(NEW.id,NEW.path,NEW.path_key,NEW.size,NEW.sha256,NEW.object_key) IS DISTINCT FROM ROW(OLD.id,OLD.path,OLD.path_key,OLD.size,OLD.sha256,OLD.object_key) THEN
   RAISE EXCEPTION 'published file identity is immutable' USING ERRCODE='23514';
  END IF;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER files_identity BEFORE INSERT OR UPDATE OR DELETE ON files FOR EACH ROW EXECUTE FUNCTION protect_file_identity();
CREATE FUNCTION protect_transfer_manifest() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.owner_id<>OLD.owner_id OR NEW.id<>OLD.id THEN
  RAISE EXCEPTION 'transfer ownership is immutable' USING ERRCODE='23514';
 END IF;
 IF OLD.published_at IS NOT NULL AND (NEW.state='draft' OR ROW(NEW.manifest,NEW.manifest_hash,NEW.published_at,NEW.published_revision,NEW.published_ttl_seconds,NEW.absolute_expires_at) IS DISTINCT FROM ROW(OLD.manifest,OLD.manifest_hash,OLD.published_at,OLD.published_revision,OLD.published_ttl_seconds,OLD.absolute_expires_at)) THEN
  RAISE EXCEPTION 'published manifest is immutable' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER transfers_manifest BEFORE UPDATE ON transfers FOR EACH ROW EXECUTE FUNCTION protect_transfer_manifest();
CREATE TABLE bootstrap_secrets (
 name text PRIMARY KEY,
 envelope bytea NOT NULL
);
