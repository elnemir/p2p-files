ALTER TABLE artifacts ADD COLUMN sha256 text CHECK(sha256 ~ '^[0-9a-f]{64}$');
ALTER TABLE transfers ADD COLUMN links_revision bigint NOT NULL DEFAULT 1;
CREATE TABLE qr_bundles(artifact_id uuid PRIMARY KEY REFERENCES artifacts(id),transfer_id uuid NOT NULL REFERENCES transfers(id),links_revision bigint NOT NULL);
CREATE INDEX qr_bundles_transfer ON qr_bundles(transfer_id);
CREATE FUNCTION invalidate_qr_links() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 UPDATE transfers SET links_revision=links_revision+1 WHERE id=NEW.transfer_id;
 UPDATE artifacts SET state='deleting' WHERE state IN ('writing','ready') AND id IN(SELECT artifact_id FROM qr_bundles WHERE transfer_id=NEW.transfer_id);
 RETURN NEW;
END $$;
CREATE TRIGGER qr_links_changed AFTER INSERT OR UPDATE OF token_hash,recipient_email,password_hash,epoch,revoked_at ON recipient_links FOR EACH ROW EXECUTE FUNCTION invalidate_qr_links();
CREATE FUNCTION invalidate_qr_transfer() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF NEW.state <> 'active' AND OLD.state='active' THEN
  UPDATE artifacts SET state='deleting' WHERE state IN ('writing','ready') AND id IN(SELECT artifact_id FROM qr_bundles WHERE transfer_id=NEW.id);
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER qr_transfer_changed AFTER UPDATE OF state ON transfers FOR EACH ROW EXECUTE FUNCTION invalidate_qr_transfer();
ALTER TABLE export_jobs ADD COLUMN auth_epoch bigint;
UPDATE export_jobs e SET auth_epoch=u.auth_epoch FROM users u WHERE u.id=e.owner_id;
ALTER TABLE export_jobs ALTER COLUMN auth_epoch SET NOT NULL;
CREATE FUNCTION stamp_export_epoch() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF NEW.auth_epoch IS NULL THEN SELECT auth_epoch INTO NEW.auth_epoch FROM users WHERE id=NEW.owner_id; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER export_epoch_insert BEFORE INSERT ON export_jobs FOR EACH ROW EXECUTE FUNCTION stamp_export_epoch();
