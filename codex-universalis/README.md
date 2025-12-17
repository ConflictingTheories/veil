# Codex Universal PWA

A Progressive Web Application for managing distributed knowledge repositories with ontological precision.

## Overview

Codex Universal is a next-generation text referencing standard that bridges academia, programming, and the humanities. This PWA provides an intuitive interface for creating, annotating, and preserving cultural heritage texts with Git-like versioning and federated synchronization.

## Features

### 📚 Core Functionality
- **Text Management**: Create and manage text fragments with URN-based referencing
- **Ontological Entities**: Define characters, places, concepts with rich metadata
- **Annotations**: Link text fragments to entities with certainty scores
- **Version Control**: Git-inspired commit system with full history tracking
- **Offline-First**: Works completely offline with IndexedDB storage

### 🌐 Distributed Architecture
- **Federated Sync**: Connect to trusted authority nodes
- **Mirror/Proxy Modes**: Choose full replication or on-demand access
- **Pull Requests**: Contribute changes to canonical datasets
- **Alternative Versions**: Track and compare divergent interpretations

### 🎨 Design Philosophy
- **Ancient Manuscript Aesthetic**: Parchment tones, illuminated borders, serif typography
- **Modern UX**: Smooth animations, intuitive navigation, responsive design
- **Accessibility**: Keyboard navigation, semantic HTML, ARIA labels

## Installation

### As a PWA (Recommended)
1. Open the app in a modern browser (Chrome, Edge, Safari)
2. Look for the "Install" prompt or "Add to Home Screen"
3. The app will work offline after installation

### Local Development
```bash
# Clone the repository
git clone https://github.com/your-org/codex-universal-pwa.git
cd codex-universal-pwa

# Serve locally (requires Python 3)
python3 -m http.server 8000

# Open in browser
open http://localhost:8000
```

## Usage

### Creating Your First Text
1. Navigate to the **Texts** tab
2. Click **Add Text**
3. Enter:
   - URN (e.g., `urn:codex:iliad@homer:book1:lines1-10`)
   - Original content
   - Language code (ISO 639-3)
   - Translation (optional)
4. Click **Add Text**

### Defining Entities
1. Go to the **Entities** tab
2. Click **Create Entity**
3. Fill in:
   - Entity ID (e.g., `achilles`)
   - Type (Character, Place, Concept, Event)
   - Labels in multiple languages
4. Click **Create Entity**

### Linking Text to Entities
1. Navigate to **Annotations**
2. Click **Add Annotation**
3. Specify:
   - Text URN
   - Entity URN
   - Character positions (start/end)
   - Role (subject, mentioned, etc.)
4. Click **Add Annotation**

### Committing Changes
1. Go to **Repository** tab
2. Click **Commit Changes**
3. Enter commit message and author info
4. Changes are saved with cryptographic hashing

### Synchronizing with Trusted Authorities
1. Navigate to **Synchronization**
2. Click **Add Remote**
3. Enter trusted authority details
4. Choose sync mode (Mirror or Proxy)
5. Pull/push changes as needed

## Architecture

### Data Model
- **Texts**: Versioned fragments with URNs and metadata
- **Entities**: Ontological objects with properties and relationships
- **Annotations**: Links between texts and entities with certainty scores
- **Commits**: Versioned snapshots with hashes and attribution

### Storage
- **IndexedDB**: Local persistence for offline functionality
- **Service Worker**: Caching and background sync
- **Content Addressing**: SHA-256 hashing for data integrity

### APIs
```javascript
// Example: Adding a text programmatically
await app.addToStore('texts', {
  urn: 'urn:codex:odyssey@homer:book1',
  content: 'Ἄνδρα μοι ἔννεπε...',
  language: 'grc',
  metadata: { work: 'Odyssey', author: 'Homer' }
});
```

## Technology Stack

- **Frontend**: Vanilla JavaScript (no framework overhead)
- **Storage**: IndexedDB for structured data
- **Styling**: Custom CSS with CSS Variables
- **PWA**: Service Worker, Web App Manifest
- **Fonts**: Crimson Pro, Cormorant Garamond, JetBrains Mono

## Browser Support

- ✅ Chrome/Edge 90+
- ✅ Safari 14+
- ✅ Firefox 88+
- ⚠️ Internet Explorer: Not supported

## Keyboard Shortcuts

- `Ctrl/Cmd + N`: New item (context-aware)
- `Ctrl/Cmd + S`: Commit changes
- `Ctrl/Cmd + K`: Search/query
- `Esc`: Close modals
- `Tab`: Navigate between tabs

## Contributing

### Development Workflow
1. Fork the repository
2. Create a feature branch
3. Make changes and test locally
4. Submit a pull request with:
   - Clear description
   - Sample data (if applicable)
   - Screenshots for UI changes

### Code Standards
- ES6+ JavaScript
- Semantic HTML5
- BEM-inspired CSS naming
- Comprehensive comments

## Roadmap

### Phase 1: MVP (Current)
- [x] Basic CRUD for texts, entities, annotations
- [x] Offline storage with IndexedDB
- [x] PWA installation
- [x] Sample data (Homer, Plato)

### Phase 2: Collaboration
- [ ] Real-time sync with server
- [ ] Conflict resolution UI
- [ ] Pull request workflow
- [ ] User authentication (ORCID)

### Phase 3: Advanced Features
- [ ] Semantic search with SPARQL-like queries
- [ ] AI-assisted entity linking
- [ ] 3D manuscript viewer integration
- [ ] Export to TEI XML, JSON-LD

### Phase 4: Federation
- [ ] P2P node discovery
- [ ] Distributed hash table (DHT)
- [ ] Cryptographic signing with GPG
- [ ] Reputation system for contributors

## License

MIT License - see [LICENSE](LICENSE) for details

## Credits

**Design Inspiration**:
- Medieval illuminated manuscripts
- Byzantine scriptoriums
- Modern archival interfaces

**Sample Data**:
- Homer's Iliad and Odyssey (public domain)
- Plato's Republic (public domain)

**Typography**:
- Crimson Pro by Jacques Le Bailly
- Cormorant Garamond by Christian Thalmann
- JetBrains Mono by JetBrains

## Support

- **Documentation**: [docs.codexuniversal.org](https://docs.codexuniversal.org)
- **Issues**: [GitHub Issues](https://github.com/your-org/codex-universal-pwa/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-org/codex-universal-pwa/discussions)
- **Email**: support@codexuniversal.org

## Citation

If you use Codex Universal in academic work, please cite:

```bibtex
@software{codex_universal_2024,
  title = {Codex Universal: A Distributed Knowledge Architecture},
  author = {Codex Foundation},
  year = {2024},
  url = {https://codexuniversal.org}
}
```

---

**Built with scholarship, for scholars. 📜**
