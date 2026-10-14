CREATE TABLE IF NOT EXISTS issuer_flows (
 state_hash text PRIMARY KEY, binding_hash text NOT NULL, verifier text NOT NULL, nonce text NOT NULL,
 expires_at timestamptz NOT NULL DEFAULT clock_timestamp()+interval '5 minutes'
);
CREATE TABLE IF NOT EXISTS issuer_sessions (
 token_hash text PRIMARY KEY, subject text NOT NULL, authenticated_at timestamptz NOT NULL,
 expires_at timestamptz NOT NULL DEFAULT clock_timestamp()+interval '15 minutes'
);
CREATE TABLE IF NOT EXISTS issuer_issuances (
 id uuid PRIMARY KEY, idempotency_key uuid NOT NULL UNIQUE, request_hash text NOT NULL,
 entitlement_id uuid NOT NULL, revision bigint NOT NULL CHECK(revision>0), installation_id uuid NOT NULL,
 customer_id text NOT NULL, claims bytea NOT NULL, actor text NOT NULL,
 state text NOT NULL DEFAULT 'approved' CHECK(state IN ('approved','issued','signing_failed')),
 envelope text NOT NULL DEFAULT '', artifact_hash text NOT NULL DEFAULT '', key_id text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(), issued_at timestamptz,
 UNIQUE(entitlement_id,revision), CHECK((state='issued')=(envelope<>''))
);
CREATE TABLE IF NOT EXISTS issuer_outbox (
 issuance_id uuid PRIMARY KEY REFERENCES issuer_issuances(id), completed_at timestamptz
);
CREATE TABLE IF NOT EXISTS issuer_audit (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, actor text NOT NULL, action text NOT NULL,
 issuance_id uuid NOT NULL REFERENCES issuer_issuances(id), occurred_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE OR REPLACE FUNCTION issuer_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'issuance history is immutable'; END IF;
 IF NEW.id IS DISTINCT FROM OLD.id OR NEW.idempotency_key IS DISTINCT FROM OLD.idempotency_key
 OR NEW.request_hash IS DISTINCT FROM OLD.request_hash OR NEW.entitlement_id IS DISTINCT FROM OLD.entitlement_id
 OR NEW.revision IS DISTINCT FROM OLD.revision OR NEW.installation_id IS DISTINCT FROM OLD.installation_id
 OR NEW.customer_id IS DISTINCT FROM OLD.customer_id OR NEW.claims IS DISTINCT FROM OLD.claims
 OR NEW.actor IS DISTINCT FROM OLD.actor OR NEW.created_at IS DISTINCT FROM OLD.created_at
 OR OLD.state='issued' THEN RAISE EXCEPTION 'issuance history is immutable'; END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS immutable_issuance ON issuer_issuances;
CREATE TRIGGER immutable_issuance BEFORE UPDATE OR DELETE ON issuer_issuances FOR EACH ROW EXECUTE FUNCTION issuer_immutable();
CREATE OR REPLACE FUNCTION issuer_audit_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'issuer audit is append only'; END $$;
DROP TRIGGER IF EXISTS immutable_issuer_audit ON issuer_audit;
CREATE TRIGGER immutable_issuer_audit BEFORE UPDATE OR DELETE ON issuer_audit FOR EACH ROW EXECUTE FUNCTION issuer_audit_immutable();

-- Host-managed configuration encryption key is separate from the signing seed.
CREATE TABLE IF NOT EXISTS issuer_setup (
 singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton),
 config bytea NOT NULL,
 completed_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
DROP TRIGGER IF EXISTS immutable_issuer_setup ON issuer_setup;
CREATE TRIGGER immutable_issuer_setup BEFORE UPDATE OR DELETE ON issuer_setup
 FOR EACH ROW EXECUTE FUNCTION issuer_audit_immutable();
