ALTER TABLE reports ADD COLUMN render_state text NOT NULL DEFAULT 'pending' CHECK(render_state IN ('pending','running','ready','failed'));
ALTER TABLE reports ADD COLUMN render_fence bigint NOT NULL DEFAULT 0;
ALTER TABLE reports ADD COLUMN render_lease_until timestamptz;
ALTER TABLE reports ADD COLUMN render_error text;
CREATE INDEX reports_render_queue ON reports(render_state,created_at);
