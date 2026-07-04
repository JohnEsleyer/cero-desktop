# Cero Desktop — Wails + Svelte

Desktop companion for [Cero Journal](https://github.com/JohnEsleyer/pocketdatabase). Discovers the mobile app on the local network, connects via WebSocket with PIN authentication, and provides a full markdown editing workspace.

## Architecture

```
┌──────────────────────────────────────────┐
│  Cero Desktop (Wails / Go / Svelte)      │
│                                          │
│  ┌─────────────┐    ┌──────────────────┐  │
│  │ Svelte UI   │◄──►│ Go Backend       │  │
│  │ (Vite)      │    │  • WS Client     │──│──── WebSocket ───► Cero Mobile
│  │  • Sidebar  │    │  • UDP Listener  │──│──── Multicast :9100
│  │  • Editor   │    │  • Page Cache    │  │
│  │  • Preview  │    └──────────────────┘  │
│  └─────────────┘                          │
└──────────────────────────────────────────┘
```

- **Go backend** (`app.go`) — WebSocket client that connects to the Flutter server, listens for UDP multicast beacons, and maintains an in-memory page cache
- **Svelte frontend** — recursive page tree sidebar, split-pane markdown editor with live preview, emoji picker
- **Lazy loading** — initial sync transfers metadata only (no content); page content is fetched on demand
- **Auth** — 4-digit PIN required for WebSocket connection; pairing approval dialog on mobile

## Features

- Automatic device discovery via UDP multicast
- Manual IP + PIN connection fallback
- Infinite nesting page tree (recursive `<svelte:self>`)
- Edit / Preview / Split Screen modes
- Markdown toolbar (bold, italic, headers, lists, code)
- Drag-and-drop ready reparenting (Move To...)
- Soft-delete archive with restore option

## Getting Started

Requires [Wails CLI v2](https://wails.io/docs/gettingstarted/installation) and [Go 1.23+](https://go.dev/dl/).

```bash
# Install frontend dependencies
cd frontend && npm install && cd ..

# Run in live development mode
wails dev

# Build production binary
wails build
```

The built binary will be placed in `build/bin/` as `cero-desktop`.
