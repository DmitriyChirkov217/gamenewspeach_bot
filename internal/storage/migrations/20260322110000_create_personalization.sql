-- +goose Up
-- +goose StatementBegin
CREATE TABLE users
(
    telegram_user_id BIGINT PRIMARY KEY,
    chat_id          BIGINT       NOT NULL,
    username         VARCHAR(255) NOT NULL DEFAULT '',
    first_name       VARCHAR(255) NOT NULL DEFAULT '',
    created_at       TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE TABLE article_tags
(
    article_id  BIGINT           NOT NULL,
    tag         VARCHAR(120)     NOT NULL,
    weight      DOUBLE PRECISION NOT NULL DEFAULT 1,
    created_at  TIMESTAMP        NOT NULL DEFAULT NOW(),
    PRIMARY KEY (article_id, tag),
    CONSTRAINT fk_article_tags_article_id
        FOREIGN KEY (article_id)
            REFERENCES articles (id)
            ON DELETE CASCADE
);

CREATE TABLE article_deliveries
(
    user_id      BIGINT    NOT NULL,
    article_id   BIGINT    NOT NULL,
    message_id   BIGINT    NOT NULL,
    delivered_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, article_id),
    CONSTRAINT fk_article_deliveries_user_id
        FOREIGN KEY (user_id)
            REFERENCES users (telegram_user_id)
            ON DELETE CASCADE,
    CONSTRAINT fk_article_deliveries_article_id
        FOREIGN KEY (article_id)
            REFERENCES articles (id)
            ON DELETE CASCADE
);

CREATE TABLE article_reactions
(
    user_id     BIGINT    NOT NULL,
    article_id  BIGINT    NOT NULL,
    reaction    SMALLINT  NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, article_id),
    CONSTRAINT fk_article_reactions_user_id
        FOREIGN KEY (user_id)
            REFERENCES users (telegram_user_id)
            ON DELETE CASCADE,
    CONSTRAINT fk_article_reactions_article_id
        FOREIGN KEY (article_id)
            REFERENCES articles (id)
            ON DELETE CASCADE,
    CONSTRAINT chk_article_reactions_reaction
        CHECK (reaction IN (-1, 1))
);

CREATE TABLE user_tag_scores
(
    user_id     BIGINT           NOT NULL,
    tag         VARCHAR(120)     NOT NULL,
    score       DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at  TIMESTAMP        NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP        NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, tag),
    CONSTRAINT fk_user_tag_scores_user_id
        FOREIGN KEY (user_id)
            REFERENCES users (telegram_user_id)
            ON DELETE CASCADE
);

CREATE INDEX idx_article_tags_tag ON article_tags (tag);
CREATE INDEX idx_articles_published_created ON articles (published_at DESC, created_at DESC);
CREATE INDEX idx_article_deliveries_user_id ON article_deliveries (user_id);
CREATE INDEX idx_user_tag_scores_user_id ON user_tag_scores (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_tag_scores;
DROP TABLE IF EXISTS article_reactions;
DROP TABLE IF EXISTS article_deliveries;
DROP TABLE IF EXISTS article_tags;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
