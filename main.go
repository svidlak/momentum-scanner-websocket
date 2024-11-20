package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
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

var lastWebSocketMessage json.RawMessage

type ServerStatus struct {
	CPUUsage         string          `json:"cpu_usage"`
	MemUsed          string          `json:"memory_used"`
	MemTotal         string          `json:"memory_total"`
	Alive            bool            `json:"alive"`
	LastMessage      json.RawMessage `json:"last_message"`
	ConnectedClients int             `json:"connected_clients"`
}

func main() {
	go startStockTitanConnection()

	http.HandleFunc("/ws", handleClientConnections)

	http.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
		cpuPercent, _ := cpu.Percent(0, false)
		memStats, _ := mem.VirtualMemory()

		cpuUsage := fmt.Sprintf("%.2f%%", cpuPercent[0])
		memUsed := formatMemory(memStats.Used)
		memTotal := formatMemory(memStats.Total)

		status := ServerStatus{
			CPUUsage:         cpuUsage,
			MemUsed:          memUsed,
			MemTotal:         memTotal,
			Alive:            true,
			LastMessage:      lastWebSocketMessage,
			ConnectedClients: len(clients),
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(status)
	})

	log.Println("WebSocket proxy is running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func formatMemory(bytes uint64) string {
	if bytes >= 1<<30 {
		return fmt.Sprintf("%.2f GB", float64(bytes)/(1<<30))
	} else if bytes >= 1<<20 {
		return fmt.Sprintf("%.2f MB", float64(bytes)/(1<<20))
	}
	return fmt.Sprintf("%d KB", bytes/1024)
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
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
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
		"Host":                  {"ws.stocktitan.net:9022"},
		"Connection":            {"Upgrade"},
		"Pragma":                {"no-cache"},
		"Cache-Control":         {"no-cache"},
		"User-Agent":            {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"},
		"Upgrade":               {"websocket"},
		"Origin":                {"https://www.stocktitan.net"},
		"Sec-WebSocket-Version": {"13"},
		"Accept-Encoding":       {"gzip, deflate, br, zstd"},
		"Accept-Language":       {"en-US,en;q=0.9"},
		// Uncomment and set a proper Sec-WebSocket-Key if needed
		// "Sec-WebSocket-Key":    {"IbEsPZXh+6/al/MlER8E+A=="},
		"Sec-WebSocket-Extensions": {"permessage-deflate; client_max_window_bits"},
	}

	// Connect to StockTitan WebSocket
	conn, _, err := websocket.DefaultDialer.Dial("wss://ws.stocktitan.net:9022/eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOjExNjY4NCwidXNlcm5hbWUiOiJzdmlkbGFrayIsInByZW1pdW0iOjEsInByZW1pdW1FeHBpcmF0aW9uIjowLCJleHAiOjE3MzI1Njg1OTEsImlhdCI6MTczMTk2Mzc5MX0.MD6X39CW-NTrjzGbgqGoW9ovzkAYYMgmTzBlCMNCREg", header)
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
			lastWebSocketMessage = messageBytes
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
