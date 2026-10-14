-- This operational latch is excluded from ordinary snapshot restore.
CREATE TABLE ops_setup (
 singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton),
 completed boolean NOT NULL DEFAULT false
);
INSERT INTO ops_setup(singleton,completed) SELECT true,EXISTS(SELECT 1 FROM users);
CREATE FUNCTION filex_setup_monotonic() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' OR (OLD.completed AND NOT NEW.completed) THEN
  RAISE EXCEPTION 'setup cannot be reopened';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER setup_monotonic BEFORE UPDATE OR DELETE ON ops_setup
 FOR EACH ROW EXECUTE FUNCTION filex_setup_monotonic();
