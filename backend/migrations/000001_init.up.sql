-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Users
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username      VARCHAR(64) NOT NULL UNIQUE,
    email         VARCHAR(255) UNIQUE,
    display_name  VARCHAR(128),
    password_hash VARCHAR(255),
    role          VARCHAR(16) NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Passkey credentials
CREATE TABLE passkey_credentials (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id BYTEA NOT NULL UNIQUE,
    public_key    BYTEA NOT NULL,
    sign_count    BIGINT NOT NULL DEFAULT 0,
    name          VARCHAR(128),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Refresh tokens
CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Invite links
CREATE TABLE invite_links (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token      VARCHAR(64) NOT NULL UNIQUE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    used_by    UUID REFERENCES users(id),
    max_uses   INT NOT NULL DEFAULT 1,
    use_count  INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- System settings
CREATE TABLE system_settings (
    key        VARCHAR(128) PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO system_settings (key, value) VALUES ('setup_complete', 'false');

-- Tags
CREATE TABLE tags (
    id   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(64) NOT NULL UNIQUE
);

-- Units
CREATE TABLE units (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name_en         VARCHAR(128) NOT NULL,
    name_de         VARCHAR(128),
    abbreviation_en VARCHAR(32),
    abbreviation_de VARCHAR(32),
    plural_en       VARCHAR(128),
    plural_de       VARCHAR(128),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unit aliases
CREATE TABLE unit_aliases (
    id      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    unit_id UUID NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    alias   VARCHAR(128) NOT NULL,
    UNIQUE(unit_id, alias)
);

-- Ingredients
CREATE TABLE ingredients (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name_en         VARCHAR(256) NOT NULL,
    name_de         VARCHAR(256),
    plural_en       VARCHAR(256),
    plural_de       VARCHAR(256),
    default_unit_id UUID REFERENCES units(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ingredient aliases
CREATE TABLE ingredient_aliases (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ingredient_id UUID NOT NULL REFERENCES ingredients(id) ON DELETE CASCADE,
    alias         VARCHAR(256) NOT NULL,
    UNIQUE(ingredient_id, alias)
);

-- Ingredient-unit usage stats
CREATE TABLE ingredient_unit_stats (
    ingredient_id UUID NOT NULL REFERENCES ingredients(id) ON DELETE CASCADE,
    unit_id       UUID NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    use_count     INT NOT NULL DEFAULT 1,
    PRIMARY KEY (ingredient_id, unit_id)
);

-- Recipes
CREATE TABLE recipes (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    author_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name          VARCHAR(256) NOT NULL,
    description   TEXT,
    photo_path    VARCHAR(512),
    work_time_min INT,
    servings      INT,
    is_public     BOOLEAN NOT NULL DEFAULT false,
    source_url    VARCHAR(1024),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Recipe tags (junction)
CREATE TABLE recipe_tags (
    recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    tag_id    UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (recipe_id, tag_id)
);

-- Recipe steps
CREATE TABLE recipe_steps (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    recipe_id    UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    step_order   INT NOT NULL,
    description  TEXT NOT NULL,
    duration_min INT,
    UNIQUE(recipe_id, step_order)
);

-- Step ingredients
CREATE TABLE step_ingredients (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    step_id       UUID NOT NULL REFERENCES recipe_steps(id) ON DELETE CASCADE,
    ingredient_id UUID NOT NULL REFERENCES ingredients(id),
    unit_id       UUID REFERENCES units(id),
    amount        NUMERIC(10,3),
    notes         TEXT
);

-- Share links
CREATE TABLE share_links (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token      VARCHAR(64) NOT NULL UNIQUE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipe_id  UUID REFERENCES recipes(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- MCP tokens
CREATE TABLE mcp_tokens (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    name       VARCHAR(128),
    scopes     TEXT[] NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_recipes_author     ON recipes(author_id);
CREATE INDEX idx_recipes_public     ON recipes(is_public);
CREATE INDEX idx_recipe_steps       ON recipe_steps(recipe_id);
CREATE INDEX idx_step_ingredients   ON step_ingredients(step_id);
CREATE INDEX idx_ingredients_en     ON ingredients USING gin(name_en gin_trgm_ops);
CREATE INDEX idx_ingredients_de     ON ingredients USING gin(name_de gin_trgm_ops);
CREATE INDEX idx_units_en           ON units USING gin(name_en gin_trgm_ops);
