CREATE TABLE identity_providers (
 id uuid PRIMARY KEY, revision bigint NOT NULL DEFAULT 1 CHECK (revision>0),
 config jsonb NOT NULL, secret_key_id uuid NOT NULL REFERENCES key_versions(id), secret_box bytea NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE external_identities (
 provider_id uuid NOT NULL REFERENCES identity_providers(id), subject text NOT NULL,
 user_id uuid NOT NULL REFERENCES users(id), created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(provider_id,subject), UNIQUE(provider_id,user_id)
);
CREATE TABLE identity_flows (
 state_hash text PRIMARY KEY, binding_hash text NOT NULL, provider_id uuid NOT NULL REFERENCES identity_providers(id),
 revision bigint NOT NULL, key_id uuid NOT NULL REFERENCES key_versions(id), box bytea NOT NULL,
 verified boolean NOT NULL DEFAULT false, expires_at timestamptz NOT NULL DEFAULT clock_timestamp()+interval '5 minutes'
);
CREATE INDEX identity_flows_expiry ON identity_flows(expires_at);
ALTER TABLE web_sessions ADD COLUMN identity_provider_id uuid REFERENCES identity_providers(id);
