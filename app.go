package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/net/ipv4"
)

// DbPage represents the database page model matching Dart's DbPage
type DbPage struct {
	ID           string  `json:"id"`
	ParentID     *string `json:"parent_id"`
	RelationType string  `json:"relation_type"` // "subpage" or "sidepage"
	Title        string  `json:"title"`
	Emoji        string  `json:"emoji"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	IsArchived   int     `json:"is_archived"`
	SortOrder    int     `json:"sort_order"`
	Revision     int     `json:"revision"`
}

// Card represents a content block within a page
type Card struct {
	ID        string `json:"id"`
	PageID    string `json:"page_id"`
	Type      string `json:"type"` // "markdown", "image", "subpage_link", "file"
	Content   string `json:"content"`
	SortOrder int    `json:"sort_order"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Revision  int    `json:"revision"`
}

// DiscoveredDevice represents a device broadcasting UDP beacons
type DiscoveredDevice struct {
	IP         string    `json:"ip"`
	Port       int       `json:"port"`
	DeviceName string    `json:"deviceName"`
	LastSeen   time.Time `json:"-"`
}

// App struct manages application state
type App struct {
	ctx        context.Context
	wsConn     *websocket.Conn
	connStatus string
	connMutex  sync.Mutex

	// Database state cache
	dbPages      []DbPage
	dbPagesMutex sync.RWMutex

	// Card cache: pageID -> []Card
	cardCache      map[string][]Card
	cardCacheMutex sync.RWMutex

	// Workspace state
	activeWorkspace string
	workspaceCache  []string // list of workspace names
	imageDir        string   // directory for storing workspace images

	// UDP Discovery state
	udpListener   *net.UDPConn
	isDiscovering bool
	discoveredMap map[string]DiscoveredDevice
	discoveredMux sync.Mutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		connStatus:    "disconnected",
		dbPages:       []DbPage{},
		cardCache:     make(map[string][]Card),
		discoveredMap: make(map[string]DiscoveredDevice),
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.StartUdpDiscovery()
}

// shutdown is called when the app closes
func (a *App) shutdown(ctx context.Context) {
	a.StopUdpDiscovery()
	a.Disconnect()
}

// GetConnectionStatus returns the current websocket connection status
func (a *App) GetConnectionStatus() string {
	a.connMutex.Lock()
	defer a.connMutex.Unlock()
	return a.connStatus
}

func (a *App) setConnectionStatus(status string) {
	a.connMutex.Lock()
	a.connStatus = status
	a.connMutex.Unlock()
	runtime.EventsEmit(a.ctx, "connection-status", status)
}

// GetDbPages returns the currently cached database pages
func (a *App) GetDbPages() []DbPage {
	a.dbPagesMutex.RLock()
	defer a.dbPagesMutex.RUnlock()
	return a.dbPages
}

// GetActiveWorkspace returns the current workspace name
func (a *App) GetActiveWorkspace() string {
	return a.activeWorkspace
}

// ListWorkspaces requests the workspace list from the server and returns cached names
func (a *App) ListWorkspaces() []string {
	a.sendToServer(map[string]interface{}{
		"type": "list_workspaces",
	})
	a.connMutex.Lock()
	conn := a.wsConn
	a.connMutex.Unlock()
	if conn == nil {
		return a.workspaceCache
	}
	return a.workspaceCache
}

// CreateWorkspace requests the Flutter server to create a new workspace
func (a *App) CreateWorkspace(name string) error {
	return a.sendToServer(map[string]interface{}{
		"type":          "create_workspace",
		"workspaceName": name,
	})
}

// SwitchWorkspace requests the Flutter server to switch to a workspace
func (a *App) SwitchWorkspace(name string) error {
	return a.sendToServer(map[string]interface{}{
		"type":          "switch_workspace",
		"workspaceName": name,
	})
}

// GetCards returns cached cards for a page
func (a *App) GetCards(pageID string) []Card {
	a.cardCacheMutex.RLock()
	defer a.cardCacheMutex.RUnlock()
	return a.cardCache[pageID]
}

// --- UDP Discovery Service ---

func (a *App) StartUdpDiscovery() {
	a.discoveredMux.Lock()
	if a.isDiscovering {
		a.discoveredMux.Unlock()
		return
	}
	a.isDiscovering = true
	a.discoveredMux.Unlock()

	go func() {
		addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:9100")
		if err != nil {
			fmt.Printf("UDP resolve address error: %v\n", err)
			return
		}

		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			fmt.Printf("UDP listen error: %v\n", err)
			return
		}

		multicastAddr := &net.UDPAddr{IP: net.IPv4(239, 255, 255, 250), Port: 9100}
		packetConn := ipv4.NewPacketConn(conn)
		if err := packetConn.JoinGroup(nil, multicastAddr); err != nil {
			fmt.Printf("Multicast join (non-fatal): %v\n", err)
		} else {
			fmt.Printf("Joined multicast group %s\n", multicastAddr.IP.String())
		}

		a.discoveredMux.Lock()
		a.udpListener = conn
		a.discoveredMux.Unlock()

		go a.pruneDevicesLoop()

		buf := make([]byte, 1024)
		for {
			a.discoveredMux.Lock()
			discovering := a.isDiscovering
			a.discoveredMux.Unlock()
			if !discovering {
				break
			}

			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				break
			}

			var payload struct {
				App        string `json:"app"`
				Port       int    `json:"port"`
				IP         string `json:"ip"`
				DeviceName string `json:"deviceName"`
			}

			if err := json.Unmarshal(buf[:n], &payload); err == nil && payload.App == "cero-journal" {
				a.discoveredMux.Lock()
				key := fmt.Sprintf("%s:%d", payload.IP, payload.Port)

				device := DiscoveredDevice{
					IP:         payload.IP,
					Port:       payload.Port,
					DeviceName: payload.DeviceName,
					LastSeen:   time.Now(),
				}

				_, exists := a.discoveredMap[key]
				a.discoveredMap[key] = device
				a.discoveredMux.Unlock()

				if !exists {
					a.emitDiscoveredDevices()
				}
			}
		}
	}()
}

func (a *App) StopUdpDiscovery() {
	a.discoveredMux.Lock()
	defer a.discoveredMux.Unlock()
	a.isDiscovering = false
	if a.udpListener != nil {
		a.udpListener.Close()
		a.udpListener = nil
	}
}

func (a *App) pruneDevicesLoop() {
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	for {
		a.discoveredMux.Lock()
		discovering := a.isDiscovering
		a.discoveredMux.Unlock()
		if !discovering {
			break
		}

		<-ticker.C

		a.discoveredMux.Lock()
		changed := false
		now := time.Now()
		for key, device := range a.discoveredMap {
			if now.Sub(device.LastSeen) > 8*time.Second {
				delete(a.discoveredMap, key)
				changed = true
			}
		}
		a.discoveredMux.Unlock()

		if changed {
			a.emitDiscoveredDevices()
		}
	}
}

func (a *App) emitDiscoveredDevices() {
	a.discoveredMux.Lock()
	devices := make([]DiscoveredDevice, 0, len(a.discoveredMap))
	for _, device := range a.discoveredMap {
		devices = append(devices, device)
	}
	a.discoveredMux.Unlock()

	runtime.EventsEmit(a.ctx, "discovered-devices", devices)
}

func (a *App) GetDiscoveredDevices() []DiscoveredDevice {
	a.discoveredMux.Lock()
	defer a.discoveredMux.Unlock()

	devices := make([]DiscoveredDevice, 0, len(a.discoveredMap))
	for _, device := range a.discoveredMap {
		devices = append(devices, device)
	}
	return devices
}

// --- WebSocket Client Service ---

func (a *App) ConnectToDevice(ip string, port int, pin string) error {
	a.Disconnect()

	a.setConnectionStatus("connecting")
	wsURL := fmt.Sprintf("ws://%s:%d/ws?pin=%s", ip, port, pin)

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		a.setConnectionStatus("disconnected")
		return fmt.Errorf("websocket dial error: %w", err)
	}

	a.connMutex.Lock()
	a.wsConn = conn
	a.connMutex.Unlock()
	a.setConnectionStatus("connecting")

	go a.readWebSocketLoop(conn)

	return nil
}

func (a *App) Disconnect() {
	a.connMutex.Lock()
	defer a.connMutex.Unlock()

	if a.wsConn != nil {
		a.wsConn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Client disconnecting"),
			time.Now().Add(1*time.Second),
		)
		a.wsConn.Close()
		a.wsConn = nil
	}

	if a.connStatus != "disconnected" {
		a.connStatus = "disconnected"
		go func() {
			runtime.EventsEmit(a.ctx, "connection-status", "disconnected")
		}()
	}
}

func (a *App) readWebSocketLoop(conn *websocket.Conn) {
	defer func() {
		a.connMutex.Lock()
		if a.wsConn == conn {
			a.wsConn = nil
			a.connMutex.Unlock()
			a.setConnectionStatus("disconnected")
		} else {
			a.connMutex.Unlock()
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var envelope struct {
			Type   string          `json:"type"`
			Data   json.RawMessage `json:"data"`
			Item   json.RawMessage `json:"item"`
			ID     string          `json:"id"`
			PageID string          `json:"page_id"`
		}

		if err := json.Unmarshal(message, &envelope); err != nil {
			fmt.Printf("Error unmarshalling WebSocket payload: %v\n", err)
			continue
		}

		switch envelope.Type {
		case "pairing_required":
			var pairInfo struct {
				RemoteAddress string `json:"remoteAddress"`
			}
			json.Unmarshal(envelope.Data, &pairInfo)
			runtime.EventsEmit(a.ctx, "pairing-required", map[string]string{
				"remoteAddress": pairInfo.RemoteAddress,
			})
		case "pairing_accepted":
			a.setConnectionStatus("connected")
			runtime.EventsEmit(a.ctx, "pairing-accepted", nil)
		case "pairing_rejected":
			a.setConnectionStatus("disconnected")
			runtime.EventsEmit(a.ctx, "pairing-rejected", nil)
			conn.Close()
			return

		// Workspace
		case "workspace_status":
			var wsInfo struct {
				ActiveWorkspace    string   `json:"activeWorkspace"`
				AvailableWorkspaces []string `json:"availableWorkspaces,omitempty"`
			}
			if err := json.Unmarshal(envelope.Data, &wsInfo); err == nil {
				a.activeWorkspace = wsInfo.ActiveWorkspace
				if len(wsInfo.AvailableWorkspaces) > 0 {
					a.workspaceCache = wsInfo.AvailableWorkspaces
				}
				runtime.EventsEmit(a.ctx, "workspace-status", wsInfo)
			}
		case "workspace_list":
			var wsList struct {
				Workspaces     []string `json:"workspaces"`
				ActiveWorkspace string  `json:"activeWorkspace"`
			}
			if err := json.Unmarshal(envelope.Data, &wsList); err == nil {
				a.workspaceCache = wsList.Workspaces
				a.activeWorkspace = wsList.ActiveWorkspace
				runtime.EventsEmit(a.ctx, "workspace-status", map[string]interface{}{
					"activeWorkspace":    wsList.ActiveWorkspace,
					"availableWorkspaces": wsList.Workspaces,
				})
			}

		// Page sync
		case "sync":
			var pages []DbPage
			if err := json.Unmarshal(envelope.Data, &pages); err == nil {
				a.dbPagesMutex.Lock()
				a.dbPages = pages
				a.dbPagesMutex.Unlock()
				runtime.EventsEmit(a.ctx, "db-update", a.dbPages)
			}
		case "add":
			var page DbPage
			if err := json.Unmarshal(envelope.Item, &page); err == nil {
				a.dbPagesMutex.Lock()
				exists := false
				for _, existing := range a.dbPages {
					if existing.ID == page.ID {
						exists = true
						break
					}
				}
				if !exists {
					a.dbPages = append(a.dbPages, page)
				}
				a.dbPagesMutex.Unlock()
				runtime.EventsEmit(a.ctx, "db-update", a.dbPages)
			}
		case "update":
			var page DbPage
			if err := json.Unmarshal(envelope.Item, &page); err == nil {
				a.dbPagesMutex.Lock()
				for i, existing := range a.dbPages {
					if existing.ID == page.ID {
						if page.Revision > 0 && page.Revision < existing.Revision {
							break
						}
						a.dbPages[i] = page
						break
					}
				}
				a.dbPagesMutex.Unlock()
				runtime.EventsEmit(a.ctx, "db-update", a.dbPages)
			}
		case "archive":
			a.dbPagesMutex.Lock()
			a.dbPages = removePageAndDescendants(a.dbPages, envelope.ID)
			a.dbPagesMutex.Unlock()
			runtime.EventsEmit(a.ctx, "db-update", a.dbPages)
		case "restore":
			runtime.EventsEmit(a.ctx, "db-refresh", nil)
		case "delete":
			a.dbPagesMutex.Lock()
			a.dbPages = removePageAndDescendants(a.dbPages, envelope.ID)
			a.dbPagesMutex.Unlock()
			runtime.EventsEmit(a.ctx, "db-update", a.dbPages)
		case "hard_delete":
			a.dbPagesMutex.Lock()
			a.dbPages = removePageAndDescendants(a.dbPages, envelope.ID)
			a.dbPagesMutex.Unlock()
			runtime.EventsEmit(a.ctx, "db-update", a.dbPages)
		case "move":
			var movePayload struct {
				ID       string  `json:"id"`
				ParentID *string `json:"parent_id"`
			}
			if err := json.Unmarshal(message, &movePayload); err == nil {
				a.dbPagesMutex.Lock()
				for i, p := range a.dbPages {
					if p.ID == movePayload.ID {
						a.dbPages[i].ParentID = movePayload.ParentID
						break
					}
				}
				a.dbPagesMutex.Unlock()
				runtime.EventsEmit(a.ctx, "db-update", a.dbPages)
			}

		// Card sync
		case "sync_cards":
			var payload struct {
				PageID string `json:"page_id"`
				Cards  []Card `json:"cards"`
			}
			if err := json.Unmarshal(envelope.Data, &payload); err == nil {
				a.cardCacheMutex.Lock()
				a.cardCache[payload.PageID] = payload.Cards
				a.cardCacheMutex.Unlock()
				runtime.EventsEmit(a.ctx, "cards-update", payload)
			}
		case "add_card":
			var card Card
			if err := json.Unmarshal(envelope.Item, &card); err == nil {
				a.cardCacheMutex.Lock()
				a.cardCache[card.PageID] = append(a.cardCache[card.PageID], card)
				a.cardCacheMutex.Unlock()
				runtime.EventsEmit(a.ctx, "cards-update", map[string]interface{}{
					"page_id": card.PageID,
					"cards":   a.cardCache[card.PageID],
				})
			}
		case "update_card":
			var card Card
			if err := json.Unmarshal(envelope.Item, &card); err == nil {
				a.cardCacheMutex.Lock()
				if cards, ok := a.cardCache[card.PageID]; ok {
					for i, c := range cards {
						if c.ID == card.ID {
							cards[i] = card
							break
						}
					}
				}
				a.cardCacheMutex.Unlock()
				runtime.EventsEmit(a.ctx, "cards-update", map[string]interface{}{
					"page_id": card.PageID,
					"cards":   a.cardCache[card.PageID],
				})
			}
		case "delete_card":
			var payload struct {
				ID     string `json:"id"`
				PageID string `json:"page_id"`
			}
			if err := json.Unmarshal(message, &payload); err == nil {
				a.cardCacheMutex.Lock()
				if cards, ok := a.cardCache[payload.PageID]; ok {
					for i, c := range cards {
						if c.ID == payload.ID {
							a.cardCache[payload.PageID] = append(cards[:i], cards[i+1:]...)
							break
						}
					}
				}
				remaining := a.cardCache[payload.PageID]
				a.cardCacheMutex.Unlock()
				runtime.EventsEmit(a.ctx, "cards-update", map[string]interface{}{
					"page_id": payload.PageID,
					"cards":   remaining,
				})
			}
		case "reorder_cards":
			var payload struct {
				PageID string   `json:"page_id"`
				Order  []string `json:"order"`
			}
			if err := json.Unmarshal(message, &payload); err == nil {
				a.cardCacheMutex.Lock()
				if cards, ok := a.cardCache[payload.PageID]; ok {
					orderMap := make(map[string]int)
					for i, id := range payload.Order {
						orderMap[id] = i
					}
					for i := range cards {
						if newOrder, ok := orderMap[cards[i].ID]; ok {
							cards[i].SortOrder = newOrder
						}
					}
					// Sort by new order
					for i := 0; i < len(cards); i++ {
						for j := i + 1; j < len(cards); j++ {
							if cards[i].SortOrder > cards[j].SortOrder {
								cards[i], cards[j] = cards[j], cards[i]
							}
						}
					}
				}
				remaining := a.cardCache[payload.PageID]
				a.cardCacheMutex.Unlock()
				runtime.EventsEmit(a.ctx, "cards-update", map[string]interface{}{
					"page_id": payload.PageID,
					"cards":   remaining,
				})
			}
		}
	}
}

func removePageAndDescendants(pages []DbPage, targetID string) []DbPage {
	var toDelete []string
	toDelete = append(toDelete, targetID)

	added := true
	for added {
		added = false
		for _, page := range pages {
			if page.ParentID != nil {
				parentInList := false
				for _, delID := range toDelete {
					if *page.ParentID == delID {
						parentInList = true
						break
					}
				}

				childInList := false
				for _, delID := range toDelete {
					if page.ID == delID {
						childInList = true
						break
					}
				}

				if parentInList && !childInList {
					toDelete = append(toDelete, page.ID)
					added = true
				}
			}
		}
	}

	var result []DbPage
	for _, page := range pages {
		inList := false
		for _, delID := range toDelete {
			if page.ID == delID {
				inList = true
				break
			}
		}
		if !inList {
			result = append(result, page)
		}
	}
	return result
}

// --- Write Operations (Sent to Flutter server) ---

func (a *App) sendToServer(message interface{}) error {
	a.connMutex.Lock()
	conn := a.wsConn
	a.connMutex.Unlock()

	if conn == nil {
		return fmt.Errorf("not connected to server")
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, payload)
}

// AddPage requests the Flutter server to add a new page
func (a *App) AddPage(parentID string, relationType string, title string, emoji string) error {
	var parentPtr *string
	if parentID != "" {
		parentPtr = &parentID
	}

	if relationType == "" {
		relationType = "subpage"
	}

	page := DbPage{
		ID:           uuid.New().String(),
		ParentID:     parentPtr,
		RelationType: relationType,
		Title:        title,
		Emoji:        emoji,
		CreatedAt:    time.Now().Format(time.RFC3339),
		UpdatedAt:    time.Now().Format(time.RFC3339),
		IsArchived:   0,
		SortOrder:    0,
	}

	message := map[string]interface{}{
		"type": "add",
		"item": page,
	}

	return a.sendToServer(message)
}

// UpdatePage requests the Flutter server to update a page
func (a *App) UpdatePage(id string, title string, emoji string) error {
	a.dbPagesMutex.RLock()
	currentRevision := 0
	for _, p := range a.dbPages {
		if p.ID == id {
			currentRevision = p.Revision
			break
		}
	}
	a.dbPagesMutex.RUnlock()

	page := DbPage{
		ID:        id,
		Title:     title,
		Emoji:     emoji,
		UpdatedAt: time.Now().Format(time.RFC3339),
		Revision:  currentRevision,
	}

	message := map[string]interface{}{
		"type": "update",
		"item": page,
	}

	return a.sendToServer(message)
}

func (a *App) DeletePage(id string) error {
	return a.sendToServer(map[string]interface{}{"type": "archive", "id": id})
}

func (a *App) RestorePage(id string) error {
	return a.sendToServer(map[string]interface{}{"type": "restore", "id": id})
}

func (a *App) HardDeletePage(id string) error {
	return a.sendToServer(map[string]interface{}{"type": "hard_delete", "id": id})
}

func (a *App) MovePage(id string, newParentID string) error {
	var parentPtr *string
	if newParentID != "" {
		parentPtr = &newParentID
	}
	return a.sendToServer(map[string]interface{}{
		"type":      "move",
		"id":        id,
		"parent_id": parentPtr,
	})
}

// --- Card Operations ---

func (a *App) FetchCards(pageID string) error {
	return a.sendToServer(map[string]interface{}{
		"type":    "fetch_cards",
		"page_id": pageID,
	})
}

func (a *App) AddCard(pageID string, cardType string, content string) error {
	card := Card{
		ID:        uuid.New().String(),
		PageID:    pageID,
		Type:      cardType,
		Content:   content,
		SortOrder: 0,
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	return a.sendToServer(map[string]interface{}{
		"type": "add_card",
		"item": card,
	})
}

func (a *App) UpdateCard(id string, pageID string, content string) error {
	a.cardCacheMutex.RLock()
	currentRevision := 0
	if cards, ok := a.cardCache[pageID]; ok {
		for _, c := range cards {
			if c.ID == id {
				currentRevision = c.Revision
				break
			}
		}
	}
	a.cardCacheMutex.RUnlock()

	card := Card{
		ID:        id,
		PageID:    pageID,
		Content:   content,
		UpdatedAt: time.Now().Format(time.RFC3339),
		Revision:  currentRevision + 1,
	}

	return a.sendToServer(map[string]interface{}{
		"type": "update_card",
		"item": card,
	})
}

func (a *App) DeleteCard(id string, pageID string) error {
	return a.sendToServer(map[string]interface{}{
		"type":     "delete_card",
		"id":       id,
		"page_id":  pageID,
	})
}

func (a *App) ReorderCards(pageID string, cardIds []string) error {
	return a.sendToServer(map[string]interface{}{
		"type":     "reorder_cards",
		"page_id":  pageID,
		"order":    cardIds,
	})
}

// --- Image Storage ---

// getImageDir returns the image directory for the active workspace.
// Creates it if it doesn't exist.
func (a *App) getImageDir() (string, error) {
	if a.imageDir != "" {
		return a.imageDir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home dir: %w", err)
	}

	wsName := a.activeWorkspace
	if wsName == "" {
		wsName = "default"
	}
	wsName = strings.ReplaceAll(wsName, "/", "_")
	wsName = strings.ReplaceAll(wsName, "\\", "_")

	dir := filepath.Join(home, ".cero", wsName+"_images")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create image dir: %w", err)
	}
	a.imageDir = dir
	return dir, nil
}

// SaveImage saves a base64-encoded image to the workspace image directory.
// Returns the relative file path to store in card.content.
func (a *App) SaveImage(base64Data string, originalName string) (string, error) {
	dir, err := a.getImageDir()
	if err != nil {
		return "", err
	}

	// Determine extension from original name or default to .png
	ext := ".png"
	if idx := strings.LastIndex(originalName, "."); idx > 0 {
		ext = originalName[idx:]
	}

	filename := uuid.New().String() + ext

	// Decode base64 (handle data URLs like "data:image/png;base64,...")
	if idx := strings.Index(base64Data, ","); idx > 0 {
		base64Data = base64Data[idx+1:]
	}

	decoded, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, decoded, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Return relative path (just the filename, since image dir is derived from workspace)
	return filename, nil
}

// GetImage returns the file path for a given image filename in the workspace image dir.
func (a *App) GetImage(filename string) (string, error) {
	dir, err := a.getImageDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("image not found: %s", filename)
	}
	return path, nil
}

// DeleteImage removes an image file from the workspace image directory.
func (a *App) DeleteImage(filename string) error {
	dir, err := a.getImageDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, filename)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete image: %w", err)
	}
	return nil
}
