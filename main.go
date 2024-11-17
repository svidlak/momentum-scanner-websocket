package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	clients      = make(map[*websocket.Conn]bool)
	clientsMutex sync.Mutex
	upgrader     = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func main() {
	go startStockTitanConnection()

	http.HandleFunc("/ws", handleClientConnections)
	http.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // Optional: Explicitly set status code to 200
		fmt.Fprintln(w, "OK")
	})
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

	// Ensure client is removed on disconnect
	defer func() {
		clientsMutex.Lock()
		delete(clients, conn)
		clientsMutex.Unlock()
		log.Println("Client disconnected")
	}()

	// Set up keep-alive with Ping messages
	pingTicker := time.NewTicker(30 * time.Second) // Adjust interval as needed
	defer pingTicker.Stop()

	// Set the initial read deadline
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // Adjust as needed

	// Handle Pong messages to reset the deadline
	conn.SetPongHandler(func(appData string) error {
		log.Println("Received Pong from client")
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // Extend deadline on pong
		return nil
	})

	// Goroutine to send periodic pings
	go func() {
		for range pingTicker.C {
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("Ping error:", err)
				return // Exit the goroutine on error
			}
			log.Println("Ping sent to client")
		}
	}()

	// Main loop to read messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break // Exit the loop on read error
		}
		// Reset the read deadline after a successful read
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		log.Println("Message received from client")
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
		_, messageBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading from StockTitan WebSocket:", err)
			break
		}

		if err := json.Unmarshal(messageBytes, &partialMessage); err != nil {
			log.Printf("Error unmarshalling message header: %v", err)
			continue
		}

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