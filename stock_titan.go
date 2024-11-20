package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func startStockTitanConnection() {
	var partialMessage struct {
		Header struct {
			Type string `json:"type"`
		} `json:"header"`
	}

	jwt := os.Getenv("JWT_TOKEN")

	header := http.Header{
		"Host":            {"ws.stocktitan.net:9022"},
		"Pragma":          {"no-cache"},
		"Cache-Control":   {"no-cache"},
		"User-Agent":      {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"},
		"Origin":          {"https://www.stocktitan.net"},
		"Accept-Encoding": {"gzip, deflate, br, zstd"},
		"Accept-Language": {"en-US,en;q=0.9"},
	}

	for {
		// Try to establish WebSocket connection
		conn, _, err := websocket.DefaultDialer.Dial("wss://ws.stocktitan.net:9022/"+jwt, header)
		if err != nil {
			log.Println("Error connecting to StockTitan WebSocket:", err)
			sendStatusMessage(1)        // Send failure status
			time.Sleep(5 * time.Second) // Wait before retrying
			continue                    // Retry connection
		}
		defer conn.Close() // Ensure connection is closed after reading

		log.Println("Connected to StockTitan WebSocket")
		sendStatusMessage(0) // Send success status

		// Handle messages from StockTitan WebSocket
		for {
			_, messageBytes, err := conn.ReadMessage() // Read message from WebSocket
			if err != nil {
				log.Println("Error reading from StockTitan WebSocket:", err)
				sendStatusMessage(1) // Send failure status
				conn.Close()         // Close the connection before reconnecting
				break                // Break out of the inner loop to reconnect
			}

			// Unmarshal and process the message
			if err := json.Unmarshal(messageBytes, &partialMessage); err != nil {
				log.Printf("Error unmarshalling message header: %v", err)
				continue // Skip this message and keep reading
			}

			if partialMessage.Header.Type == "journal" {
				lastWebSocketMessage = messageBytes
				sendDiscordMessage(messageBytes)
				broadcastToClients(messageBytes)
			}
		}
	}
}
