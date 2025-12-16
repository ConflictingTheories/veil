# Veil Enhancement Plan - TODO

## Phase 1: Core Content Types & URI System ✅ COMPLETED
- [x] Extend Node types in main.go (page, post, canvas, shader-demo, code-snippet, image, video, audio, document, todo, reminder)
- [x] Implement universal URI scheme: veil://site/path/entity
- [x] Add URI resolution and linking throughout the system
- [x] Update database migrations for new content types
- [x] Fix database migration UNIQUE constraint issue
- [x] Create separate migration file to fix column addition issues

## Phase 2: Creative Tools Integration ✅ COMPLETED
- [x] Create SVG Drawing Plugin (svg_plugin.go)
  - [x] Canvas-based SVG editor
  - [x] Hotkey activation (Ctrl+Shift+D)
  - [x] Context awareness and save/export/share
- [x] Create Code Snippet Editor Plugin (code_plugin.go)
  - [x] Syntax-highlighted editor for multiple languages
  - [x] Code execution capabilities
- [ ] Create Shader Demo Editor Plugin (shader_plugin.go)
  - [ ] WebGL shader editor with live preview
- [ ] Create Image Tools Plugin (image_plugin.go)
  - [ ] Basic editing, annotations, galleries

## Phase 3: Productivity Features
- [ ] Create Todo System Plugin (todo_plugin.go)
  - [ ] Task management with due dates, priorities
  - [ ] Assignment capabilities
- [ ] Create Reminder System Plugin (reminder_plugin.go)
  - [ ] Time-based notifications
  - [ ] Recurring reminders
- [ ] Create Video Chat Integration Plugin (video_plugin.go)
  - [ ] Integration with external video services

## Phase 4: Advanced Publishing & Deployment
- [ ] Add publishing channels: Static site generation
- [ ] Add GitHub Pages deployment
- [ ] Add Netlify/Vercel deployment options
- [ ] Website building with themes/templates
- [ ] One-click hosting for interactive content

## Phase 5: UI/UX Enhancements
- [ ] Customizable UI themes and layouts
- [ ] Headless mode for API-only usage
- [ ] Hotkey system for quick actions
- [ ] Context-aware tool suggestions
- [ ] Enhanced web/app.js with new editors and tools

## Phase 6: Knowledge Graph Expansion
- [ ] Enhanced references and backlinks
- [ ] Content relationships and dependencies
- [ ] Advanced search and discovery

## Phase 7: Plugin Ecosystem
- [ ] Plugin marketplace/registry
- [ ] API for third-party plugin development
- [ ] Plugin configuration and management UI

## Testing & Iteration
- [ ] Test all new features
- [ ] Performance optimization
- [ ] Bug fixes and refinements
