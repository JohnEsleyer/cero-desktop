# Cero Desktop — Wails + Svelte

Desktop companion for [Cero Journal](https://github.com/JohnEsleyer/pocketdatabase). Discovers the mobile app on the local network, connects via WebSocket with PIN authentication, and provides a block-based card editing workspace.

> **Upgrading**: See [root README](../README.md) for the full technical plan transitioning from single-text markdown to multi-workspace card-based editing with subpages and side pages.

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
│  │  • Cards    │    │  • Card Cache    │  │
│  └─────────────┘    └──────────────────┘  │
└──────────────────────────────────────────┘
```

- **Go backend** (`app.go`) — WebSocket client that connects to the Flutter server, listens for UDP multicast beacons, and maintains an in-memory page + card cache
- **Svelte frontend** — recursive page tree sidebar, block-based card editor, side pages panel, workspace selector
- **Lazy loading** — initial sync transfers metadata only; page content and card data fetched on demand
- **Auth** — 4-digit PIN required for WebSocket connection; pairing approval dialog on mobile

## Features

- Multi-workspace switching (dynamic `.db` file selection)
- Automatic device discovery via UDP multicast
- Manual IP + PIN connection fallback
- Infinite nesting page tree (recursive `<svelte:self>`)
- Block-based card editor (markdown, image, subpage link cards)
- Hierarchical subpages (left sidebar) + contextual side pages (right sidebar)
- Drag-and-drop card reordering
- Markdown toolbar (bold, italic, headers, lists, code)
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
