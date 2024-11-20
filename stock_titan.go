package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func startStockTitanConnection() {
	var partialMessage struct {
		Header struct {
			Type string `json:"type"`
		} `json:"header"`
	}

	header := http.Header{
		"Host":            {"ws.stocktitan.net:9022"},
		"Pragma":          {"no-cache"},
		"Cache-Control":   {"no-cache"},
		"User-Agent":      {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"},
		"Origin":          {"https://www.stocktitan.net"},
		"Accept-Encoding": {"gzip, deflate, br, zstd"},
		"Accept-Language": {"en-US,en;q=0.9"},
	}
	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOjExNjY4NCwidXNlcm5hbWUiOiJzdmlkbGFrayIsInByZW1pdW0iOjEsInByZW1pdW1FeHBpcmF0aW9uIjoxNzMyNjcxNTg1LCJleHAiOjE3MzI1ODUxODUsImlhdCI6MTczMTk4MDM4NX0.7kXyeDVSvPADWmmMBAibLD2ZkHR4XLupa00-S9SN5fY"

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
			continue
		}

		if partialMessage.Header.Type == "journal" {
			lastWebSocketMessage = messageBytes
			sendDiscordMessage(messageBytes)
			broadcastToClients(messageBytes)
		}
	}
}
