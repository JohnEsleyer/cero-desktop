package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/net/ipv4"
)

// DbPage represents the database page model matching Dart's DbPage
type DbPage struct {
	ID         string  `json:"id"`
	ParentID   *string `json:"parent_id"` // Nullable
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Emoji      string  `json:"emoji"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
	IsArchived int     `json:"is_archived"` // 0 or 1
	SortOrder  int     `json:"sort_order"`
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
	connStatus string // "disconnected", "connecting", "connected"
	connMutex  sync.Mutex
	
	// Database state cache
	dbPages      []DbPage
	dbPagesMutex sync.RWMutex

	// UDP Discovery state
	udpListener  *net.UDPConn
	isDiscovering bool
	discoveredMap map[string]DiscoveredDevice
	discoveredMux sync.Mutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		connStatus:    "disconnected",
		dbPages:       []DbPage{},
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

// setConnectionStatus updates status and notifies frontend
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

// --- UDP Discovery Service ---

// StartUdpDiscovery starts listening for UDP beacons
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

		// Join multicast group for better router traversal
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

// StopUdpDiscovery stops listening for UDP beacons
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

// GetDiscoveredDevices returns current discovered servers
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

// ConnectToDevice establishes connection to Flutter server with auth PIN
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
	// Don't set "connected" yet - wait for pairing approval from user
	a.setConnectionStatus("connecting")

	go a.readWebSocketLoop(conn)

	return nil
}

// Disconnect closes the current websocket connection
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
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
			Item json.RawMessage `json:"item"`
			ID   string          `json:"id"`
		}

		if err := json.Unmarshal(message, &envelope); err != nil {
			fmt.Printf("Error unmarshalling WebSocket payload: %v\n", err)
			continue
		}

		switch envelope.Type {
		case "pairing_required":
			// Notify frontend that pairing is needed
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
			// Close connection since pairing was rejected
			conn.Close()
			return
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
			// Full sync needed after restore to get archived pages back in active view
			// The server will re-sync, but for now we can just emit a refresh event
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
		case "content":
			var contentPayload struct {
				ID      string `json:"id"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal(message, &contentPayload); err == nil {
				a.dbPagesMutex.Lock()
				for i, p := range a.dbPages {
					if p.ID == contentPayload.ID {
						a.dbPages[i].Content = contentPayload.Content
						break
					}
				}
				a.dbPagesMutex.Unlock()
				runtime.EventsEmit(a.ctx, "page-content", contentPayload)
			}
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
		}
	}
}

// Recursive helper to remove a page and all its nested subpages from cache
func removePageAndDescendants(pages []DbPage, targetID string) []DbPage {
	// Find all pages that are children of targetID
	var toDelete []string
	toDelete = append(toDelete, targetID)
	
	// Keep searching for nested children until no new ones are found
	added := true
	for added {
		added = false
		for _, page := range pages {
			if page.ParentID != nil {
				// If parent is in delete list, and this child is not already in delete list
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
	
	// Create new slice without items in toDelete list
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
func (a *App) AddPage(parentID string, title string, content string, emoji string) error {
	var parentPtr *string
	if parentID != "" {
		parentPtr = &parentID
	}

	page := DbPage{
		ID:         uuid.New().String(),
		ParentID:   parentPtr,
		Title:      title,
		Content:    content,
		Emoji:      emoji,
		CreatedAt:  time.Now().Format(time.RFC3339),
		UpdatedAt:  time.Now().Format(time.RFC3339),
		IsArchived: 0,
		SortOrder:  0,
	}

	message := map[string]interface{}{
		"type": "add",
		"item": page,
	}

	return a.sendToServer(message)
}

// UpdatePage requests the Flutter server to update a page
func (a *App) UpdatePage(id string, parentID string, title string, content string, emoji string) error {
	var parentPtr *string
	if parentID != "" {
		parentPtr = &parentID
	}

	page := DbPage{
		ID:         id,
		ParentID:   parentPtr,
		Title:      title,
		Content:    content,
		Emoji:      emoji,
		CreatedAt:  time.Now().Format(time.RFC3339), // Fallback, server will retain original
		UpdatedAt:  time.Now().Format(time.RFC3339),
		IsArchived: 0,
		SortOrder:  0,
	}

	message := map[string]interface{}{
		"type": "update",
		"item": page,
	}

	return a.sendToServer(message)
}

// DeletePage requests the Flutter server to archive a page
func (a *App) DeletePage(id string) error {
	message := map[string]interface{}{
		"type": "archive",
		"id":   id,
	}

	return a.sendToServer(message)
}

// RestorePage requests the Flutter server to restore an archived page
func (a *App) RestorePage(id string) error {
	message := map[string]interface{}{
		"type": "restore",
		"id":   id,
	}

	return a.sendToServer(message)
}

// HardDeletePage requests permanent deletion
func (a *App) HardDeletePage(id string) error {
	message := map[string]interface{}{
		"type": "hard_delete",
		"id":   id,
	}

	return a.sendToServer(message)
}

// MovePage moves a page to a new parent
func (a *App) MovePage(id string, newParentID string) error {
	var parentPtr *string
	if newParentID != "" {
		parentPtr = &newParentID
	}

	message := map[string]interface{}{
		"type":      "move",
		"id":        id,
		"parent_id": parentPtr,
	}

	return a.sendToServer(message)
}

// FetchPageContent requests full content for a specific page
func (a *App) FetchPageContent(id string) error {
	message := map[string]interface{}{
		"type": "fetch",
		"id":   id,
	}

	return a.sendToServer(message)
}
