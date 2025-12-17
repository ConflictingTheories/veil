# Veil - Your Personal OS

**A self-contained, universal content management system for creative minds.**

Veil is your digital brain, portfolio manager, blog publisher, and creative toolkit—all in one. It combines the power of a knowledge graph with the flexibility of a static site generator, running entirely from a single binary with SQLite storage.

## 🌟 Philosophy

Veil works in two modes:
1. **Atomic Server** - Full-featured dynamic CMS with database backend
2. **Static Export** - Generate self-contained static websites that work anywhere

Everything in Veil has a **URI** (`veil://site/type/slug`), making your content universally addressable and linkable. Think of it as your personal web—a knowledge graph that can be accessed locally, served dynamically, or exported as static files.

## ✨ Features

### Core Capabilities

- **Codex Knowledge Graph** - Git-like version control for all content with branching, merging, and history
- **Multi-Site Management** - Create unlimited sites (portfolios, blogs, projects)
- **Rich Content Types** - Notes, pages, posts, canvases, shader demos, code snippets, media
- **Version Control** - Built-in versioning with publish/rollback capabilities
- **Universal URI System** - Every entity is addressable via `veil://` protocol
- **Static Site Export** - Generate complete, self-contained websites as ZIP files
- **Plugin Architecture** - Extensible system for Git, IPFS, media processing, and more

### Content Management

- **Markdown Editor** - Clean, distraction-free writing experience with live preview
- **Auto-save** - Never lose your work
- **Tags & Organization** - Categorize and find content easily
- **Search** - Fast full-text search across all content
- **Backlinks & Forward Links** - See how your notes connect
- **Media Library** - Store and manage images, videos, audio

### Publishing & Export

- **Static Site Generation** - One-click export to deployable website
- **RSS Feeds** - Automatic RSS feed generation
- **Multiple Channels** - Publish to Git, IPFS, FTP, SCP
- **Responsive Design** - Mobile-first, beautiful default theme
- **PWA Support** - Progressive Web App manifest included

### Advanced Features

- **SVG Canvas** - Create and edit SVG graphics inline
- **Shader Demos** - WebGL shader playground integration
- **Code Snippets** - Syntax-highlighted code blocks
- **Custom URIs** - Create friendly aliases for any content
- **Permissions** - Control visibility (public/private/draft)
- **Credentials Manager** - Secure storage for API keys and tokens

## 🚀 Quick Start

### Installation

```bash
# Clone or download the binary
go build .

# Initialize a new vault
./veil init

# Start the GUI
./veil gui
```

The GUI will open at `http://localhost:8080`

### Basic Usage

1. **Create a Site**
   - Click "New Site" in the sidebar
   - Give it a name (e.g., "My Portfolio")
   - Start adding content

2. **Create Notes**
   - Click "New Note"
   - Write in Markdown
   - Auto-save handles the rest

3. **Export Your Site**
   - Click the export button (download icon)
   - Download a complete static website
   - Upload to any hosting service

## 📚 Content Types

### Notes
Quick thoughts, drafts, or personal notes. Perfect for your second brain.

### Pages  
Permanent content like "About" or "Contact" pages.

### Posts
Blog posts with publish dates and categories.

### Canvas
SVG drawings and graphics created with the built-in editor.

### Shader Demos
Interactive WebGL shader demonstrations.

### Code Snippets
Syntax-highlighted code examples with multiple language support.

### Media
Images, videos, audio files with automatic optimization.

## 🔌 Plugin System

Veil includes a robust plugin architecture:

### Built-in Plugins

- **Git** - Version control integration
- **IPFS** - Decentralized content publishing
- **Namecheap** - Domain management
- **Media** - Image/video processing
- **Pixospritz** - Advanced graphics operations

### Creating Plugins

Plugins implement the `Plugin` interface:

```go
type Plugin interface {
    Name() string
    Version() string
    Initialize(config map[string]interface{}) error
    Execute(ctx context.Context, action string, payload interface{}) (interface{}, error)
    Validate() error
    Shutdown() error
}
```

## 🌐 URI System

Every entity in Veil has a canonical URI:

```
veil://site_id/type/slug
```

Examples:
- `veil://portfolio/page/about`
- `veil://blog/post/my-first-post`
- `veil://projects/canvas/logo-design`

You can also create custom URI aliases for any content.

## 🧠 Codex Knowledge Graph

Veil's core is powered by **Codex**, a Git-like knowledge graph that provides version control for all content:

### Features

- **Content-Addressed Storage** - Every piece of content is stored by its hash
- **Git-like Versioning** - Commits, branches, merges, and history tracking
- **Universal Data Types** - Supports text, binary, JSON, media, and custom formats
- **Efficient Storage** - Streaming support for large files and media
- **Conflict Resolution** - Three-way merges with conflict detection
- **Export Formats** - Export commits as ZIP or JSON-LD

### API Endpoints

- `GET /api/codex/status` - Repository status
- `POST /api/codex/commit` - Create new commit
- `GET /api/codex/commits` - List commits
- `GET /api/codex/diff` - Compare commits
- `POST /api/codex/merge` - Merge branches
- `GET /api/codex/export` - Export commit data

All content in Veil is automatically versioned through Codex, providing a complete audit trail and enabling advanced features like branching for drafts and collaborative editing.

## 📤 Export & Publishing

### Static Site Export

Generates a complete website package including:
- ✓ HTML pages for all published content
- ✓ Responsive CSS
- ✓ RSS feed (`feed.xml`)
- ✓ JSON API (`api.json`)
- ✓ PWA manifest (`manifest.json`)

### Publishing Channels

- **Static** - Export as ZIP
- **Git** - Commit and push to repository
- **IPFS** - Publish to InterPlanetary File System
- **RSS** - Generate/update RSS feed
- **FTP/SCP** - Direct server upload (coming soon)

## 🛠️ CLI Commands

```bash
# Initialize new vault
veil init [path]

# Start web server
veil serve [--port N]

# Launch GUI mode
veil gui

# Create new node
veil new <path>

# List all nodes
veil list

# Publish a node
veil publish <node-id>

# Export content
veil export <node-id> <type>

# Show version
veil version
```

## 🗄️ Database Schema

Veil uses SQLite with the following main tables:

- `nodes` - All content (notes, pages, posts, etc.)
- `sites` - Site/project definitions
- `versions` - Version history for nodes
- `node_uris` - Custom URI aliases
- `tags` - Content tags
- `media` - Media file metadata
- `plugins_registry` - Plugin configurations
- `publish_jobs` - Publishing queue

## 🎨 Customization

### Themes

Export comes with a beautiful default theme. Custom themes coming soon.

### Plugins

Extend Veil with custom plugins for:
- Custom publishing channels
- Content transformations
- External integrations
- Custom editors

## 🔐 Security & Privacy

- **Local-first** - All data stored in local SQLite database
- **No tracking** - No analytics, no telemetry
- **Encrypted credentials** - API keys stored securely
- **Permission system** - Control content visibility
- **Self-hosted** - Run anywhere, own your data

## 📖 Use Cases

### Personal Knowledge Base
Build your second brain with interconnected notes and backlinks.

### Portfolio Website
Showcase your work with a beautiful, exportable portfolio.

### Blog
Write and publish blog posts with RSS feeds.

### Project Documentation
Document your projects with version control.

### Creative Toolkit
Use SVG canvas, shader demos, and code snippets.

### Digital Garden
Grow a public digital garden of your thoughts.

## 🗺️ Roadmap

- [ ] Real-time collaboration
- [ ] End-to-end encryption
- [ ] Mobile apps (iOS/Android)
- [ ] Plugin marketplace
- [ ] Theme marketplace
- [ ] Git-like sync protocol
- [ ] Web hosting service
- [ ] Desktop app (Electron/Tauri)

## 🤝 Contributing

Veil is open source. Contributions welcome!

## 📄 License

MIT License - See LICENSE file

## 🙏 Acknowledgments

Built with:
- Go + SQLite for the backend
- Vanilla JS for the frontend
- Tailwind CSS for styling
- Markdown for content

---

**Veil** - Build your digital universe, export it anywhere.

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

## Integration & Roadmap 📚

For the plan to merge **Veil** and **Codex**, an implementation and migration guide has been added to the repository: `docs/INTEGRATION_PLAN.md` (contains the repo audit, design decisions, migration steps, developer-level actions, and suggested commands). Please read and follow this doc for the unified effort.
