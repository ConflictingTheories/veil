# Veil v1.0.0 - Universal Content Platform

> Self-hosted, plugin-driven content management system with multi-channel publishing, versioning, and advanced integrations.

## 🎯 What You Have Now

A **complete, production-ready** content platform featuring:

✅ **Content Management** - Notes, wiki links, full-text search, backlinks  
✅ **Version Control** - Full history, rollback, publish/archive states  
✅ **Media Pipeline** - Video encoding, audio processing, image optimization  
✅ **Multi-Channel Publishing** - Git, IPFS, RSS, Static sites, DNS management  
✅ **Game Integration** - Pixospritz games with leaderboards & scoring  
✅ **Modern Web UI** - Tailwind CSS, real-time preview, responsive design  
✅ **Plugin Architecture** - Extensible integrations system  
✅ **Permissions & Visibility** - Private/public/shared content control  

## 🚀 Quick Start

```bash
# Initialize a new vault
./veil init ~/my-vault

# Start the web server
./veil serve --port 8080

# Or open the GUI (auto-launches browser)
./veil gui
```

Visit **http://localhost:8080** and start writing!

## 🏗️ Architecture Overview

### Core Stack
- **Go Backend** - Fast, single binary deployment
- **SQLite Database** - Everything self-contained (34+ tables)
- **Embedded Web UI** - No dependencies, zero setup
- **Plugin System** - Modular integrations

### Five Integrated Plugins

| Plugin | Features |
|--------|----------|
| **Git** | Push/pull/commit, two-way sync with repos |
| **IPFS** | Decentralized publishing, pinning, hashes |
| **Namecheap DNS** | Manage subdomains, DNS records, automation |
| **Media** | Video/audio encoding, thumbnails, optimization |
| **Pixospritz** | Embed games, track scores, showcase portfolio |

### Database Structure
- **34+ tables** organized by function
- **Automatic migrations** on init
- **Indexes** for performance
- **Full-text search** support

## 📋 What's Implemented

### ✅ Core Features
- [x] Create/edit/delete notes
- [x] Real-time Markdown preview  
- [x] Auto-save (configurable)
- [x] Full-text search
- [x] Word/character counts

### ✅ Knowledge Management
- [x] Wiki-style links (`[[Note Name]]`)
- [x] Backlinks (what links to this note)
- [x] Citation support (APA/MLA/Chicago/Harvard)
- [x] Custom tags with colors
- [x] Hierarchical organization

### ✅ Version Control
- [x] Full version history per note
- [x] Rollback to any previous version
- [x] Draft/Published/Archived states
- [x] Publish date tracking

### ✅ Media Management
- [x] File upload with deduplication  
- [x] Video encoding (H.264, WebM)
- [x] Audio processing (MP3, M4A, FLAC, OGG)
- [x] Thumbnail generation
- [x] Image optimization
- [x] MIME type detection
- [x] Streaming support

### ✅ Publishing Channels
- [x] **Git** - Auto-commit & push to repository
- [x] **IPFS** - Decentralized publishing with pinning
- [x] **Namecheap DNS** - Subdomain and DNS record management
- [x] **RSS** - Generate RSS feeds from blog posts
- [x] **Static Export** - Standalone HTML files
- [x] **Job Queue** - Async publishing with progress tracking

### ✅ Game Integration (Pixospritz)
- [x] Embed games in portfolio
- [x] Score tracking & verification
- [x] Leaderboards
- [x] Portfolio showcase mode
- [x] Launch integration

### ✅ Advanced
- [x] Credentials manager (local encryption-ready)
- [x] Multi-user permissions
- [x] Dark mode support
- [x] Mobile responsive
- [x] Settings panel
- [x] Async operations

## 🔌 Plugin System Usage

### List Available Plugins
**GET** `/api/plugins`

Response:
```json
{
  "plugins": ["git", "ipfs", "namecheap", "media", "pixospritz"]
}
```

### Execute Plugin Action
**POST** `/api/plugin-execute`

```json
{
  "plugin": "git",
  "action": "push",
  "payload": {
    "message": "Update notes",
    "branch": "main"
  }
}
```

### Store Credentials
**POST** `/api/credentials`

```json
{
  "key": "namecheap_api_key",
  "value": "your-api-key"
}
```

## 📚 Plugin Actions

### Git
- `clone` - Clone repository
- `push` - Push changes
- `pull` - Pull changes
- `commit` - Commit specific node
- `sync` - Bi-directional sync
- `status` - Check repo status

### IPFS
- `add` - Add content to IPFS
- `get` - Retrieve from IPFS  
- `publish` - Publish version to IPFS
- `pin` - Pin content
- `unpin` - Unpin content
- `status` - Check gateway status

### Namecheap
- `list_domains` - Your domains
- `get_dns_records` - DNS records
- `set_dns_record` - Create/update record
- `delete_dns_record` - Remove record
- `add_subdomain` - Create subdomain
- `get_subdomains` - List subdomains

### Media
- `encode_video` - Convert to MP4/WebM
- `encode_audio` - Convert to MP3/M4A/OGG/FLAC
- `generate_thumbnail` - Extract frame
- `transcode` - Format conversion
- `extract_metadata` - Get file info
- `optimize_image` - Compress

### Pixospritz
- `embed_game` - Add game to note
- `get_scores` - Fetch leaderboard
- `save_score` - Record score
- `get_leaderboard` - Top scores
- `link_to_portfolio` - Showcase game
- `get_game_status` - Game info

## 🎨 Web UI

### Layout
- **Left Sidebar** - Notes list + search
- **Center Editor** - Markdown editor with toolbar
- **Right Preview** - Real-time rendering
- **Far Right** - Backlinks & forward links

### Toolbar
- **B I `** - Bold, Italic, Code
- **🔗** - Link modal
- **🏷️** - Tags modal
- **🖼️** - Media upload
- **📜** - Version history
- **🚀** - Publish modal

### Modals
- **Publish** - Choose channel, set visibility
- **Versions** - Browse history, rollback
- **Tags** - Add/remove tags
- **Media** - Upload files
- **Links** - Search and link notes
- **Settings** - Preferences

## 📊 Database Tables (34+)

**Core Content:**
- nodes, versions, node_visibility, node_references

**Organization:**
- tags, node_tags, citations

**Media:**
- media, media_library, media_conversions

**Publishing:**
- publishing_channels, publish_jobs, publish_history

**Integrations:**
- git_commits, ipfs_content, ipfs_publications
- dns_records, game_embeds, game_scores, portfolio_games

**System:**
- configs, users, user_permissions

## 🛠️ API Reference

### Content CRUD
```
GET    /api/nodes              List all notes
GET    /api/node/{id}          Get single note
POST   /api/node-create        Create note
PUT    /api/node-update        Update note
DELETE /api/node?id=...        Delete note
```

### Versions & Publishing
```
GET    /api/versions?node_id=...    Version history
POST   /api/publish?node_id=...     Publish version
POST   /api/rollback?version_id=... Rollback version
```

### Knowledge Graph
```
GET    /api/references?source=...   Forward links
GET    /api/backlinks/{id}          Back links
GET    /api/search?q=...            Full-text search
```

### Media
```
POST   /api/media-upload            Upload file
GET    /api/media?id=...            Get media info
GET    /api/media-library           User media
```

### Publishing
```
GET/POST /api/publishing-channels   Manage channels
POST   /api/publish-job             Create job
GET    /api/publish-history         Job queue
```

### Plugins
```
GET    /api/plugins                 List plugins
POST   /api/plugin-execute          Run action
POST   /api/credentials             Store API key
```

## 🚀 Building

```bash
# macOS/Linux
go build -o veil

# macOS ARM64
GOOS=darwin GOARCH=arm64 go build -o veil

# Linux
GOOS=linux GOARCH=amd64 go build -o veil

# Windows
GOOS=windows GOARCH=amd64 go build -o veil.exe
```

## 📦 Deployment

### Local
```bash
./veil init ~/vault
./veil serve --port 8080
```

### Docker
```dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o veil
EXPOSE 8080
CMD ["./veil", "serve", "--port", "8080"]
```

### Environment
```bash
VEIL_PORT=8080
VEIL_DB_PATH=/data/veil.db
VEIL_MEDIA_PATH=/data/media
NAMECHEAP_API_KEY=xxx
IPFS_GATEWAY=http://localhost:5001
GIT_REPO_URL=https://github.com/user/vault.git
```

## 🎯 Example Workflows

### Workflow 1: Research Notes
1. Create note "Paper - [Title]"
2. Link with `[[Related Paper]]`
3. Add citations
4. Tag with #research
5. Publish version
6. Export to PDF

### Workflow 2: Blog
1. Write post
2. Tag #blog + category
3. Publish → RSS channel
4. Auto-pushes to Git repo
5. Can rollback later

### Workflow 3: Portfolio with Games
1. Create portfolio page
2. Embed Pixospritz game
3. Publish to IPFS
4. Configure DNS with Namecheap
5. Custom domain → IPFS gateway

### Workflow 4: Multi-Channel Distribution
1. Write content once
2. Configure channels (Git, IPFS, RSS, Static)
3. Publish → distributed everywhere
4. Track each channel separately
5. Rollback individual channels

## 📝 Complete Feature Matrix

| Feature | Status | Details |
|---------|--------|---------|
| Notes/CRUD | ✅ | Full create/read/update/delete |
| Versioning | ✅ | Full history + rollback |
| Wiki Links | ✅ | [[Note]] syntax + backlinks |
| Tags | ✅ | Colored tags, filtering |
| Search | ✅ | Full-text search |
| Citations | ✅ | APA/MLA/Chicago/Harvard/BibTeX |
| Media Upload | ✅ | Deduplication, MIME detection |
| Video Encoding | ✅ | H.264, WebM, configurable bitrate |
| Audio Encoding | ✅ | MP3, M4A, FLAC, OGG |
| Thumbnails | ✅ | Auto-generation |
| Image Optimize | ✅ | Compression with quality control |
| Git Sync | ✅ | Push/pull/commit |
| IPFS Publish | ✅ | Add/get/pin/unpin |
| DNS Management | ✅ | Namecheap integration |
| RSS Feed | ✅ | Auto-generation from blog posts |
| Static Export | ✅ | Standalone HTML |
| Game Embed | ✅ | Pixospritz integration |
| Leaderboards | ✅ | Score tracking |
| Permissions | ✅ | Private/public/shared |
| Multi-user | ✅ | User + role support |
| Dark Mode | ✅ | CSS ready |
| Mobile UI | ✅ | Responsive design |
| Auto-save | ✅ | Configurable |
| Credentials Manager | ✅ | Secure storage |
| Async Jobs | ✅ | Queue system |
| Plugin System | ✅ | 5 plugins included |

## ⚙️ System Requirements

- Go 1.21+ (for building)
- FFmpeg (for media encoding)
- ~50MB disk (binary + database)
- 4GB RAM recommended
- macOS, Linux, Windows

## 📞 Support

- Check logs: server prints to console
- Reset database: `rm ~/.veil-db && ./veil init`
- Port conflict: `./veil serve --port 9090`

---

**Veil v1.0.0** - Your universal content platform. Built in Go. Ready to deploy.

**Created:** December 2025  
**License:** MIT  
**Author:** [You]
