package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	clients      = make(map[*websocket.Conn]bool) // Connected clients
	clientsMutex sync.Mutex                       // Mutex to protect the clients map
	upgrader     = websocket.Upgrader{            // WebSocket upgrader
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func main() {
	// Start the StockTitan WebSocket connection in a separate goroutine
	go startStockTitanConnection()

	// Start the HTTP server for client WebSocket connections
	http.HandleFunc("/ws", handleClientConnections)
	log.Println("WebSocket proxy is running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleClientConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close()

	// Add the new client to the clients map
	clientsMutex.Lock()
	clients[conn] = true
	clientsMutex.Unlock()

	log.Println("New client connected")

	// Remove the client when the connection closes
	defer func() {
		clientsMutex.Lock()
		delete(clients, conn)
		clientsMutex.Unlock()
		log.Println("Client disconnected")
	}()

	// Keep the connection open and listen for client messages
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

func startStockTitanConnection() {
	var partialMessage struct {
		Header struct {
			Type string `json:"type"`
		} `json:"header"`
	}

	header := http.Header{
		"Origin": {"https://www.stocktitan.net"},
	}
	// Connect to StockTitan WebSocket
	conn, _, err := websocket.DefaultDialer.Dial("wss://ws.stocktitan.net:9021/null", header)
	if err != nil {
		log.Fatal("Error connecting to StockTitan WebSocket:", err)
	}
	defer conn.Close()

	log.Println("Connected to StockTitan WebSocket")

	for {
		// Read a message from StockTitan
		_, messageBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading from StockTitan WebSocket:", err)
			break
		}

		// Parse only the "header" part

		if err := json.Unmarshal(messageBytes, &partialMessage); err != nil {
			log.Printf("Error unmarshalling message header: %v", err)
			continue
		}

		// Check if "header.type" is "journal"
		if partialMessage.Header.Type == "journal" {
			log.Println("Broadcasting journal message")
			broadcastToClients(messageBytes)
		}
	}
}

func broadcastToClients(message []byte) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Println("Error writing to client:", err)
			client.Close()
			delete(clients, client)
		}
	}
}
