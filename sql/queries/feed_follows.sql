-- name: CreateFeedFollow :one
WITH inserted AS (
  INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
  VALUES ($1, $2, $3, $4, $5)
  RETURNING *
)
SELECT inserted.*, f.name AS feed_name, u.name AS user_name
FROM inserted
JOIN feeds f ON inserted.feed_id = f.id
JOIN users u ON inserted.user_id = u.id;

-- name: GetFeedFollowsForUser :many
SELECT
    ff.*,
    feeds.name AS feed_name,
    users.name AS user_name
FROM feed_follows ff
JOIN feeds  ON ff.feed_id   = feeds.id
JOIN users  ON ff.user_id   = users.id
WHERE ff.user_id = $1;

-- name: DeleteFeedFollow :exec
DELETE FROM feed_follows 
WHERE feed_follows.user_id = $1 AND feed_follows.feed_id = (
    SELECT id FROM feeds WHERE url = $2
);
