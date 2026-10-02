CREATE TABLE IF NOT EXISTS users (
    id           BIGSERIAL PRIMARY KEY,
    username     VARCHAR(64) UNIQUE NOT NULL,
    email        VARCHAR(255) UNIQUE,
    avatar_url   TEXT,
    github_id    BIGINT UNIQUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS repositories (
    id           BIGSERIAL PRIMARY KEY,
    namespace    VARCHAR(64) NOT NULL,
    name         VARCHAR(128) NOT NULL,
    description  TEXT,
    readme       TEXT,
    owner_id     BIGINT NOT NULL REFERENCES users(id),
    is_public    BOOLEAN NOT NULL DEFAULT TRUE,
    stars        INT NOT NULL DEFAULT 0,
    pulls        BIGINT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(namespace, name)
);
CREATE INDEX IF NOT EXISTS idx_repos_ns_name ON repositories(namespace, name);

CREATE TABLE IF NOT EXISTS tags (
    id           BIGSERIAL PRIMARY KEY,
    repo_id      BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    tag          VARCHAR(64) NOT NULL,
    digest       VARCHAR(128),
    size_bytes   BIGINT,
    os           VARCHAR(16) NOT NULL,
    arch         VARCHAR(16) NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(repo_id, tag, os, arch)
);

CREATE TABLE IF NOT EXISTS sources (
    id           BIGSERIAL PRIMARY KEY,
    tag_id       BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    type         VARCHAR(16) NOT NULL,
    url          TEXT NOT NULL,
    priority     INT NOT NULL DEFAULT 100,
    region       VARCHAR(16),
    digest       VARCHAR(128),
    size_bytes   BIGINT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sources_tag ON sources(tag_id);

CREATE TABLE IF NOT EXISTS sessions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   VARCHAR(128) NOT NULL,
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
