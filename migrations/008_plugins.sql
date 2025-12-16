-- Plugin and Publishing Tables

CREATE TABLE IF NOT EXISTS publishing_channels (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    config TEXT,
    active BOOLEAN DEFAULT 1,
    created_at INTEGER,
    updated_at INTEGER
);

CREATE TABLE IF NOT EXISTS publish_jobs (
    id TEXT PRIMARY KEY,
    node_id TEXT,
    version_id TEXT,
    channel_id TEXT,
    status TEXT DEFAULT 'queued',
    progress INTEGER DEFAULT 0,
    result TEXT,
    error TEXT,
    created_at INTEGER,
    completed_at INTEGER
);

CREATE TABLE IF NOT EXISTS git_commits (
    id TEXT PRIMARY KEY,
    node_id TEXT,
    message TEXT,
    created_at INTEGER
);

CREATE TABLE IF NOT EXISTS ipfs_content (
    id TEXT PRIMARY KEY,
    hash TEXT UNIQUE,
    name TEXT,
    content TEXT,
    pinned BOOLEAN DEFAULT 0,
    created_at INTEGER
);

CREATE TABLE IF NOT EXISTS ipfs_publications (
    id TEXT PRIMARY KEY,
    version_id TEXT,
    node_id TEXT,
    ipfs_hash TEXT,
    created_at INTEGER
);

CREATE TABLE IF NOT EXISTS dns_records (
    id TEXT PRIMARY KEY,
    domain TEXT,
    hostname TEXT,
    record_type TEXT,
    address TEXT,
    ttl TEXT,
    created_at INTEGER
);

CREATE TABLE IF NOT EXISTS media_conversions (
    id TEXT PRIMARY KEY,
    input_path TEXT,
    output_path TEXT,
    format TEXT,
    quality TEXT,
    created_at INTEGER
);

CREATE TABLE IF NOT EXISTS game_embeds (
    id TEXT PRIMARY KEY,
    node_id TEXT,
    game_id TEXT,
    title TEXT,
    description TEXT,
    embed_code TEXT,
    created_at INTEGER
);

CREATE TABLE IF NOT EXISTS game_scores (
    id TEXT PRIMARY KEY,
    game_id TEXT,
    player_id TEXT,
    score INTEGER,
    timestamp INTEGER,
    metadata TEXT
);

CREATE TABLE IF NOT EXISTS portfolio_games (
    id TEXT PRIMARY KEY,
    node_id TEXT,
    game_id TEXT,
    showcase BOOLEAN DEFAULT 0,
    created_at INTEGER
);

CREATE TABLE IF NOT EXISTS configs (
    id TEXT PRIMARY KEY,
    key TEXT UNIQUE,
    value TEXT,
    updated_at INTEGER
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_publish_jobs_channel ON publish_jobs(channel_id);
CREATE INDEX IF NOT EXISTS idx_publish_jobs_node ON publish_jobs(node_id);
CREATE INDEX IF NOT EXISTS idx_ipfs_hash ON ipfs_content(hash);
CREATE INDEX IF NOT EXISTS idx_game_scores_game ON game_scores(game_id);
CREATE INDEX IF NOT EXISTS idx_game_scores_player ON game_scores(player_id);
CREATE INDEX IF NOT EXISTS idx_dns_domain ON dns_records(domain);
CREATE INDEX IF NOT EXISTS idx_configs_key ON configs(key);
