CREATE UNIQUE INDEX one_running_rotation ON rotation_jobs(class) WHERE state='running';
ALTER TABLE artifacts ADD COLUMN maintenance_epoch bigint NOT NULL DEFAULT 0;
ALTER TABLE artifacts ADD COLUMN maintenance_until timestamptz;
CREATE FUNCTION protect_publish_request() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF OLD.published_at IS NOT NULL AND NEW.publish_request_hash IS DISTINCT FROM OLD.publish_request_hash THEN RAISE EXCEPTION 'publish request is immutable' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER transfers_request BEFORE UPDATE ON transfers FOR EACH ROW EXECUTE FUNCTION protect_publish_request();
