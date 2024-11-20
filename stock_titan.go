package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

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

	conn, _, err := websocket.DefaultDialer.Dial("wss://ws.stocktitan.net:9022/"+jwt, header)
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
			startStockTitanConnection()
		}

		if partialMessage.Header.Type == "journal" {
			lastWebSocketMessage = messageBytes
			sendDiscordMessage(messageBytes)
			broadcastToClients(messageBytes)
		}
	}
}
