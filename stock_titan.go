package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

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

	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal("Error loading New York location:", err)
	}

	for {
		currentTime := time.Now().In(nyc) // Get current time in New York timezone

		// Define the pre-market start time (4:00 AM EST/EDT) and after-market close time (8:00 PM EST/EDT)
		startTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 3, 45, 0, 0, nyc) // 4:00 AM EST/EDT
		endTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 20, 5, 0, 0, nyc)   // 8:00 PM EST/EDT

		// Adjust for the next day if the current time is past after-market close (8:00 PM)
		if currentTime.After(endTime) {
			startTime = startTime.Add(24 * time.Hour)
			endTime = endTime.Add(24 * time.Hour)
		}

		// Sleep until the pre-market opens (4:00 AM EST/EDT) or after-market closes (8:00 PM EST/EDT)
		if currentTime.Before(startTime) {
			// If it's before pre-market opening, sleep until 4:00 AM EST/EDT
			sleepDuration := time.Until(startTime)
			log.Printf("Sleeping for %v until pre-market opens at 4:00 AM EST/EDT...\n", sleepDuration)
			time.Sleep(sleepDuration)
		} else if currentTime.After(endTime) {
			// If it's after after-market close, sleep until 4:00 AM EST/EDT the next day
			sleepDuration := time.Until(startTime.Add(24 * time.Hour))
			log.Printf("Sleeping for %v until pre-market opens at 4:00 AM EST/EDT next day...\n", sleepDuration)
			time.Sleep(sleepDuration)
		}
		// WebSocket connection should be established only during the allowed time window
		wsConn, err := GetJWT()
		fmt.Println(wsConn, err)
		conn, _, err := websocket.DefaultDialer.Dial(wsConn, header)
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

			currentTime = time.Now().In(nyc)
			// If after the after-market (after 8:00 PM), close the WebSocket connection
			if currentTime.After(endTime) {
				log.Println("After market close. Closing WebSocket connection.")
				conn.Close()
				break // Break the loop to close and reconnect after sleeping
			}
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
				go sendDiscordMessage(messageBytes)
				go broadcastToClients(messageBytes)
			}
		}
	}
}
