-- Preserve article contents if rolling back to the original workflow.
UPDATE posts SET status='draft' WHERE status IN ('archived','deleted');
ALTER TABLE posts DROP CONSTRAINT posts_status_check;
ALTER TABLE posts ADD CONSTRAINT posts_status_check CHECK (status IN ('draft','published'));
ALTER TABLE users DROP COLUMN disabled_at;
