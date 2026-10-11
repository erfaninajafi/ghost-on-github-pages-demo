ALTER TABLE users ADD COLUMN disabled_at timestamptz;
ALTER TABLE posts DROP CONSTRAINT posts_status_check;
ALTER TABLE posts ADD CONSTRAINT posts_status_check
 CHECK (status IN ('draft','published','archived','deleted'));
