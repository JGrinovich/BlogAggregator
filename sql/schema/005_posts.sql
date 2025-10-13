-- sql
-- +goose Up

CREATE TABLE posts (id UUID PRIMARY KEY,
created_at TIMESTAMPTZ NOT NULL,
updated_at TIMESTAMPTZ NOT NULL,
title TEXT NOT NULL,
url TEXT NOT NULL,
description TEXT,
published_at TIMESTAMPTZ,
feed_id UUID NOT NULL,
CONSTRAINT posts_url_unique UNIQUE (url),
CONSTRAINT fk_feedid
FOREIGN KEY (feed_id)
REFERENCES feeds(id)
ON DELETE CASCADE);

-- +goose Down
DROP TABLE posts;