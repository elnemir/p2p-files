CREATE TABLE branding_revisions (
 revision bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 created_by uuid NOT NULL REFERENCES users(id),
 value jsonb NOT NULL CHECK (jsonb_typeof(value)='object' AND octet_length(value::text)<=614400),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX branding_revisions_creator ON branding_revisions(created_by);
ALTER TABLE reports ADD COLUMN branding jsonb;
ALTER TABLE security_incident_reports ADD COLUMN branding jsonb;
-- Presentation changes must never rewrite an already issued report revision.
CREATE FUNCTION protect_branding_snapshot() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF NEW.branding IS DISTINCT FROM OLD.branding THEN
  RAISE EXCEPTION 'report branding is immutable' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER report_branding_immutable BEFORE UPDATE ON reports FOR EACH ROW EXECUTE FUNCTION protect_branding_snapshot();
CREATE TRIGGER incident_branding_immutable BEFORE UPDATE ON security_incident_reports FOR EACH ROW EXECUTE FUNCTION protect_branding_snapshot();
CREATE FUNCTION protect_branding_revision() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 RAISE EXCEPTION 'branding revision is immutable' USING ERRCODE='23514';
END $$;
CREATE TRIGGER branding_revision_immutable BEFORE UPDATE ON branding_revisions FOR EACH ROW EXECUTE FUNCTION protect_branding_revision();
