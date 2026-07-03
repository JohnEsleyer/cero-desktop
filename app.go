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
)

// DbItem represents the database model matching Dart's DbItem
type DbItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	UpdatedAt string `json:"updatedAt"`
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
	dbItems      []DbItem
	dbItemsMutex sync.RWMutex

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
		dbItems:       []DbItem{},
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

// GetDbItems returns the currently cached database items
func (a *App) GetDbItems() []DbItem {
	a.dbItemsMutex.RLock()
	defer a.dbItemsMutex.RUnlock()
	return a.dbItems
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

		a.discoveredMux.Lock()
		a.udpListener = conn
		a.discoveredMux.Unlock()

		// Start a goroutine to prune old discovered devices (inactive for > 8s)
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
				// Will trigger when connection is closed
				break
			}

			var payload struct {
				App        string `json:"app"`
				Port       int    `json:"port"`
				IP         string `json:"ip"`
				DeviceName string `json:"deviceName"`
			}

			if err := json.Unmarshal(buf[:n], &payload); err == nil && payload.App == "pocketdatabase" {
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
					// Notify frontend that a new device is discovered
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

// ConnectToDevice establishes connection to Flutter server
func (a *App) ConnectToDevice(ip string, port int) error {
	a.Disconnect()

	a.setConnectionStatus("connecting")
	wsURL := fmt.Sprintf("ws://%s:%d/ws", ip, port)

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
	a.setConnectionStatus("connected")

	// Start reading loop in a goroutine
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
		// Can't invoke setConnectionStatus directly here because of mutex, so emit manually or run in goroutine
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
		case "sync":
			var items []DbItem
			if err := json.Unmarshal(envelope.Data, &items); err == nil {
				a.dbItemsMutex.Lock()
				a.dbItems = items
				a.dbItemsMutex.Unlock()
				runtime.EventsEmit(a.ctx, "db-update", a.dbItems)
			}
		case "add":
			var item DbItem
			if err := json.Unmarshal(envelope.Item, &item); err == nil {
				a.dbItemsMutex.Lock()
				// Prevent duplicates
				exists := false
				for _, existing := range a.dbItems {
					if existing.ID == item.ID {
						exists = true
						break
					}
				}
				if !exists {
					a.dbItems = append(a.dbItems, item)
				}
				a.dbItemsMutex.Unlock()
				runtime.EventsEmit(a.ctx, "db-update", a.dbItems)
			}
		case "update":
			var item DbItem
			if err := json.Unmarshal(envelope.Item, &item); err == nil {
				a.dbItemsMutex.Lock()
				for i, existing := range a.dbItems {
					if existing.ID == item.ID {
						a.dbItems[i] = item
						break
					}
				}
				a.dbItemsMutex.Unlock()
				runtime.EventsEmit(a.ctx, "db-update", a.dbItems)
			}
		case "delete":
			a.dbItemsMutex.Lock()
			for i, existing := range a.dbItems {
				if existing.ID == envelope.ID {
					a.dbItems = append(a.dbItems[:i], a.dbItems[i+1:]...)
					break
				}
			}
			a.dbItemsMutex.Unlock()
			runtime.EventsEmit(a.ctx, "db-update", a.dbItems)
		}
	}
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

// AddItem requests the Flutter server to add a new item
func (a *App) AddItem(title string, content string) error {
	item := DbItem{
		ID:        uuid.New().String(),
		Title:     title,
		Content:   content,
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	message := map[string]interface{}{
		"type": "add",
		"item": item,
	}

	return a.sendToServer(message)
}

// UpdateItem requests the Flutter server to update an item
func (a *App) UpdateItem(id string, title string, content string) error {
	item := DbItem{
		ID:        id,
		Title:     title,
		Content:   content,
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	message := map[string]interface{}{
		"type": "update",
		"item": item,
	}

	return a.sendToServer(message)
}

// DeleteItem requests the Flutter server to delete an item
func (a *App) DeleteItem(id string) error {
	message := map[string]interface{}{
		"type": "delete",
		"id":   id,
	}

	return a.sendToServer(message)
}
