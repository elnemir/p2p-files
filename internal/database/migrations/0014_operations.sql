-- The recovery ledger survives application data replacement.
CREATE TABLE ops_control (
 singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton),
 mode text NOT NULL DEFAULT 'open' CHECK(mode IN ('open','maintenance','restoring','reconcile','blocked')),
 generation uuid NOT NULL DEFAULT gen_random_uuid(),
 reason text NOT NULL DEFAULT '', actor text NOT NULL DEFAULT 'bootstrap',
 restore_id uuid, reconciliation_id uuid,
 changed_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
INSERT INTO ops_control(singleton) VALUES(true);
CREATE TABLE ops_secrets(name text PRIMARY KEY,envelope bytea NOT NULL);
CREATE TABLE ops_jobs(
 id uuid PRIMARY KEY,kind text NOT NULL CHECK(kind IN ('backup','restore','reconcile','audit','storage')),
 state text NOT NULL DEFAULT 'queued' CHECK(state IN ('queued','running','completed','failed')),
 actor text NOT NULL,reason text NOT NULL,source_id uuid,provider_id uuid,provider_envelope bytea,
 pre_backup_id uuid,progress bigint NOT NULL DEFAULT 0,error_code text,result jsonb NOT NULL DEFAULT '{}',
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),started_at timestamptz,completed_at timestamptz
);
CREATE INDEX ops_jobs_queue ON ops_jobs(created_at,id) WHERE state IN ('queued','running');
CREATE INDEX ops_jobs_history ON ops_jobs(created_at DESC,id);
CREATE TABLE ops_events(id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,payload bytea NOT NULL,previous_hash text NOT NULL,hash text NOT NULL);
CREATE TABLE ops_reconciliations(id uuid PRIMARY KEY,generation uuid NOT NULL,report jsonb NOT NULL,hash text NOT NULL,passed boolean NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp());
CREATE TABLE ops_provider_history(id uuid PRIMARY KEY,envelope bytea NOT NULL);
CREATE TABLE ops_schedules(kind text PRIMARY KEY CHECK(kind IN ('backup','audit','storage')),interval_seconds int NOT NULL CHECK(interval_seconds=0 OR interval_seconds BETWEEN 60 AND 2592000),next_at timestamptz NOT NULL,provider_id uuid);
INSERT INTO ops_schedules VALUES('backup',0,clock_timestamp(),NULL),('audit',3600,clock_timestamp()+interval '1 hour',NULL),('storage',86400,clock_timestamp()+interval '1 day',NULL);
CREATE TABLE security_incidents(id uuid PRIMARY KEY,kind text NOT NULL,subject text NOT NULL,severity text NOT NULL CHECK(severity IN ('warning','critical')),state text NOT NULL DEFAULT 'open' CHECK(state IN ('open','acknowledged','resolved')),created_at timestamptz NOT NULL DEFAULT clock_timestamp(),updated_at timestamptz NOT NULL DEFAULT clock_timestamp());
CREATE UNIQUE INDEX security_incident_active ON security_incidents(kind,subject) WHERE state<>'resolved';
CREATE TABLE security_incident_reports(id uuid PRIMARY KEY,incident_id uuid NOT NULL REFERENCES security_incidents(id),revision int NOT NULL,key_id uuid NOT NULL REFERENCES key_versions(id),envelope bytea NOT NULL,sha256 text NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),UNIQUE(incident_id,revision));
CREATE INDEX security_report_key ON security_incident_reports(key_id);
CREATE TABLE audit_checkpoints(id uuid PRIMARY KEY,kind text NOT NULL CHECK(kind IN ('regular','recovery')),event_id bigint NOT NULL,event_hash text NOT NULL,evidence_hash text NOT NULL,payload bytea NOT NULL,signature text NOT NULL,reason text NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp());
CREATE INDEX audit_checkpoint_latest ON audit_checkpoints(created_at DESC,id);
CREATE TABLE ops_checkpoints(id uuid PRIMARY KEY,event_id bigint NOT NULL,payload bytea NOT NULL,signature text NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp());
