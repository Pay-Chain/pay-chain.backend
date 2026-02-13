CREATE TABLE IF NOT EXISTS teams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(120) NOT NULL,
    role VARCHAR(120) NOT NULL,
    bio TEXT NOT NULL,
    image_url TEXT NOT NULL,
    github_url TEXT,
    twitter_url TEXT,
    linkedin_url TEXT,
    display_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_teams_display_order ON teams(display_order);
CREATE INDEX IF NOT EXISTS idx_teams_is_active ON teams(is_active);
