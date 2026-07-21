
# Mini Instagram — Implementation Tasks

---

## 0. Project Context (read first)

### 0.1 Architecture

Clean architecture (controller → usecase → repo), Go 1.25.

```
mini-instagram/
├── cmd/app/main.go                     # entrypoint: load config, call app.Run
├── config/config.go                    # env-based config (caarlos0/env + godotenv)
├── migrations/                         # SQL migrations (golang-migrate format, up/down)
├── internal/
│   ├── app/app.go                      # wiring: postgres, repos, usecases, router, http server
│   ├── entity/                         # domain structs (user.go, errors.go, ...) — one file per entity
│   ├── controller/restapi/
│   │   ├── router.go                   # root router, /healthz, mounts /api/v1
│   │   └── v1/
│   │       ├── router.go               # V1 struct + route registration (NewRoutes)
│   │       ├── auth.go                 # handlers grouped by domain (auth.go, user.go, post.go, ...)
│   │       └── http/                   # response envelope + status helpers
│   ├── usecase/
│   │   ├── contracts.go                # usecase interfaces + Input structs with Validate()
│   │   └── auth/auth.go                # implementations, one package per usecase group
│   ├── repo/
│   │   ├── contracts.go                # repo interfaces
│   │   └── persistent/user/user.go     # pgx implementations, one package per aggregate
│   └── worker/                         # (to create in T22) background workers, e.g. likesync
└── pkg/                                # reusable, project-agnostic packages
    ├── httpserver/                     # gin server wrapper with options
    ├── postgres/                       # pgxpool wrapper with options
    ├── logger/                         # slog-based logger
    ├── jwt/                            # JWT create/parse helpers
    ├── image/                          # image decode/resize/thumbnail helpers
    ├── storage/                        # local-disk file storage (save/delete)
    └── redis/                          # (to create) go-redis client wrapper with options
```

**Layer rules:**

- Handlers (`controller/restapi/v1`) parse/bind requests, call usecases, map errors to HTTP statuses via `handleResponse`. No business logic.
- Usecases contain business logic + input validation (`Validate()` on Input structs using `go-playground/validator`).
- Repos contain only SQL (pgx). No business logic.
- New usecase interfaces go into `internal/usecase/contracts.go`; new repo interfaces into `internal/repo/contracts.go`.
- Wire everything in `internal/app/app.go` and pass to `restapi.NewRouter`.

### 0.2 Packages (already in go.mod — do not add others unless a task says so)

| Package                                                       | Purpose                                              |
| ------------------------------------------------------------- | ---------------------------------------------------- |
| `github.com/gin-gonic/gin`                                  | HTTP framework                                       |
| `github.com/go-playground/validator/v10`                    | input validation                                     |
| `github.com/jackc/pgx/v5` (pgxpool)                         | PostgreSQL driver                                    |
| `github.com/golang-jwt/jwt/v5`                              | JWT tokens                                           |
| `golang.org/x/crypto` (bcrypt)                              | password hashing                                     |
| `github.com/caarlos0/env/v11`, `github.com/joho/godotenv` | config                                               |
| `github.com/redis/go-redis/v9`                              | Redis — rate limiting + cache (add to go.mod in T3) |

Allowed additions (only when a task requires): `golang.org/x/image/draw` or `github.com/disintegration/imaging` (thumbnails) — prefer stdlib `image` first.

### 0.3 API conventions

- Base path: `/api/v1` (already mounted). All routes below are relative to it.
- Response envelope (already implemented in `internal/controller/restapi/v1/http`):

```json
// success
{ "status": "ok", "description": "...", "data": { } }
// error
{ "status": "error", "description": "...", "errors": [ { "field": "email", "message": "..." } ] }
```

- HTTP codes: 200 OK, 400 validation, 401 unauthorized, 403 forbidden, 404 not found, 409 conflict, 429 too many requests, 500 internal.
- Pagination: query params `page` (default 1, min 1) and `per_page` (default 10, min 1, max 100). Offset = `(page-1)*per_page`. List responses: `{ "count": <total rows>, "items": [...] }` where `count` is the TOTAL matching rows (separate `COUNT(*)` query), not page size.
- Timestamps: RFC3339 UTC strings in JSON.

### 0.4 Global decisions (pre-answered questions — follow, do not ask)

- **Auth**: single JWT access token (HS256), TTL 24h, secret from env `JWT_SECRET` (add to config + `.env.example`). Claims: `user_id`, `exp`, `iat`. No refresh tokens for MVP.
- **Auth middleware**: gin middleware in `internal/controller/restapi/v1` that reads `Authorization: Bearer <token>`, validates via `pkg/jwt`, puts `user_id` (int64) into gin context. Missing/invalid → 401 envelope error. **The ONLY public API endpoints are `POST /auth/sign-up` and `POST /auth/login` — every other endpoint requires a valid token (401 otherwise).** No optional-auth middleware exists; `is_liked` / `is_following` are always computed for the authenticated caller. Non-API routes `/healthz` and the `GET /media/*` static file route stay public (infrastructure).
- **IDs**: bigint ids everywhere — NO UUIDs anywhere in this project. Users are addressed by numeric `user_id` (`users.id`), posts by numeric `post_id` (`posts.id`), comments and notifications by their bigint ids. Every `:post_id` / `:user_id` / `:comment_id` path param is a bigint; non-numeric value → 400. Follow endpoints use `user_id` (not username, overriding the sheet).
- **Image upload**: `multipart/form-data` for all image endpoints (NOT base64 JSON). File field name: `image` for posts, `avatar` for profile.
- **Storage**: local disk via `pkg/storage`. Root dir from env `STORAGE_DIR` (default `./storage`). Layout: `storage/avatars/<name>.<ext>`, `storage/posts/<name>.<ext>`, `storage/thumbnails/<name>.jpg`, where `<name>` is a random unique filename generated in Go (e.g. 16 random bytes hex-encoded via `crypto/rand` — NOT a UUID). DB stores relative paths (e.g. `posts/ab12….jpg`). Serve files via gin static route `GET /media/*` → `STORAGE_DIR` (register in `restapi/router.go`, outside `/api/v1`). No S3 for MVP.
- **Allowed image types**: jpeg, png, webp — detect by content sniffing (`http.DetectContentType`), not extension. Reject others with 400.
- **Redis**: used for rate limiting AND data cache. Create `pkg/redis` (go-redis v9 client wrapper, ping on startup), env `REDIS_URL` (add to config + `.env.example`, e.g. `redis://localhost:6379/0`). Wire in `internal/app/app.go`. **Fail-open policy**: if Redis is unavailable, log the error and continue — rate limiting allows the request, caching falls through to the DB. Redis errors must NEVER fail a user request.
- **Rate limiting**: Redis-backed, keyed by email. Keys `rl:signup:<email>` / `rl:login:<email>`; algorithm: `INCR` + `EXPIRE 60s` on first hit; count > 5 → **429**. Implement as reusable gin middleware in `internal/controller/restapi/v1/middleware.go`. Limit: 5 requests per minute per email for `/sign-up` and `/login`.
- **Passwords**: bcrypt, cost `bcrypt.DefaultCost`. Never log or return passwords.
- **Errors**: sentinel errors in `internal/entity/errors.go` (`ErrNotFound` exists; add `ErrEmailTaken`, `ErrUsernameTaken`, `ErrInvalidCredentials`, `ErrForbidden`, `ErrNotLiked`, `ErrAlreadyFollowing`, `ErrNotFollowing`, `ErrSelfFollow` as needed). Usecases return these; handlers map them to HTTP codes with `errors.Is`.
- **Migrations**: schema already exists in `migrations/20260716000001_create_instagram_tables.up.sql` (users, posts, likes, follows, comments, notifications, hashtags, post_hashtags). Do NOT recreate tables. If a task needs a schema change, add a NEW timestamped up/down migration pair.
- **Testing**: unit tests for usecases with hand-written repo fakes (implement repo interfaces in `_test.go` files). No testcontainers/mocks libraries. Handlers verified via `httptest` where cheap. Run `go build ./... && go vet ./... && go test ./...` after every task.
- **Logging**: use `pkg/logger`. Log errors at handler boundary; do not `panic`.

---

## Epic 1 — Auth & Accounts

### T1. Sign up with email/password

`POST /api/v1/auth/sign-up` — partially implemented (`v1/auth.go`, `usecase/auth`, `repo/persistent/user`). Audit the existing code against this spec and complete what is missing.

**Request (JSON):**

```json
{ "email": "", "full_name": "", "username": "", "bio": "", "password": "" }
```

Avatar is NOT part of sign-up (upload later via `PUT /profile`).

**Response `data`:** `{ "access_token": "" }` — user is logged in immediately after sign-up.

**Behavior:**

1. Validate input (see T2).
2. Check email and username uniqueness (see T2).
3. Hash password with bcrypt, insert user, generate JWT, return token.

**Acceptance:** valid sign-up returns 200 with a parseable JWT containing `user_id`; the user row exists with a bcrypt hash.

### T2. Validations and uniqueness check (sign-up)

Extends T1 — same endpoint.

**Validation rules (in `SignUpInput.Validate()`):**

- `email`: required, valid email, max 128
- `password`: required, min 8, max 72 (bcrypt limit)
- `username`: required, 3–32 chars, regex `^[a-z0-9_.]+$` (lowercase; lowercase the input before validating)
- `full_name`: required, max 64
- `bio`: optional, max 512

**Uniqueness:** check email and username before insert; on conflict return **409** with field-level error (`{"field":"email","message":"email is already taken"}`). Also handle the pg unique-violation race (pgx error code `23505`) and map it to the same 409 — the pre-check alone is not enough.

**Validation error mapping:** convert `validator.ValidationErrors` into the envelope `errors` array with lowercased field names (a small helper in the handler layer or `v1/http` package). All validated endpoints reuse this helper.

**Acceptance:** each invalid field produces a 400 with the correct `field` entry; duplicate email/username → 409.

### T3. Rate limiting by email (sign-up)

First set up Redis: add `github.com/redis/go-redis/v9`, create `pkg/redis`, add `REDIS_URL` to config/`.env.example`, wire in `app.go` (see 0.4).

Then apply the Redis-backed rate-limit middleware (see 0.4) to `POST /auth/sign-up`, keyed by the `email` field in the body (peek body via `c.ShouldBindBodyWith` or read+restore body; fallback key = client IP when email is empty/unparseable).

- Limit: 5/min per email (`INCR` + `EXPIRE`). Exceeded → **429**, description "Too Many Requests", error message "too many attempts, try again later".
- Redis down → fail-open (allow the request, log the error).

**Acceptance:** 6th sign-up attempt with the same email within a minute → 429; different emails are independent.

### T4. Log in with email/password

`POST /api/v1/auth/login`

**Request:** `{ "email": "", "password": "" }`
**Response `data`:** `{ "access_token": "" }`

**Behavior:**

- Add `Login` to auth usecase (interface method already declared in `usecase/contracts.go`): fetch user by email, `bcrypt.CompareHashAndPassword`, issue JWT.
- Wrong email OR wrong password → identical **401** response with message "invalid email or password" (never reveal which one is wrong; map both `entity.ErrNotFound` from repo and bcrypt mismatch to `entity.ErrInvalidCredentials`).
- `is_active = false` users → same 401.

**Acceptance:** correct credentials → token; wrong password and unknown email produce byte-identical error bodies.

### T5. Rate limiting by email (login)

Reuse T3 middleware on `POST /auth/login`, same limits (5/min per email). Separate key prefixes per endpoint (`rl:login:` vs `rl:signup:` — sign-up attempts must not consume login quota).

**Acceptance:** 6th login attempt for the same email in a minute → 429.

### T6. Log out

`POST /api/v1/auth/logout` — requires auth middleware.

**Decision:** stateless JWT, no server-side blacklist. The endpoint only verifies the token and returns success (`data: null`, description "Logged out"); the client discards the token. Do NOT build token storage/blacklist.

**Acceptance:** with valid token → 200; without token → 401.

### T7. Profile page — view profile

Two endpoints:

1. `GET /api/v1/profile` — **auth required**, returns the caller's own profile (`user_id` from token).
2. `GET /api/v1/users/:user_id` — **auth required**.

**Response `data` (both):**

```json
{
  "user_id": 0, "username": "", "full_name": "", "bio": "", "avatar_path": "",
  "posts_count": 0, "followers_count": 0, "following_count": 0, "is_following": false
}
```

- Counts via `COUNT(*)` subqueries in one SQL statement (posts filtered by `deleted_at IS NULL`).
- `is_following`: whether the CALLER follows this user; `false` for own profile callers.
- inactive `user_id` → 404. Non-numeric `user_id` → 400.

Also implement here (belongs to profile page):

`GET /api/v1/users/:user_id/posts` — auth required, paginated (0.3), user's posts newest-first, excluding soft-deleted.

**Response `data`:** `{ "count": 0, "items": [ { "post_id": 0, "thumbnail_path": "", "caption": "", "created_at": "" } ] }`

Create `internal/usecase/user` (interface `User` in contracts) and extend the user repo; handler file `v1/user.go`. Note: post listing needs the post repo — if T9 is not done yet, define the `repo.Post` interface + minimal `ListByUser` now and extend it in T9 (returning empty list until posts exist is acceptable).

**Acceptance:** counts are correct after follows/posts exist; `is_following` reflects the caller; 404 for missing users.

### T8. Edit profile

`PUT /api/v1/profile` — auth required. **Multipart form-data** (because of avatar):

- fields: `username`, `full_name`, `bio` (strings), `avatar` (optional file).
- All text fields required-on-update semantics: the client sends the full desired state for `username`, `full_name`, `bio` (same validation rules as T2). Email and password are NOT changeable here.
- `avatar`: **max 5 MB** (enforce via `http.MaxBytesReader` / size check before decode), jpeg/png/webp only, saved to `storage/avatars/<random-hex>.<ext>`. Old avatar file is deleted from storage after a successful update (ignore file-not-found errors, log others).
- Username uniqueness: changing to a taken username → 409 (skip check if unchanged).

**Response `data`:** `null`, description "Profile updated".

**Acceptance:** text-only update works without `avatar` part; >5MB avatar → 400 with field `avatar`; taken username → 409; old avatar file removed from disk.

---

## Epic 2 — Core MVP: Posts & Feed

### T9. Create post

`POST /api/v1/post` — auth required. **Multipart form-data**: `image` (file, required), `caption` (string, optional, max 2048).

**Behavior:**

1. Enforce **max 10 MB** image size (reject before reading the whole body where possible), allowed types per 0.4.
2. Save original to `storage/posts/<random-hex>.<ext>` (filename per 0.4); insert row into `posts` with `INSERT ... RETURNING id`.
3. If DB insert fails after the file was written, delete the file (best-effort cleanup).

**Response `data`:** `null` (per LLD), description "Post created". (Returning the post id inside `data` is NOT required — keep `null`.)

Create `internal/usecase/post`, `internal/repo/persistent/post`, handler `v1/post.go`. Add `Post` interfaces to both contracts files.

**Acceptance:** authorized multipart upload creates a row + file on disk; 11 MB file → 400; missing image part → 400 field `image`; no auth → 401.

### T10. Thumbnail generation

Extends T9 — same endpoint, same transaction of work.

- After saving the original, generate a thumbnail using `pkg/image`: max side **320px**, preserve aspect ratio, encode as JPEG quality ~80 regardless of source format.
- Save to `storage/thumbnails/<random-hex>.jpg`, store relative path in `posts.thumbnail_path`.
- **Decision:** generate synchronously in the request (no queue/goroutine — MVP). If thumbnail generation fails, fail the whole request and clean up saved files.

**Acceptance:** created post has a non-empty `thumbnail_path` and the file exists; thumbnail's largest dimension ≤ 320.

### T11. Home feed

`GET /api/v1/feed` — auth required, paginated (0.3).

Posts authored by users the caller **follows** (NOT own posts — decision), `deleted_at IS NULL`, ordered by `posts.created_at DESC, posts.id DESC` (tie-breaker). Single SQL with JOIN on `follows`; `count` = total matching posts.

**Response `data`:**

```json
{ "count": 0, "items": [ { "user_id": 0, "username": "", "post_id": 0, "caption": "",
  "image_path": "", "likes_count": 0, "comments_count": 0, "created_at": "" } ] }
```

`likes_count`/`comments_count` come from the denormalized columns. Following nobody → `{"count":0,"items":[]}` (200, not 404). Note: needs the `follows` repo — if T18 is not done, add a minimal `repo.Follow` interface now.

**Acceptance:** feed shows only followed users' posts, newest first; pagination boundaries correct; empty feed → empty items.

### T12. Like a post

`POST /api/v1/post/:post_id/like` — auth required (`:post_id` = bigint post id).

In ONE transaction: insert into `likes` + increment `posts.like_count`.

- Post missing or soft-deleted → 404.
- Already liked → **idempotent success**: return 200 with `data: null`, do NOT insert and do NOT increment the counter. Implement via `INSERT ... ON CONFLICT (user_id, post_id) DO NOTHING` — if 0 rows inserted, skip the counter update and commit (no error to the client).
- Liking own post is allowed.

Create `internal/repo/persistent/like` (or fold into post repo — **decision: separate `repo.Like` interface, implemented in the post repo package is fine; keep interfaces separate**). Usecase: extend `usecase/post`.

**Response `data`:** `null`.

**Acceptance:** like increments `like_count` by exactly 1; repeating the same like any number of times → always 200 and counter stays unchanged.

### T13. Unlike a post

`DELETE /api/v1/post/:post_id/like` — auth required.

In ONE transaction: delete like row + decrement `like_count`. If no like row existed → **409** message "post is not liked" (decision; do not decrement). Missing post → 404.

**Acceptance:** unlike after like restores the original counter; unlike without like → 409.

### T14. View a single post

`GET /api/v1/post/:post_id` — auth required.

**Response `data`:**

```json
{ "post_id": 0, "user_id": 0, "username": "", "caption": "", "image_path": "",
  "likes_count": 0, "comments_count": 0, "created_at": "", "is_liked": false }
```

- `is_liked` = caller has liked it.
- Soft-deleted or unknown id → 404. Non-numeric `post_id` → 400 with field `post_id`.

**Acceptance:** counts match reality; `is_liked` correct per caller; deleted post → 404.

### T15. Comment on a post

Two endpoints:

1. `POST /api/v1/post/:post_id/comments` — auth required. Body: `{ "content": "" }`, required, 1–2048 chars after trimming whitespace. In ONE transaction: insert comment + increment `posts.comment_count`. Missing/deleted post → 404. Response `data`: `null`.
2. `GET /api/v1/post/:post_id/comments` — auth required, paginated (0.3), ordered `created_at ASC, id ASC` (oldest first — decision), excluding soft-deleted comments.

**List response `data`:** `{ "count": 0, "items": [ { "comment_id": 0, "post_id": 0, "user_id": 0, "username": "", "content": "", "created_at": "" } ] }` (include `comment_id` — needed for T16 even though LLD omits it).

Create `internal/repo/persistent/comment` + `repo.Comment` interface; usecase `usecase/comment` (or extend post usecase — **decision: separate `usecase/comment` package**). Handler `v1/comment.go`.

**Acceptance:** comment increments `comment_count`; empty/whitespace content → 400; list pagination + ordering correct.

### T16. Delete a comment

`DELETE /api/v1/comments/:comment_id` — auth required (`:comment_id` = bigint id).

- Allowed for the **comment author OR the author of the post** the comment belongs to. Anyone else → **403**.
- Soft delete (`deleted_at = NOW()`) + decrement `posts.comment_count` in one transaction.
- Already-deleted or unknown comment → 404.

**Acceptance:** author and post owner can delete; third user → 403; counter decrements once; second delete → 404.

### T17. Delete a post

`DELETE /api/v1/post/:post_id` — auth required.

- Only the post owner. Someone else's post → **403** (post exists but not yours), unknown/already-deleted → 404. Check existence BEFORE ownership so 403 vs 404 is deterministic.
- Soft-delete the post row (`deleted_at = NOW()`).
- **Storage cleanup:** delete the original image and thumbnail files from disk AFTER the DB update commits (best-effort: log failures, do not fail the request). Likes/comments rows stay (post is filtered out everywhere by `deleted_at IS NULL`).

**Response `data`:** `null`.

**Acceptance:** owner deletes → post disappears from feed/profile/single-view; files removed from disk; non-owner → 403.

---

## Epic 3 — Social Graph

### T18. Follow / unfollow

Endpoints (**decision: address by user_id, not username**):

1. `POST /api/v1/users/:user_id/follow` — auth required. Insert into `follows (follower_id=caller, following_id=:user_id)`.
   - `:user_id` == caller → **400** message "cannot follow yourself" (validate BEFORE any DB call).
   - Target user missing/inactive → 404.
   - Already following (`23505` on `uq_follows_follower_following`) → **409** "already following".
2. `DELETE /api/v1/users/:user_id/follow` — auth required. Delete the row; not following → **409** "not following"; missing target → 404.

No follower-count denormalization (T7 computes counts live). Response `data`: `null` for both.

Create `internal/repo/persistent/follow` + `repo.Follow` (may already exist minimally from T11 — extend it); usecase `usecase/user` extension or `usecase/follow` — **decision: put Follow/Unfollow into `usecase/user`**.

**Acceptance:** follow then T7 counts update; self-follow → 400; duplicate → 409; unfollow reverses everything.

---

## Epic 4 — Engagement & Discovery

### T19. Notifications

Uses the existing `notifications` table (`action_type` enum: `like|comment|follow`).

**Creation (side effects added to T12/T15/T18 usecases):**

- like → notify post owner (`actor_id`=liker, `post_id` set) — ONLY when a new like row was actually inserted (repeated idempotent likes from T12 must NOT create duplicate notifications)
- comment → notify post owner (`comment_id` + `post_id` set)
- follow → notify followed user (`actor_id`=follower)
- **Never notify yourself** (liking/commenting your own post → no notification).
- **Decision:** create the notification in the SAME transaction as the triggering write. `message` column: leave NULL (clients render from `action_type`). No notification deletion on unlike/uncomment (MVP).

**Endpoints:**

1. `GET /api/v1/notifications` — auth required, paginated, caller's notifications newest first.
   **Response `data`:** `{ "count": 0, "items": [ { "notification_id": 0, "action_type": "like", "actor_id": 0, "actor_username": "", "post_id": 0, "is_read": false, "created_at": "" } ] }`
2. `PUT /api/v1/notifications/:notification_id/read` — auth required, sets `is_read=true`. Not the caller's notification → 404 (not 403 — don't leak existence). Already read → 200 idempotent. Response `data`: `null`.

Create `repo.Notification` + `internal/repo/persistent/notification`, `usecase/notification`, handler `v1/notification.go`.

**Acceptance:** like/comment/follow each produce exactly one notification for the right user; self-actions produce none; mark-read is idempotent and owner-scoped.

### T20. Search users

`GET /api/v1/search/users?q=<term>` — auth required, paginated (0.3).

- `q`: required, 1–32 chars after trim; missing/empty → 400 field `q`.
- Match: case-insensitive **substring** on username — `WHERE username ILIKE '%' || $1 || '%'`, escape `%`/`_` in user input. Order: exact match first, then prefix matches, then alphabetical (`ORDER BY (username = q) DESC, (username LIKE q || '%') DESC, username ASC`). Only `is_active = true`.

**Response `data`:** `{ "count": 0, "items": [ { "user_id": 0, "username": "", "full_name": "", "avatar_path": "" } ] }`

Extend user repo + `usecase/user`; route in `v1/user.go`.

**Acceptance:** substring matching works; exact match ranks first; `%` in query doesn't match everything; pagination correct.

### T21. Hashtags

Uses existing `hashtags` + `post_hashtags` tables.

**Parsing (extends T9 create-post usecase):**

- Extract hashtags from caption with regex `#([\p{L}\p{N}_]+)` — lowercase them, dedupe, cap at first **30** tags, each max 64 chars (skip longer ones).
- Upsert: `INSERT INTO hashtags(name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id`, then insert `post_hashtags` pairs (`ON CONFLICT DO NOTHING`). Same transaction as post creation.

**Search endpoint:** `GET /api/v1/search/posts?tag=<name>` — auth required, paginated.

- `tag`: required, leading `#` stripped if present, lowercased; empty → 400.
- Posts joined through `post_hashtags`, `deleted_at IS NULL`, newest first.
- Unknown tag → 200 with empty items.

**Response `data`:** `{ "count": 0, "items": [ { "post_id": 0, "user_id": 0, "username": "", "thumbnail_path": "", "caption": "", "likes_count": 0, "comments_count": 0, "created_at": "" } ] }`

Create `repo.Hashtag` + `internal/repo/persistent/hashtag`; parsing lives in `usecase/post`; search in `usecase/post` too; route in `v1/post.go`.

**Acceptance:** caption "sunset #Beach #beach_life" creates tags `beach`, `beach_life` linked to the post; searching `beach` (or `#Beach`) returns it; deleted posts excluded.

---

## Epic 5 — Performance

### T22. Like/unlike cache (write buffer + cron flush)

Caching scope is **likes only** — nothing else is cached. Like and unlike events are buffered in Redis instead of being written to Postgres directly; a background worker flushes them to the DB every **1 minute**. 

#### Cache structure

Two responsibilities:

**1. User like state** — pending like/unlike events per user until flushed.

- Key: `like:{user_id}:{post_id}`, type: Redis String
- Value: `"1"` = liked, `"0"` = unliked
- Written on every like/unlike event; read by the flush worker and the `is_liked` read path; deleted after a successful flush or when a pending action is cancelled.
- No TTL.

**2. Like counter** — fast read path for the current count.

- Key: `like-count:{post_id}`, type: Redis Hash, fields: `count` (int), `updated_at` (unix seconds, last change)
- Updated on every like/unlike event; read whenever a post's `likes_count` is returned; deleted on post delete (add `DEL like-count:{post_id}` to T17).

#### Handling like (replaces T12 DB tx)

Guard first (preserves T12 idempotency): resolve current `is_liked` via the read path below; if already `true` → return 200, change nothing. Otherwise:

1. If `like:{user_id}:{post_id}` exists and equals `"0"` → `DEL` the key (cancel the pending unlike).
2. Else → `SET like:{user_id}:{post_id} "1"`.
3. `HINCRBY like-count:{post_id} count 1` (initialize from DB via the read path if the hash is missing).
4. `HSET like-count:{post_id} updated_at <now unix>`.

Post existence check (404 for missing/soft-deleted) stays before any of this.

#### Handling unlike (replaces T13 DB tx)

Guard: resolve `is_liked`; if `false` → **409** "post is not liked" (T13 behavior unchanged). Otherwise:

1. If `like:{user_id}:{post_id}` exists and equals `"1"` → `DEL` the key (cancel the pending like).
2. Else → `SET like:{user_id}:{post_id} "0"`.
3. Decrement `count` by 1, but never below 0 (check current value first).
4. `HSET like-count:{post_id} updated_at <now unix>`.

#### Flush worker — every 1 minute

Implement in `internal/worker/likesync` (new package), started as a goroutine with a `time.Ticker` from `app.go`, stopped gracefully on shutdown. The worker keeps `lastFlushAt` (unix seconds) in memory; `0` on first run so everything is flushed.

**Step 1 — flush like states:** `SCAN` keys matching `like:*:*` (use SCAN, never KEYS). For each: parse `user_id`/`post_id`, read value:
- `"1"` → `INSERT INTO likes (user_id, post_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;`
- `"0"` → `DELETE FROM likes WHERE user_id = $1 AND post_id = $2;`
- On success `DEL` the processed key; on DB error keep the key (retried next run).

**Step 2 — flush counters:** `SCAN` `like-count:*`; for each read `count` and `updated_at`; if `updated_at > lastFlushAt`:

```sql
UPDATE posts SET like_count = $1, updated_at = now() WHERE id = $2;
```

(column is `like_count` in the schema). Then set `lastFlushAt = now` (taken BEFORE the scan started).

#### Read path

**Like count** — every place a post's `likes_count` is returned (T11 feed, T14 single post, T21 hashtag search) must serve it from cache:

1. `HGET like-count:{post_id} count` → hit: return it.
2. Miss: `SELECT COUNT(*) FROM likes WHERE post_id = $1` (NOTE: from now on the `likes` table is the source of truth, not `posts.like_count`), then `HSET count=<dbCount> updated_at=0` and return it. For list endpoints batch this (pipeline `HGET`s, single grouped `COUNT` query for the misses).

**is_liked** (T14):

1. `GET like:{user_id}:{post_id}` → `"1"` = true, `"0"` = false.
2. Miss: `SELECT id FROM likes WHERE user_id = $1 AND post_id = $2 LIMIT 1` → exists = true.

#### Decisions

- **Redis down** → fall back to the old direct-DB transactional path from T12/T13 for that request (fail-open, consistent with 0.4); read paths fall back to the DB queries above.
- **Notifications (T19)**: the like notification is created at EVENT time (when the state transition to liked is accepted), not at flush time; the idempotency guard already prevents duplicates.
- Layering: cache logic lives in the **usecase layer** (`usecase/post`) behind a small cache interface; the flush worker talks to `repo.Like`/`repo.Post` directly.

**Acceptance:** like → `likes_count` reflects immediately on next read (from cache) while the `likes` row appears only after the next flush (≤1 min); like→unlike before a flush leaves no pending keys and no DB write; counter never goes below 0; killing Redis keeps like/unlike working via the DB fallback; worker retries failed flushes without losing events.
