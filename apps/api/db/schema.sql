CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE users (
 id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
 email citext UNIQUE NOT NULL,
 name text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 100),
 password_hash text NOT NULL,
 role text NOT NULL DEFAULT 'reader' CHECK (role IN ('admin','author','reader')),
 disabled_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE categories (
 id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
 name text NOT NULL UNIQUE, slug text NOT NULL UNIQUE
);
CREATE TABLE tags (
 id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
 name text NOT NULL UNIQUE, slug text NOT NULL UNIQUE
);
CREATE TABLE posts (
 id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
 author_id uuid NOT NULL REFERENCES users(id),
 category_id uuid REFERENCES categories(id) ON DELETE SET NULL,
 slug text NOT NULL UNIQUE CHECK (char_length(slug) BETWEEN 1 AND 200),
 title text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 240),
 content_html text NOT NULL DEFAULT '',
 content_json jsonb NOT NULL DEFAULT '{}',
 cover_image_url text,
 excerpt text NOT NULL DEFAULT '' CHECK (char_length(excerpt) <= 500),
 reading_time_minutes integer NOT NULL DEFAULT 1 CHECK (reading_time_minutes > 0),
 views_count bigint NOT NULL DEFAULT 0 CHECK (views_count >= 0),
 status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published','archived','deleted')),
 published_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK (status <> 'published' OR published_at IS NOT NULL)
);
CREATE TABLE post_tags (
 post_id uuid NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
 tag_id uuid NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
 PRIMARY KEY(post_id, tag_id)
);
CREATE TABLE comments (
 id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
 post_id uuid NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
 user_id uuid REFERENCES users(id) ON DELETE SET NULL,
 parent_id uuid,
 author_name text NOT NULL CHECK (char_length(author_name) BETWEEN 2 AND 100),
 author_email citext NOT NULL,
 content text NOT NULL CHECK (char_length(content) BETWEEN 2 AND 5000),
 status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','spam')),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(id,post_id),
 FOREIGN KEY (parent_id,post_id) REFERENCES comments(id,post_id) ON DELETE CASCADE,
 CHECK (parent_id IS NULL OR parent_id <> id)
);
-- An immutable parent relation makes cycles impossible through the application.
CREATE INDEX posts_published_idx ON posts(published_at DESC,id DESC) WHERE status='published';
CREATE INDEX posts_popular_idx ON posts(views_count DESC,id) WHERE status='published';
CREATE INDEX posts_author_idx ON posts(author_id);
CREATE INDEX posts_category_idx ON posts(category_id,published_at DESC);
CREATE INDEX post_tags_tag_idx ON post_tags(tag_id,post_id);
CREATE INDEX comments_post_idx ON comments(post_id,status,created_at,id);
CREATE INDEX comments_parent_idx ON comments(parent_id);
-- simple does not stem Persian; trigram indexes support Persian substring suggestions.
CREATE INDEX posts_title_trgm_idx ON posts USING gin(title gin_trgm_ops);
CREATE INDEX posts_content_trgm_idx ON posts USING gin(content_html gin_trgm_ops);
CREATE INDEX posts_search_idx ON posts USING gin(to_tsvector('simple',title || ' ' || content_html));

CREATE TABLE email_outbox (
 id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
 recipient text NOT NULL, subject text NOT NULL, body text NOT NULL,
 attempts integer NOT NULL DEFAULT 0, available_at timestamptz NOT NULL DEFAULT now(),
 sent_at timestamptz, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX email_outbox_ready_idx ON email_outbox(available_at) WHERE sent_at IS NULL;
