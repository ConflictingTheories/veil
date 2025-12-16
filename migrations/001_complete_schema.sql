-- Complete Veil Database Schema
-- Single migration with all tables

-- Sites
CREATE TABLE IF NOT EXISTS sites (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT DEFAULT 'project',
    created_at INTEGER NOT NULL,
    modified_at INTEGER NOT NULL
);

-- Nodes (all content types)
CREATE TABLE IF NOT EXISTS nodes (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    parent_id TEXT,
    site_id TEXT,
    path TEXT NOT NULL,
    title TEXT,
    content TEXT,
    slug TEXT,
    canonical_uri TEXT,
    body TEXT,
    metadata TEXT,
    status TEXT DEFAULT 'draft',
    visibility TEXT DEFAULT 'public',
    mime_type TEXT,
    created_at INTEGER NOT NULL,
    modified_at INTEGER NOT NULL,
    deleted_at INTEGER,
    FOREIGN KEY (parent_id) REFERENCES nodes(id),
    FOREIGN KEY (site_id) REFERENCES sites(id)
);

-- Versions
CREATE TABLE IF NOT EXISTS versions (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    version_number INTEGER NOT NULL,
    content TEXT,
    title TEXT,
    status TEXT DEFAULT 'draft',
    published_at INTEGER,
    created_at INTEGER NOT NULL,
    modified_at INTEGER NOT NULL,
    is_current INTEGER DEFAULT 0,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

-- Users
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE,
    password_hash TEXT,
    created_at INTEGER NOT NULL
);

-- Permissions
CREATE TABLE IF NOT EXISTS user_permissions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    node_id TEXT NOT NULL,
    permission TEXT NOT NULL,
    granted_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

-- Node Visibility
CREATE TABLE IF NOT EXISTS node_visibility (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    visibility TEXT NOT NULL,
    shared_token TEXT,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

-- Media
CREATE TABLE IF NOT EXISTS media (
    id TEXT PRIMARY KEY,
    node_id TEXT,
    filename TEXT,
    original_filename TEXT,
    mime_type TEXT,
    file_size INTEGER,
    hash TEXT,
    storage_url TEXT,
    uploaded_by TEXT,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

-- Media Library
CREATE TABLE IF NOT EXISTS media_library (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    media_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (media_id) REFERENCES media(id)
);

-- Media Conversions
CREATE TABLE IF NOT EXISTS media_conversions (
    id TEXT PRIMARY KEY,
    source_media_id TEXT NOT NULL,
    target_media_id TEXT NOT NULL,
    conversion_type TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (source_media_id) REFERENCES media(id),
    FOREIGN KEY (target_media_id) REFERENCES media(id)
);

-- References (links between nodes)
CREATE TABLE IF NOT EXISTS node_references (
    id TEXT PRIMARY KEY,
    source_node_id TEXT NOT NULL,
    target_node_id TEXT NOT NULL,
    link_type TEXT,
    link_text TEXT,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (source_node_id) REFERENCES nodes(id),
    FOREIGN KEY (target_node_id) REFERENCES nodes(id)
);

-- URIs
CREATE TABLE IF NOT EXISTS node_uris (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    uri TEXT NOT NULL UNIQUE,
    is_primary INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

-- Tags
CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    color TEXT
);

-- Node Tags
CREATE TABLE IF NOT EXISTS node_tags (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    tag_id TEXT NOT NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id),
    UNIQUE(node_id, tag_id)
);

-- Blog Posts
CREATE TABLE IF NOT EXISTS blog_posts (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    slug TEXT UNIQUE,
    excerpt TEXT,
    publish_date INTEGER,
    category TEXT,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

-- Citations
CREATE TABLE IF NOT EXISTS citations (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    citation_key TEXT,
    authors TEXT,
    title TEXT,
    year INTEGER,
    publication TEXT,
    url TEXT,
    format TEXT,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

-- Publishing Channels
CREATE TABLE IF NOT EXISTS publishing_channels (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    config TEXT,
    active INTEGER DEFAULT 1,
    created_at INTEGER NOT NULL
);

-- Publish Jobs
CREATE TABLE IF NOT EXISTS publish_jobs (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    version_id TEXT,
    channel_id TEXT NOT NULL,
    status TEXT NOT NULL,
    progress INTEGER DEFAULT 0,
    result TEXT,
    error TEXT,
    created_at INTEGER NOT NULL,
    completed_at INTEGER,
    FOREIGN KEY (node_id) REFERENCES nodes(id),
    FOREIGN KEY (channel_id) REFERENCES publishing_channels(id)
);

-- Publish History
CREATE TABLE IF NOT EXISTS publish_history (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    version_id TEXT,
    published_at INTEGER NOT NULL,
    result TEXT,
    FOREIGN KEY (node_id) REFERENCES nodes(id),
    FOREIGN KEY (channel_id) REFERENCES publishing_channels(id)
);

-- Exports
CREATE TABLE IF NOT EXISTS exports (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    export_type TEXT NOT NULL,
    file_path TEXT,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

-- Plugins Registry
CREATE TABLE IF NOT EXISTS plugins_registry (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    manifest TEXT,
    enabled INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Credentials
CREATE TABLE IF NOT EXISTS credentials (
    id TEXT PRIMARY KEY,
    key TEXT UNIQUE NOT NULL,
    value TEXT NOT NULL,
    encrypted INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Configs
CREATE TABLE IF NOT EXISTS configs (
    id TEXT PRIMARY KEY,
    key TEXT UNIQUE NOT NULL,
    value TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Git Commits
CREATE TABLE IF NOT EXISTS git_commits (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    commit_hash TEXT NOT NULL,
    message TEXT,
    author TEXT,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

-- IPFS Content
CREATE TABLE IF NOT EXISTS ipfs_content (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    ipfs_hash TEXT NOT NULL,
    pinned INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

-- IPFS Publications
CREATE TABLE IF NOT EXISTS ipfs_publications (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    ipfs_hash TEXT NOT NULL,
    gateway_url TEXT,
    published_at INTEGER NOT NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

-- DNS Records
CREATE TABLE IF NOT EXISTS dns_records (
    id TEXT PRIMARY KEY,
    domain TEXT NOT NULL,
    record_type TEXT NOT NULL,
    name TEXT NOT NULL,
    value TEXT NOT NULL,
    ttl INTEGER DEFAULT 3600,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Game Embeds
CREATE TABLE IF NOT EXISTS game_embeds (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    game_id TEXT NOT NULL,
    embed_code TEXT,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

-- Game Scores
CREATE TABLE IF NOT EXISTS game_scores (
    id TEXT PRIMARY KEY,
    game_id TEXT NOT NULL,
    player_name TEXT NOT NULL,
    score INTEGER NOT NULL,
    metadata TEXT,
    created_at INTEGER NOT NULL
);

-- Portfolio Games
CREATE TABLE IF NOT EXISTS portfolio_games (
    id TEXT PRIMARY KEY,
    game_id TEXT NOT NULL,
    node_id TEXT NOT NULL,
    featured INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

-- Todos
CREATE TABLE IF NOT EXISTS todos (
    id TEXT PRIMARY KEY,
    node_id TEXT,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'pending',
    priority TEXT DEFAULT 'medium',
    due_date INTEGER,
    assigned_to TEXT,
    completed_at INTEGER,
    created_at INTEGER NOT NULL,
    modified_at INTEGER NOT NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

-- Reminders
CREATE TABLE IF NOT EXISTS reminders (
    id TEXT PRIMARY KEY,
    node_id TEXT,
    title TEXT NOT NULL,
    description TEXT,
    remind_at INTEGER NOT NULL,
    status TEXT DEFAULT 'pending',
    recurrence TEXT,
    notification_sent INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    modified_at INTEGER NOT NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_nodes_path ON nodes(path);
CREATE INDEX IF NOT EXISTS idx_nodes_type ON nodes(type);
CREATE INDEX IF NOT EXISTS idx_nodes_parent ON nodes(parent_id);
CREATE INDEX IF NOT EXISTS idx_nodes_site_id ON nodes(site_id);
CREATE INDEX IF NOT EXISTS idx_nodes_slug ON nodes(slug);
CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);

CREATE INDEX IF NOT EXISTS idx_versions_node_id ON versions(node_id);
CREATE INDEX IF NOT EXISTS idx_versions_status ON versions(status);

CREATE INDEX IF NOT EXISTS idx_permissions_node_id ON user_permissions(node_id);
CREATE INDEX IF NOT EXISTS idx_permissions_user_id ON user_permissions(user_id);

CREATE INDEX IF NOT EXISTS idx_node_visibility_token ON node_visibility(shared_token);

CREATE INDEX IF NOT EXISTS idx_media_node_id ON media(node_id);
CREATE INDEX IF NOT EXISTS idx_media_hash ON media(hash);

CREATE INDEX IF NOT EXISTS idx_media_library_user_id ON media_library(user_id);

CREATE INDEX IF NOT EXISTS idx_node_references_source ON node_references(source_node_id);
CREATE INDEX IF NOT EXISTS idx_node_references_target ON node_references(target_node_id);

CREATE INDEX IF NOT EXISTS idx_node_uris_node ON node_uris(node_id);

CREATE INDEX IF NOT EXISTS idx_node_tags_node ON node_tags(node_id);
CREATE INDEX IF NOT EXISTS idx_node_tags_tag ON node_tags(tag_id);

CREATE INDEX IF NOT EXISTS idx_blog_posts_node_id ON blog_posts(node_id);
CREATE INDEX IF NOT EXISTS idx_blog_posts_slug ON blog_posts(slug);

CREATE INDEX IF NOT EXISTS idx_citations_node_id ON citations(node_id);

CREATE INDEX IF NOT EXISTS idx_publish_jobs_node_id ON publish_jobs(node_id);
CREATE INDEX IF NOT EXISTS idx_publish_jobs_status ON publish_jobs(status);

CREATE INDEX IF NOT EXISTS idx_publish_history_node_id ON publish_history(node_id);

CREATE INDEX IF NOT EXISTS idx_exports_node_id ON exports(node_id);

CREATE INDEX IF NOT EXISTS idx_plugins_slug ON plugins_registry(slug);

CREATE INDEX IF NOT EXISTS idx_git_commits_node_id ON git_commits(node_id);

CREATE INDEX IF NOT EXISTS idx_ipfs_content_node_id ON ipfs_content(node_id);
CREATE INDEX IF NOT EXISTS idx_ipfs_content_hash ON ipfs_content(ipfs_hash);

CREATE INDEX IF NOT EXISTS idx_ipfs_publications_node_id ON ipfs_publications(node_id);

CREATE INDEX IF NOT EXISTS idx_dns_records_domain ON dns_records(domain);

CREATE INDEX IF NOT EXISTS idx_game_embeds_node_id ON game_embeds(node_id);

CREATE INDEX IF NOT EXISTS idx_todos_node_id ON todos(node_id);
CREATE INDEX IF NOT EXISTS idx_todos_status ON todos(status);

CREATE INDEX IF NOT EXISTS idx_reminders_node_id ON reminders(node_id);
CREATE INDEX IF NOT EXISTS idx_reminders_remind_at ON reminders(remind_at);
CREATE INDEX IF NOT EXISTS idx_reminders_status ON reminders(status);
