-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: ListFeedsWithUsers :many
SELECT
  f.name  AS feed_name,
  f.url   AS feed_url,
  u.name  AS user_name
FROM feeds f
JOIN users u ON u.id = f.user_id
ORDER BY f.created_at;


-- name: GetFeedByURL :one
SELECT *
FROM feeds
WHERE url = $1;

-- name: MarkFeedFetched :one
UPDATE feeds SET last_fetched_at = NOW(), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetNextFeedToFetch :one
SELECT * FROM feeds
ORDER BY last_fetched_at NULLS FIRST, created_at
LIMIT 1;