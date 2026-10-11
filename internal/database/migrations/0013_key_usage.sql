ALTER TABLE key_versions ADD COLUMN seal_count bigint NOT NULL DEFAULT 0 CHECK(seal_count>=0 AND seal_count<=1048576);
-- Prior releases did not count attempts; force rotation before further root-key seals.
UPDATE key_versions SET seal_count=1048576;
CREATE TABLE crypto_usage(name text PRIMARY KEY,seal_count bigint NOT NULL CHECK(seal_count>=0 AND seal_count<=1048576));
INSERT INTO crypto_usage(name,seal_count) SELECT 'kek',count(*)+1 FROM key_versions;
