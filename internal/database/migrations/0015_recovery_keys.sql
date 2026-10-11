-- A fresh database must not reuse an unknown historical KEK usage budget.
ALTER TABLE ops_control ADD COLUMN instance_id uuid NOT NULL DEFAULT gen_random_uuid();
