package websocket

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub manages WebSocket connections, each identified by IP
type Hub struct {
	clients    map[string]*websocket.Conn // IP -> Connection
	broadcast  chan []byte                // Optional: Broadcast to all
	register   chan clientRegistration
	unregister chan string // IP
	mu         sync.Mutex
}

// clientRegistration represents a new WebSocket connection
type clientRegistration struct {
	conn *websocket.Conn
	ip   string
}

// NewHub initializes the WebSocket hub
func NewHub() *Hub {

	fmt.Println("HERE >> HUB")
	return &Hub{
		clients:    make(map[string]*websocket.Conn),
		broadcast:  make(chan []byte),
		register:   make(chan clientRegistration),
		unregister: make(chan string),
	}
}

// Run starts the hub loop for handling registrations and messaging
func (h *Hub) Run() {
	fmt.Println("HERE >> RUN")
	for {
		select {
		case reg := <-h.register:
			h.mu.Lock()
			h.clients[reg.ip] = reg.conn
			h.mu.Unlock()

		case ip := <-h.unregister:
			h.mu.Lock()
			if conn, ok := h.clients[ip]; ok {
				conn.Close()
				delete(h.clients, ip)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.Lock()
			for _, conn := range h.clients {
				conn.WriteMessage(websocket.TextMessage, message)
			}
			h.mu.Unlock()
		}
	}
}

// HandleWebSocket upgrades the HTTP request to a WebSocket and registers the client
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	fmt.Println("HERE >> HANDLE WEBSOCKET")
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}


	// Extract client IP (preferred: query param `?ip=...`)
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		ip = r.RemoteAddr // fallback
	}

	h.register <- clientRegistration{conn: conn, ip: ip}

	// Handle disconnect
	go func() {
		defer func() {
			h.unregister <- ip
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

// Send broadcasts a message to all clients
func (h *Hub) Send(message []byte) {
	h.broadcast <- message
}

// SendToIP sends a message to a specific client identified by IP
func (h *Hub) SendToIP(ip string, message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conn, ok := h.clients[ip]; ok {
		conn.WriteMessage(websocket.TextMessage, message)
	}
}
