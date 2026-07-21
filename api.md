
# 1. Creating all tables in the database

# 2. POST /sign-up

### Step 1 — Parse request body

Request uses `Content-Type: multipart/form-data` because it may include an image file.
Parse text fields using `r.FormValue()` and the avatar file using `r.FormFile()`.

```go
r.ParseMultipartForm(10 << 20) // 10 MB limit

email    := r.FormValue("email")
fullName := r.FormValue("full_name")
username := r.FormValue("username")
bio      := r.FormValue("bio")
password := r.FormValue("password")

file, header, err := r.FormFile("avatar") // optional
```

### Step 2 — Validation

- `email` is required
- `full_name` is required
- `username` is required
- `password` is required
- `bio` is optional
- `avatar` is optional
- `password` must be at least 8 characters

If validation fails → return `400 Bad Request`.

### Step 3 — Check email uniqueness

Query the DB:

```sql
SELECT id FROM users WHERE email = $1 LIMIT 1;
```

If a row is found → return `409 Conflict` (`email already exists`).

### Step 4 — Check username uniqueness

Query the DB:

```sql
SELECT id FROM users WHERE username = $1 LIMIT 1;
```

If a row is found → return `409 Conflict` (`username already exists`).

### Step 5 — Upload avatar (optional)

If `avatar` file is provided, upload it to storage and get back `avatar_path`.
If not provided, set `avatar_path` = `""`.

### Step 6 — Hash password

```go
hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
```

### Step 7 — Insert user into DB

`user_id` is generated automatically by DB since the column is `BIGSERIAL`.
`is_active` is set to `true` by default.

```sql
INSERT INTO users (email, full_name, username, bio, avatar_path, password, is_active)
VALUES ($1, $2, $3, $4, $5, $6, true)
RETURNING id, username, email, full_name, is_active;
```

### Step 8 — Generate JWT access token

JWT payload:

```json
{
    "user_id": 1,
    "username": "",
    "email": "",
    "full_name": "",
    "is_active": true
}
```

```go
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
accessToken, err := token.SignedString([]byte(secretKey))
```

### Step 9 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": {
        "access_token": ""
    }
}
```

# 3. POST /login

### Step 1 — Parse request body

Parse the JSON body into a struct using `json.Unmarshal`.

```go
type LoginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}
```

### Step 2 — Validation

- `email` is required
- `password` is required

If validation fails → return `400 Bad Request`.

### Step 3 — Find user by email

Query the DB:

```sql
SELECT id, username, email, full_name, is_active, password FROM users WHERE email = $1 LIMIT 1;
```

If no row is found → return `401 Unauthorized` (`invalid email or password`).

### Step 4 — Verify password

Compare the provided password against the stored hash using `bcrypt`.

```go
err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
```

If they don't match → return `401 Unauthorized` (`invalid email or password`).

### Step 5 — Generate JWT access token

JWT payload:

```json
{
    "user_id": 1,
    "username": "",
    "email": "",
    "full_name": "",
    "is_active": true
}
```

```go
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
accessToken, err := token.SignedString([]byte(secretKey))
```

### Step 6 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": {
        "access_token": ""
    }
}
```

# 4. POST /logout

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.

### Step 2 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": null
}
```

# 5. GET /users/:user_id

### Step 1 — Parse path parameter

Extract `user_id` from the URL path.
If `user_id` is not a valid integer → return `400 Bad Request`.

### Step 2 — Validate access token

Read the `Authorization` header and verify the JWT token.
Extract `user_id` from the token claims (needed for `is_following` check).

### Step 3 — Fetch user profile from DB

```sql
SELECT id, username, full_name, bio, avatar_path FROM users WHERE id = $1 LIMIT 1;
```

If no row is found → return `404 Not Found` (`user not found`).

### Step 4 — Fetch posts count

```sql
SELECT COUNT(id) FROM posts WHERE user_id = $1 AND deleted_at IS NULL;
```

### Step 5 — Fetch followers count

```sql
SELECT 
    COUNT(id) filter (where following_id = $1) as followers_count,
    count(id) filter (where follower_id = $1) as following_count
FROM follows;
```

### Step 6 — Check is_following

If the requesting user's `user_id` (from JWT) equals the profile's `user_id` → set `is_following` = `null` (viewing own profile).
Otherwise query the DB:

```sql
SELECT id FROM follows WHERE follower_id = $1 AND following_id = $2 LIMIT 1;
```

If a row is found → `is_following = true`, otherwise `is_following = false`.

### Step 7 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": {
        "user_id": 1,
        "username": "",
        "full_name": "",
        "bio": "",
        "avatar_path": "",
        "posts_count": 0,
        "followers_count": 0,
        "following_count": 0,
        "is_following": false
    }
}
```

# 6. GET /users/:user_id/posts

### Step 1 — Parse path parameter

Extract `user_id` from the URL path.
If `user_id` is not a valid integer → return `400 Bad Request`.

### Step 2 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.

### Step 3 — Parse query parameters

- `page` (default: 1)
- `per_page` (default: 10)

Calculate offset:

```go
offset := (page - 1) * perPage
```

### Step 4 — Fetch total count from DB

```sql
SELECT COUNT(id) FROM posts WHERE user_id = $1 AND deleted_at IS NULL;
```

### Step 5 — Fetch posts from DB

```sql
SELECT id, thumbnail_path, caption, created_at
FROM posts
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
```

If no rows found → return empty list.

### Step 6 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": {
        "count": 10,
        "items": [
            {
                "post_id": 1,
                "thumbnail_path": "",
                "caption": "",
                "created_at": ""
            }
        ]
    }
}
```

# 7. PUT /profile

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.
Extract `user_id` from the token claims.

### Step 2 — Parse request body

Request uses `Content-Type: multipart/form-data` because it may include an image file.
Parse text fields using `r.FormValue()` and the avatar file using `r.FormFile()`.

```go
r.ParseMultipartForm(5 << 20) // 5 MB limit

username := r.FormValue("username")
fullName := r.FormValue("full_name")
bio      := r.FormValue("bio")

file, header, err := r.FormFile("avatar") // optional
```

### Step 3 — Validation

All fields are optional, but at least one must be provided:
- `username`
- `full_name`
- `bio`
- `avatar`

If none of the fields are provided → return `400 Bad Request` (`at least one field must be provided`).

### Step 4 — Check username uniqueness

If `username` is provided, first check if it differs from the current value. If changed, query the DB:

```sql
SELECT id FROM users WHERE username = $1 AND id != $2 LIMIT 1;
```

If a row is found → return `409 Conflict` (`username already exists`).

### Step 5 — Validate and upload avatar (optional)

If `avatar` file is provided:

- Validate file size is not larger than **5 MB**. If it is → return `400 Bad Request` (`avatar must be at most 5 MB`).
- Validate file type is an image (e.g., `image/jpeg`, `image/png`, `image/webp`). If invalid → return `400 Bad Request` (`invalid image format`).
- Upload to storage and get back `avatar_path`.

If not provided, keep the existing `avatar_path`.

### Step 6 — Build dynamic UPDATE query

Only include fields that were provided in the request.

```sql
UPDATE users
SET username = $1, full_name = $2, bio = $3, avatar_path = $4, updated_at = now()
WHERE id = $5;
```

If a field was not provided, keep its current value in the database.

### Step 7 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": null
}
```

# 8. POST /post

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.
Extract `user_id` from the token claims.

### Step 2 — Parse request body

Request uses `Content-Type: multipart/form-data` because it includes an image file.
Parse text fields using `r.FormValue()` and the image file using `r.FormFile()`.

```go
r.ParseMultipartForm(10 << 20) // 10 MB limit

caption := r.FormValue("caption")

file, header, err := r.FormFile("image") // required
```

### Step 3 — Validation

- `image` is required
- `caption` is optional

If validation fails → return `400 Bad Request`.

### Step 4 — Validate and upload image

- Validate file size is not larger than **10 MB**. If it is → return `400 Bad Request` (`image must be at most 10 MB`).
- Validate file type is an image (e.g., `image/jpeg`, `image/png``). If invalid → return `400 Bad Request` (`invalid image format`).
- Upload the image to storage and get back `image_path`.
- Generate a thumbnail and get back `thumbnail_path`.

### Step 5 — Insert post into DB

```sql
INSERT INTO posts (user_id, image_path, thumbnail_path, caption)
VALUES ($1, $2, $3, $4)
RETURNING id;
```

### Step 6 — Extract and save hashtags (if any)

Parse `caption` for hashtags (e.g. `#nature`).
For each hashtag:

```sql
INSERT INTO hashtags (name) VALUES ($1)
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING id;
```

```sql
INSERT INTO post_hashtags (post_id, hashtag_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;
```

### Step 7 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": null
}
```

# 9. GET /feed

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.
Extract `user_id` from the token claims.

### Step 2 — Parse query parameters

- `page` (default: 1)
- `per_page` (default: 10)

### Step 3 — Fetch feed posts from DB

```sql
SELECT p.id, p.user_id, u.username, p.caption, p.thumbnail_path, p.likes_count, p.comments_count, p.created_at
FROM posts p
JOIN follows f ON f.following_id = p.user_id
JOIN users u ON u.id = p.user_id
WHERE f.follower_id = $1 AND p.deleted_at IS NULL
ORDER BY p.created_at DESC
LIMIT $2 OFFSET $3;
```

### Step 4 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": {
        "count": 10,
        "items": [
            {
                "user_id": 1,
                "username": "",
                "post_id": 1,
                "caption": "",
                "image_path": "",
                "likes_count": 0,
                "comments_count": 0,
                "created_at": ""
            }
        ]
    }
}
```

# 10. POST /post/:post_id/like

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.
Extract `user_id` from the token claims.

### Step 2 — Parse path parameter

Extract `post_id` from the URL path.
If `post_id` is not a valid integer → return `400 Bad Request`.

### Step 3 — Check post exists

```sql
SELECT id FROM posts WHERE id = $1 AND deleted_at IS NULL LIMIT 1;
```

If no row found → return `404 Not Found` (`post not found`).

### Step 4 — Insert like

```sql
INSERT INTO likes (user_id, post_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;
```

### Step 5 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": null
}
```

# 11. DELETE /post/:post_id/like

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.
Extract `user_id` from the token claims.

### Step 2 — Parse path parameter

Extract `post_id` from the URL path.
If `post_id` is not a valid integer → return `400 Bad Request`.

### Step 3 — Delete like

```sql
DELETE FROM likes WHERE user_id = $1 AND post_id = $2;
```

### Step 4 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": null
}
```

# 12. GET /post/:post_id

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.
Extract `user_id` from the token claims.

### Step 2 — Parse path parameter

Extract `post_id` from the URL path.
If `post_id` is not a valid integer → return `400 Bad Request`.

### Step 3 — Fetch post from DB

```sql
SELECT p.id, p.user_id, u.username, p.caption, p.image_path, p.likes_count, p.comments_count, p.created_at
FROM posts p
JOIN users u ON u.id = p.user_id
WHERE p.id = $1 AND p.deleted_at IS NULL
LIMIT 1;
```

If no row found → return `404 Not Found` (`post not found`).

### Step 4 — Check is_liked

```sql
SELECT id FROM likes WHERE user_id = $1 AND post_id = $2 LIMIT 1;
```

If a row is found → `is_liked = true`, otherwise `is_liked = false`.

### Step 5 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": {
        "post_id": 1,
        "user_id": 1,
        "username": "",
        "caption": "",
        "image_path": "",
        "likes_count": 0,
        "comments_count": 0,
        "created_at": "",
        "is_liked": false
    }
}
```

# 13. POST /post/:post_id/comments

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.
Extract `user_id` from the token claims.

### Step 2 — Parse path parameter

Extract `post_id` from the URL path.
If `post_id` is not a valid integer → return `400 Bad Request`.

### Step 3 — Parse request body

```go
type CreateCommentRequest struct {
    Content string `json:"content"`
}
```

### Step 4 — Validation

- `content` is required

If validation fails → return `400 Bad Request`.

### Step 5 — Check post exists

```sql
SELECT id FROM posts WHERE id = $1 AND deleted_at IS NULL LIMIT 1;
```

If no row found → return `404 Not Found` (`post not found`).

### Step 6 — Insert comment into DB

```sql
INSERT INTO comments (user_id, post_id, content) VALUES ($1, $2, $3);
```

### Step 7 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": null
}
```

# 14. DELETE /comments/:comment_id

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.
Extract `user_id` from the token claims.

### Step 2 — Parse path parameter

Extract `comment_id` from the URL path.
If `comment_id` is not a valid integer → return `400 Bad Request`.

### Step 3 — Fetch comment from DB

```sql
SELECT id, user_id, post_id FROM comments WHERE id = $1 AND deleted_at IS NULL LIMIT 1;
```

If no row found → return `404 Not Found` (`comment not found`).

### Step 4 — Check permission

The requesting user must be the author of the comment OR the author of the post.

```sql
SELECT id FROM posts WHERE id = $1 AND user_id = $2 LIMIT 1;
```

If the requesting user is neither the comment author nor the post author → return `403 Forbidden`.

### Step 5 — Soft delete comment

```sql
UPDATE comments SET deleted_at = now() WHERE id = $1;
```

### Step 6 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": null
}
```

# 15. GET /post/:post_id/comments

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.

### Step 2 — Parse path and query parameters

Extract `post_id` from the URL path.
If `post_id` is not a valid integer → return `400 Bad Request`.

- `page` (default: 1)
- `per_page` (default: 10)

### Step 3 — Check post exists

```sql
SELECT id FROM posts WHERE id = $1 AND deleted_at IS NULL LIMIT 1;
```

If no row found → return `404 Not Found` (`post not found`).

### Step 4 — Fetch comments from DB

```sql
SELECT c.id, c.user_id, u.username, c.content, c.created_at
FROM comments c
JOIN users u ON u.id = c.user_id
WHERE c.post_id = $1 AND c.deleted_at IS NULL
ORDER BY c.created_at ASC
LIMIT $2 OFFSET $3;
```

### Step 5 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": {
        "count": 10,
        "items": [
            {
                "post_id": 1,
                "user_id": 1,
                "username": "",
                "content": "",
                "created_at": ""
            }
        ]
    }
}
```

# 16. DELETE /post/:post_id

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.
Extract `user_id` from the token claims.

### Step 2 — Parse path parameter

Extract `post_id` from the URL path.
If `post_id` is not a valid integer → return `400 Bad Request`.

### Step 3 — Fetch post from DB

```sql
SELECT id, user_id FROM posts WHERE id = $1 AND deleted_at IS NULL LIMIT 1;
```

If no row found → return `404 Not Found` (`post not found`).

### Step 4 — Check ownership

If the requesting user's `user_id` (from JWT) does not match the post's `user_id` → return `403 Forbidden`.

### Step 5 — Soft delete post

```sql
UPDATE posts SET deleted_at = now() WHERE id = $1;
```

### Step 6 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": null
}
```

# 17. POST /users/:user_id/follow

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.
Extract `follower_id` (requesting user) from the token claims.

### Step 2 — Parse path parameter

Extract `user_id` (the user to follow) from the URL path.
If `user_id` is not a valid integer → return `400 Bad Request`.

### Step 3 — Prevent self-follow

If `follower_id` == `user_id` → return `400 Bad Request` (`cannot follow yourself`).

### Step 4 — Check target user exists

```sql
SELECT id FROM users WHERE id = $1 LIMIT 1;
```

If no row found → return `404 Not Found` (`user not found`).

### Step 5 — Insert follow

```sql
INSERT INTO follows (follower_id, following_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;
```

### Step 6 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": null
}
```

# 18. DELETE /users/:user_id/follow

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.
Extract `follower_id` from the token claims.

### Step 2 — Parse path parameter

Extract `user_id` (the user to unfollow) from the URL path.
If `user_id` is not a valid integer → return `400 Bad Request`.

### Step 3 — Delete follow

```sql
DELETE FROM follows WHERE follower_id = $1 AND following_id = $2;
```

### Step 4 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": null
}
```

# 19. GET /notifications

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.
Extract `user_id` from the token claims.

### Step 2 — Parse query parameters

- `page` (default: 1)
- `per_page` (default: 10)

### Step 3 — Fetch notifications from DB

```sql
SELECT n.id, n.action_type, n.message, n.is_read, n.created_at,
       n.actor_id, n.post_id, n.comment_id
FROM notifications n
WHERE n.user_id = $1
ORDER BY n.created_at DESC
LIMIT $2 OFFSET $3;
```

### Step 4 — Fetch actor details for each notification

```sql
SELECT id, username, full_name, avatar_path FROM users WHERE id = $1 LIMIT 1;
```

Check if the requesting user follows the actor:

```sql
SELECT id FROM follows WHERE follower_id = $1 AND following_id = $2 LIMIT 1;
```

### Step 5 — Fetch post details (if post_id is not null)

```sql
SELECT id, user_id, caption, image_path, created_at FROM posts WHERE id = $1 AND deleted_at IS NULL LIMIT 1;
```

Get post owner username:

```sql
SELECT username FROM users WHERE id = $1 LIMIT 1;
```

### Step 6 — Fetch comment details (if comment_id is not null)

```sql
SELECT c.id, u.username, c.content, c.created_at
FROM comments c
JOIN users u ON u.id = c.user_id
WHERE c.id = $1 AND c.deleted_at IS NULL LIMIT 1;
```

### Step 7 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": {
        "count": 10,
        "items": [
            {
                "notification_id": 1,
                "action_type": "",
                "message": "",
                "actor": {
                    "user_id": 1,
                    "username": "",
                    "full_name": "",
                    "avatar_path": "",
                    "is_following": false
                },
                "post": {
                    "post_id": 1,
                    "username": "",
                    "caption": "",
                    "image_path": "",
                    "created_at": ""
                },
                "comment": {
                    "comment_id": 1,
                    "username": "",
                    "content": "",
                    "created_at": ""
                },
                "is_read": false,
                "created_at": ""
            }
        ]
    }
}
```

# 20. POST /notification/:notification_id/read

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.
Extract `user_id` from the token claims.

### Step 2 — Parse path parameter

Extract `notification_id` from the URL path.
If `notification_id` is not a valid integer → return `400 Bad Request`.

### Step 3 — Check notification exists and belongs to user

```sql
SELECT id, user_id FROM notifications WHERE id = $1 LIMIT 1;
```

If no row found → return `404 Not Found` (`notification not found`).
If `user_id` does not match → return `403 Forbidden`.

### Step 4 — Mark as read

```sql
UPDATE notifications SET is_read = true, updated_at = now() WHERE id = $1;
```

### Step 5 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": null
}
```

# 21. GET /users?query=

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.

### Step 2 — Parse query parameters

- `query` is required
- `page` (default: 1)
- `per_page` (default: 10)

If `query` is empty → return `400 Bad Request`.

### Step 3 — Search users in DB

```sql
SELECT id, username, full_name, avatar_path
FROM users
WHERE username ILIKE '%' || $1 || '%' OR full_name ILIKE '%' || $1 || '%'
ORDER BY username ASC
LIMIT $2 OFFSET $3;
```

### Step 4 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": {
        "count": 10,
        "items": [
            {
                "user_id": 1,
                "username": "",
                "full_name": "",
                "avatar_path": ""
            }
        ]
    }
}
```

# 22. GET /hashtags?query=

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.

### Step 2 — Parse query parameters

- `query` is required
- `page` (default: 1)
- `per_page` (default: 10)

If `query` is empty → return `400 Bad Request`.

### Step 3 — Search hashtags in DB

```sql
SELECT h.name, COUNT(ph.post_id) AS post_count
FROM hashtags h
LEFT JOIN post_hashtags ph ON ph.hashtag_id = h.id
WHERE h.name ILIKE '%' || $1 || '%'
GROUP BY h.name
ORDER BY post_count DESC
LIMIT $2 OFFSET $3;
```

### Step 4 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": {
        "count": 10,
        "items": [
            {
                "name": "",
                "post_count": 0
            }
        ]
    }
}
```

# 23. GET /hashtags/:name/posts

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.

### Step 2 — Parse path and query parameters

Extract `name` from the URL path.
- `page` (default: 1)
- `per_page` (default: 10)

### Step 3 — Check hashtag exists

```sql
SELECT id FROM hashtags WHERE name = $1 LIMIT 1;
```

If no row found → return `404 Not Found` (`hashtag not found`).

### Step 4 — Fetch posts by hashtag

```sql
SELECT p.id, p.user_id, u.username, p.caption, p.thumbnail_path, p.likes_count, p.comments_count, p.created_at
FROM posts p
JOIN post_hashtags ph ON ph.post_id = p.id
JOIN hashtags h ON h.id = ph.hashtag_id
JOIN users u ON u.id = p.user_id
WHERE h.name = $1 AND p.deleted_at IS NULL
ORDER BY p.created_at DESC
LIMIT $2 OFFSET $3;
```

### Step 5 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": {
        "count": 10,
        "items": [
            {
                "post_id": 1,
                "user_id": 1,
                "username": "",
                "caption": "",
                "thumbnail_path": "",
                "likes_count": 0,
                "comments_count": 0,
                "created_at": ""
            }
        ]
    }
}
```

# 24. GET /profile/followers

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.
Extract `user_id` from the token claims.

### Step 2 — Parse query parameters

- `page` (default: 1)
- `per_page` (default: 10)

### Step 3 — Fetch followers from DB

```sql
SELECT u.id, u.username, u.full_name, u.avatar_path
FROM follows f
JOIN users u ON u.id = f.follower_id
WHERE f.following_id = $1
ORDER BY f.created_at DESC
LIMIT $2 OFFSET $3;
```

### Step 4 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": {
        "count": 10,
        "items": [
            {
                "user_id": 1,
                "username": "",
                "full_name": "",
                "avatar_path": ""
            }
        ]
    }
}
```

# 25. GET /profile/following

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.
Extract `user_id` from the token claims.

### Step 2 — Parse query parameters

- `page` (default: 1)
- `per_page` (default: 10)

### Step 3 — Fetch following from DB

```sql
SELECT u.id, u.username, u.full_name, u.avatar_path
FROM follows f
JOIN users u ON u.id = f.following_id
WHERE f.follower_id = $1
ORDER BY f.created_at DESC
LIMIT $2 OFFSET $3;
```

### Step 4 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": {
        "count": 10,
        "items": [
            {
                "user_id": 1,
                "username": "",
                "full_name": "",
                "avatar_path": ""
            }
        ]
    }
}
```

# 26. PUT /post/:post_id

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.
Extract `user_id` from the token claims.

### Step 2 — Parse path parameter

Extract `post_id` from the URL path.
If `post_id` is not a valid integer → return `400 Bad Request`.

### Step 3 — Parse request body

```go
type UpdatePostRequest struct {
    Caption string `json:"caption"`
}
```

### Step 4 — Fetch post from DB

```sql
SELECT id, user_id FROM posts WHERE id = $1 AND deleted_at IS NULL LIMIT 1;
```

If no row found → return `404 Not Found` (`post not found`).

### Step 5 — Check ownership

If the requesting user's `user_id` (from JWT) does not match the post's `user_id` → return `403 Forbidden`.

### Step 6 — Update post in DB

```sql
UPDATE posts SET caption = $1, updated_at = now() WHERE id = $2;
```

### Step 7 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": null
}
```

# 27. PUT /comments/:comment_id

### Step 1 — Validate access token

Read the `Authorization` header and verify the JWT token.
If missing or invalid → return `401 Unauthorized`.
Extract `user_id` from the token claims.

### Step 2 — Parse path parameter

Extract `comment_id` from the URL path.
If `comment_id` is not a valid integer → return `400 Bad Request`.

### Step 3 — Parse request body

```go
type UpdateCommentRequest struct {
    Content string `json:"content"`
}
```

### Step 4 — Validation

- `content` is required

If validation fails → return `400 Bad Request`.

### Step 5 — Fetch comment from DB

```sql
SELECT id, user_id FROM comments WHERE id = $1 AND deleted_at IS NULL LIMIT 1;
```

If no row found → return `404 Not Found` (`comment not found`).

### Step 6 — Check ownership

If the requesting user's `user_id` (from JWT) does not match the comment's `user_id` → return `403 Forbidden`.

### Step 7 — Update comment in DB

```sql
UPDATE comments SET content = $1, updated_at = now() WHERE id = $2;
```

### Step 8 — Return response

```json
{
    "status": "ok",
    "description": "The request has succeeded",
    "data": null
}
```

# 28. Like/Unlike Cache Implementation

## Goal

Buffer like and unlike events in cache before writing them to the database. A cron job runs every 1 minute to flush cached events to the DB.

## Cache structure

There are two separate cache responsibilities:

1. **User like state cache** — stores per-user like/unlike state until it is flushed to the DB.
2. **Like counter cache** — fast read path for current like count, with a timestamp used by the flush worker.

### 1. User like state cache

- **Key:** `like:{user_id}:{post_id}`
- **Type:** Redis String (integer)
- **Value:** `1` = user has liked the post; `0` = user has unliked the post
- **When written:** On every like or unlike event.
- **When read:** Sync worker reads it before flushing to the database; read path checks `is_liked`.
- **When deleted:** After successful database flush, or when a pending action is cancelled.

### 2. Like counter cache

- **Key:** `like-count:{post_id}`
- **Type:** Redis Hash
- **Fields:**
  - `count` (integer) — current like count.
  - `updated_at` (integer) — last unix timestamp (seconds) when the count changed.
- **When updated:** On every like or unlike event.
- **When read:** When returning post details or feed.
- **When deleted:** When the post is deleted or on cache miss.

## Handling like

When a user likes a post:

1. If `like:{user_id}:{post_id}` exists and equals `0`:
   - Delete the key (cancel the pending unlike).
2. Otherwise:
   - Set `like:{user_id}:{post_id}` to `1`.
3. Increment `like-count:{post_id}` `count` by 1.
4. Update `like-count:{post_id}` `updated_at` to the current unix timestamp.

```go
// Pseudocode
if redis.Get(likeKey(userID, postID)) == "0" {
    redis.Del(likeKey(userID, postID))
} else {
    redis.Set(likeKey(userID, postID), "1")
}
redis.HIncrBy(likeCountKey(postID), "count", 1)
redis.HSet(likeCountKey(postID), "updated_at", nowUnix())
```

## Handling unlike

When a user unlikes a post:

1. If `like:{user_id}:{post_id}` exists and equals `1`:
   - Delete the key (cancel the pending like).
2. Otherwise:
   - Set `like:{user_id}:{post_id}` to `0`.
3. Decrement `like-count:{post_id}` `count` by 1, but not below 0.
4. Update `like-count:{post_id}` `updated_at` to the current unix timestamp.

```go
// Pseudocode
if redis.Get(likeKey(userID, postID)) == "1" {
    redis.Del(likeKey(userID, postID))
} else {
    redis.Set(likeKey(userID, postID), "0")
}
current := redis.HGet(likeCountKey(postID), "count")
if current > 0 {
    redis.HIncrBy(likeCountKey(postID), "count", -1)
}
redis.HSet(likeCountKey(postID), "updated_at", nowUnix())
```

## Cron job — flush cache to DB every 1 minute

The worker keeps the timestamp of the last successful flush (`lastFlushAt`). On each run it processes only the records that changed since then. On the first run, `lastFlushAt` is `0`, so all cached counters are flushed.

### 1. Flush user like states

Scan all keys matching `like:*:*`.

For each key:

- Parse `user_id` and `post_id` from the key.
- Read the value.
- If value is `1`:
  ```sql
  INSERT INTO likes (user_id, post_id) VALUES ($1, $2)
  ON CONFLICT DO NOTHING;
  ```
- If value is `0`:
  ```sql
  DELETE FROM likes WHERE user_id = $1 AND post_id = $2;
  ```
- Delete the processed key.

### 2. Flush like counts

1. Scan all keys matching `like-count:*`.
2. For each key, read `count` and `updated_at`.
3. If `updated_at > lastFlushAt`:
   - Update the post row:
     ```sql
     UPDATE posts SET likes_count = $1, updated_at = now() WHERE id = $2;
     ```
4. Remember the current time as `lastFlushAt` for the next run.

```go
// Pseudocode
now := time.Now().Unix()
for _, key := range redis.Scan("like-count:*") {
    postID := parsePostID(key)
    count := redis.HGet(key, "count")
    updatedAt := redis.HGet(key, "updated_at")
    if updatedAt > lastFlushAt {
        db.Exec("UPDATE posts SET likes_count = ?, updated_at = now() WHERE id = ?", count, postID)
    }
}
lastFlushAt = now
```

## Read path

### Like count

When returning a single post, a feed, or any post list (hashtag, profile, etc.), the like count must always be served from cache.

For each `post_id`:

1. Read `count` field from `like-count:{post_id}`.
2. If cache miss:
   - Compute the count from DB:
     ```sql
     SELECT COUNT(*) FROM likes WHERE post_id = $1;
     ```
   - Write the result to cache:
     ```go
     redis.HSet(likeCountKey(postID), "count", dbCount)
     redis.HSet(likeCountKey(postID), "updated_at", 0)
     ```
3. Return the value that is now in cache.

### is_liked

1. Read `like:{user_id}:{post_id}` from cache.
   - If value is `1` → `is_liked = true`.
   - If value is `0` → `is_liked = false`.
2. If cache miss, query the DB:
   ```sql
   SELECT id FROM likes WHERE user_id = $1 AND post_id = $2 LIMIT 1;
   ```
   - If row exists → `is_liked = true`.
   - Otherwise → `is_liked = false`.