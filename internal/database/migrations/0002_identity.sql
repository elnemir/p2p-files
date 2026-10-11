ALTER TABLE users ADD COLUMN password_hash text;
ALTER TABLE users ADD COLUMN disabled boolean NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN auth_epoch bigint NOT NULL DEFAULT 1;
ALTER TABLE users ADD COLUMN locale text NOT NULL DEFAULT 'ru' CHECK (locale IN ('ru','en'));
CREATE UNIQUE INDEX users_email_folded ON users(lower(email));
CREATE TABLE password_history(id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,user_id uuid NOT NULL REFERENCES users(id),hash text NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp());
CREATE INDEX password_history_user ON password_history(user_id,id DESC);
CREATE TABLE web_sessions(
 id uuid PRIMARY KEY,user_id uuid NOT NULL REFERENCES users(id),token_hash text NOT NULL UNIQUE,
 csrf_hash text NOT NULL,auth_epoch bigint NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 expires_at timestamptz NOT NULL,reauth_at timestamptz NOT NULL DEFAULT clock_timestamp(),revoked_at timestamptz
);
CREATE INDEX web_sessions_user ON web_sessions(user_id);
CREATE TABLE api_tokens(id uuid PRIMARY KEY,user_id uuid NOT NULL REFERENCES users(id),token_hash text NOT NULL UNIQUE,scopes text[] NOT NULL,auth_epoch bigint NOT NULL,expires_at timestamptz NOT NULL,revoked_at timestamptz);
CREATE INDEX api_tokens_user ON api_tokens(user_id);
CREATE TABLE auth_limits(key text PRIMARY KEY,window_start timestamptz NOT NULL,count int NOT NULL);
CREATE TABLE login_history(id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,user_id uuid REFERENCES users(id),occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(),success boolean NOT NULL);
CREATE INDEX login_history_user ON login_history(user_id,id DESC);
CREATE TABLE audit_events(id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,actor_id text NOT NULL,action text NOT NULL,subject_id text NOT NULL,occurred_at timestamptz NOT NULL,payload bytea NOT NULL,previous_hash text NOT NULL,hash text NOT NULL);
ALTER TABLE app_settings ADD COLUMN created_by uuid REFERENCES users(id);
ALTER TABLE app_settings ADD COLUMN validated boolean NOT NULL DEFAULT false;
CREATE INDEX app_settings_creator ON app_settings(created_by);
UPDATE app_settings SET validated=true,value=value || '{"publicOrigin":"http://localhost:8090","sessionTtlSeconds":43200}'::jsonb;
