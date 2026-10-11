-- name: CreateUser :one
INSERT INTO users(email,name,password_hash,role) VALUES($1,$2,$3,$4) RETURNING *;
-- name: UserByEmail :one
SELECT * FROM users WHERE email=$1;
-- name: UserByID :one
SELECT * FROM users WHERE id=$1;
-- name: ListAuthors :many
SELECT * FROM users WHERE role='author' ORDER BY created_at DESC,id LIMIT $1 OFFSET $2;
-- name: CountAuthors :one
SELECT count(*) FROM users WHERE role='author';
-- name: UpdateAuthor :one
UPDATE users SET name=$2,email=$3,
 disabled_at=CASE WHEN sqlc.arg(disabled)::boolean THEN COALESCE(disabled_at,now()) ELSE NULL END
WHERE id=$1 AND role='author' RETURNING *;
-- name: ListPosts :many
SELECT p.* FROM posts p
WHERE p.status='published' AND p.published_at<=now()
AND (sqlc.arg(category)::text='' OR p.category_id IN (SELECT id FROM categories WHERE slug=sqlc.arg(category)))
AND (sqlc.arg(tag)::text='' OR EXISTS (SELECT 1 FROM post_tags pt JOIN tags t ON t.id=pt.tag_id WHERE pt.post_id=p.id AND t.slug=sqlc.arg(tag)))
ORDER BY p.published_at DESC,p.id DESC LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);
-- name: CountPosts :one
SELECT count(*) FROM posts p WHERE p.status='published' AND p.published_at<=now()
AND (sqlc.arg(category)::text='' OR p.category_id IN (SELECT id FROM categories WHERE slug=sqlc.arg(category)))
AND (sqlc.arg(tag)::text='' OR EXISTS (SELECT 1 FROM post_tags pt JOIN tags t ON t.id=pt.tag_id WHERE pt.post_id=p.id AND t.slug=sqlc.arg(tag)));
-- name: PopularPosts :many
SELECT * FROM posts WHERE status='published' AND published_at<=now() ORDER BY views_count DESC,published_at DESC,id LIMIT 6;
-- name: SuggestPosts :many
SELECT * FROM posts WHERE status='published' AND published_at<=now() AND (title ILIKE sqlc.arg(pattern)::text OR content_html ILIKE sqlc.arg(pattern)::text)
ORDER BY (title ILIKE sqlc.arg(pattern)::text) DESC,views_count DESC,id LIMIT 8;
-- name: PostBySlug :one
SELECT * FROM posts WHERE slug=$1 AND status='published' AND published_at<=now();
-- name: PostByID :one
SELECT * FROM posts WHERE id=$1;
-- name: IncrementViews :one
UPDATE posts SET views_count=views_count+1 WHERE id=$1 AND status='published' RETURNING views_count;
-- name: OwnPosts :many
SELECT sqlc.embed(p),u.name AS author_name FROM posts p JOIN users u ON u.id=p.author_id
WHERE p.status<>'deleted' AND (sqlc.arg(is_admin)::boolean OR (p.author_id=sqlc.arg(author_id) AND p.status<>'archived')) ORDER BY p.updated_at DESC,p.id LIMIT 200;
-- name: DeletedPosts :many
SELECT sqlc.embed(p),u.name AS author_name FROM posts p JOIN users u ON u.id=p.author_id
WHERE p.status='deleted' ORDER BY p.updated_at DESC,p.id LIMIT 200;
-- name: ChangePostStatus :one
UPDATE posts SET status=sqlc.arg(new_status)::text,updated_at=now()
WHERE id=$1 AND (sqlc.arg(new_status)::text IN ('deleted','draft') OR status<>'deleted') RETURNING *;
-- name: SavePost :one
INSERT INTO posts(id,author_id,category_id,slug,title,content_html,content_json,excerpt,reading_time_minutes,status,published_at,cover_image_url)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,CASE WHEN $10='published' THEN now() ELSE NULL END,$11)
ON CONFLICT(id) DO UPDATE SET category_id=EXCLUDED.category_id,slug=EXCLUDED.slug,title=EXCLUDED.title,content_html=EXCLUDED.content_html,content_json=EXCLUDED.content_json,excerpt=EXCLUDED.excerpt,reading_time_minutes=EXCLUDED.reading_time_minutes,status=EXCLUDED.status,cover_image_url=EXCLUDED.cover_image_url,published_at=CASE WHEN EXCLUDED.status='published' THEN COALESCE(posts.published_at,now()) ELSE posts.published_at END,updated_at=now()
WHERE posts.status NOT IN ('archived','deleted') AND posts.author_id=EXCLUDED.author_id
RETURNING *;
-- name: ClearPostTags :exec
DELETE FROM post_tags WHERE post_id=$1;
-- name: AddPostTag :exec
INSERT INTO post_tags(post_id,tag_id) VALUES($1,$2);
-- name: PostTags :many
SELECT t.* FROM tags t JOIN post_tags pt ON pt.tag_id=t.id WHERE pt.post_id=$1 ORDER BY t.name;
-- name: Categories :many
SELECT * FROM categories ORDER BY name;
-- name: Tags :many
SELECT * FROM tags ORDER BY name;
-- name: AddCategory :one
INSERT INTO categories(name,slug) VALUES($1,$2) RETURNING *;
-- name: AddTag :one
INSERT INTO tags(name,slug) VALUES($1,$2) RETURNING *;
-- name: CommentTree :many
WITH RECURSIVE tree AS (
 SELECT c.*,1 AS depth FROM comments c WHERE c.post_id=$1 AND c.parent_id IS NULL AND c.status='approved'
 UNION ALL
 SELECT c.*,t.depth+1 FROM comments c JOIN tree t ON c.parent_id=t.id AND c.post_id=t.post_id WHERE c.status='approved'
) SELECT id,post_id,user_id,parent_id,author_name,author_email,content,status,created_at,updated_at FROM tree ORDER BY created_at,id;
-- name: CommentByID :one
SELECT * FROM comments WHERE id=$1;
-- name: CommentDepth :one
WITH RECURSIVE ancestors AS (
 SELECT root.id,root.parent_id,1 AS depth FROM comments root WHERE root.id=$1
 UNION ALL SELECT c.id,c.parent_id,a.depth+1 FROM comments c JOIN ancestors a ON a.parent_id=c.id
) SELECT COALESCE(max(depth),0)::int FROM ancestors;
-- name: AddComment :one
INSERT INTO comments(post_id,user_id,parent_id,author_name,author_email,content) VALUES($1,$2,$3,$4,$5,$6) RETURNING *;
-- name: ModerateComment :one
UPDATE comments SET status=$2,updated_at=now() WHERE id=$1 RETURNING *;
-- name: ModerationComments :many
SELECT c.* FROM comments c JOIN posts p ON p.id=c.post_id WHERE c.status=sqlc.arg(status)::text AND (sqlc.arg(is_admin)::boolean OR p.author_id=sqlc.arg(author_id)) ORDER BY c.created_at,c.id LIMIT 200;
-- name: EnqueueEmail :exec
INSERT INTO email_outbox(recipient,subject,body) VALUES($1,$2,$3);
-- name: ClaimEmail :one
UPDATE email_outbox SET available_at=now()+interval '5 minutes',attempts=attempts+1
WHERE id=(SELECT id FROM email_outbox WHERE sent_at IS NULL AND available_at<=now() AND attempts<12 ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1) RETURNING *;
-- name: CompleteEmail :exec
UPDATE email_outbox SET sent_at=now() WHERE id=$1;
