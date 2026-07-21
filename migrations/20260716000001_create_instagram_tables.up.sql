CREATE TYPE notification_action AS ENUM ('like', 'comment', 'follow');

CREATE TABLE IF NOT EXISTS users (
    id bigserial PRIMARY KEY,
    email varchar(128) NOT NULL UNIQUE,
    password varchar(255) NOT NULL,
    username varchar(32) NOT NULL UNIQUE,
    avatar_path varchar(512),
    bio varchar(512),
    full_name varchar(64),
    is_active boolean DEFAULT true,
    created_at timestamp DEFAULT NOW(),
    updated_at timestamp DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS posts (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL,
    image_path varchar(128) NOT NULL,
    thumbnail_path varchar(512),
    caption varchar(2048),
    like_count bigint NOT NULL DEFAULT 0 CHECK (like_count >= 0),
    comment_count bigint NOT NULL DEFAULT 0 CHECK (comment_count >= 0),
    deleted_at timestamp,
    created_at timestamp DEFAULT NOW(),
    updated_at timestamp DEFAULT NOW(),
    CONSTRAINT fk_posts_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS likes (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL,
    post_id bigint NOT NULL,
    created_at timestamp DEFAULT NOW(),
    CONSTRAINT fk_likes_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_likes_post_id FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    CONSTRAINT uq_likes_user_id_post_id UNIQUE (user_id, post_id)
);

CREATE INDEX IF NOT EXISTS idx_likes_post_id ON likes (post_id);

CREATE TABLE IF NOT EXISTS follows (
    id bigserial PRIMARY KEY,
    follower_id bigint NOT NULL,
    following_id bigint NOT NULL,
    created_at timestamp DEFAULT NOW(),
    CONSTRAINT fk_follows_follower_id FOREIGN KEY (follower_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_follows_following_id FOREIGN KEY (following_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT uq_follows_follower_following UNIQUE (follower_id, following_id)
);

CREATE INDEX IF NOT EXISTS idx_follows_following_id ON follows (following_id);

CREATE TABLE IF NOT EXISTS comments (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL,
    post_id bigint NOT NULL,
    content varchar(2048) NOT NULL,
    deleted_at timestamp,
    created_at timestamp DEFAULT NOW(),
    updated_at timestamp DEFAULT NOW(),
    CONSTRAINT fk_comments_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_comments_post_id FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_comments_post_id_created_at ON comments (post_id, created_at);

CREATE TABLE IF NOT EXISTS notifications (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL,
    actor_id bigint,
    action_type notification_action NOT NULL,
    post_id bigint,
    comment_id bigint,
    message varchar(512),
    is_read boolean DEFAULT false,
    created_at timestamp DEFAULT NOW(),
    updated_at timestamp DEFAULT NOW(),
    CONSTRAINT fk_notifications_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_notifications_actor_id FOREIGN KEY (actor_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_notifications_post_id FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    CONSTRAINT fk_notifications_comment_id FOREIGN KEY (comment_id) REFERENCES comments(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id_read_created ON notifications (user_id, is_read, created_at);

CREATE TABLE IF NOT EXISTS hashtags (
    id bigserial PRIMARY KEY,
    name varchar(64) NOT NULL UNIQUE,
    created_at timestamp DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS post_hashtags (
    post_id bigint NOT NULL,
    hashtag_id bigint NOT NULL,
    CONSTRAINT fk_post_hashtags_post_id FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    CONSTRAINT fk_post_hashtags_hashtag_id FOREIGN KEY (hashtag_id) REFERENCES hashtags(id) ON DELETE CASCADE,
    CONSTRAINT uq_post_hashtags_post_id_hashtag_id UNIQUE (post_id, hashtag_id)
);

CREATE INDEX IF NOT EXISTS idx_post_hashtags_hashtag_id ON post_hashtags (hashtag_id);
