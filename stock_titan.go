package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	preMarketStartHour   = 3  // 4:00 AM EST/EDT
	afterMarketCloseHour = 20 // 8:00 PM EST/EDT
)

func getNextMarketOpenTime(nyc *time.Location, currentTime time.Time) time.Time {
	// Define the pre-market start time (4:00 AM EST/EDT)
	return time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), preMarketStartHour, 45, 0, 0, nyc)
}

func getNextMarketCloseTime(nyc *time.Location, currentTime time.Time) time.Time {
	// Define the after-market close time (8:00 PM EST/EDT)
	return time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), afterMarketCloseHour, 5, 0, 0, nyc)
}

func sleepUntil(nextTime time.Time, condition string) {
	sleepDuration := time.Until(nextTime)
	log.Println(condition)
	log.Printf("Sleeping for %v until %v...\n", sleepDuration, nextTime)
	time.Sleep(sleepDuration)
}

func connectToStockTitanWebSocket(header http.Header) (*websocket.Conn, error) {
	wsConn, err := GetJWT()
	if err != nil {
		return nil, err
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsConn, header)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func startStockTitanConnection() {
	// Prepare WebSocket headers
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
		currentTime := time.Now().In(nyc)

		// Check if today is Saturday or Sunday
		if currentTime.Weekday() == time.Saturday || currentTime.Weekday() == time.Sunday {
			// Sleep until Monday
			nextMonday := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day()+int(time.Monday-currentTime.Weekday()), 3, 45, 0, 0, nyc)
			sleepUntil(nextMonday, "weekend")
			continue
		}

		// Calculate next market open and close times
		startTime := getNextMarketOpenTime(nyc, currentTime)
		endTime := getNextMarketCloseTime(nyc, currentTime)

		// Adjust for the next day if past after-market close
		if currentTime.After(endTime) {
			startTime = startTime.Add(24 * time.Hour)
			endTime = endTime.Add(24 * time.Hour)
		}

		// Sleep until market opens or closes
		if currentTime.Before(startTime) {
			sleepUntil(startTime, "before")
		} else if currentTime.After(endTime) {
			sleepUntil(startTime.Add(24*time.Hour), "after")
		}

		// Establish WebSocket connection during valid trading hours
		conn, err := connectToStockTitanWebSocket(header)
		if err != nil {
			log.Println("Error connecting to StockTitan WebSocket:", err)
			sendStatusMessage(1)
			time.Sleep(5 * time.Second)
			continue
		}
		defer conn.Close()

		log.Println("Connected to StockTitan WebSocket")
		sendStatusMessage(0)

		// Handle WebSocket messages
		var partialMessage struct {
			Header struct {
				Type string `json:"type"`
			} `json:"header"`
		}

		for {
			currentTime = time.Now().In(nyc)

			// If it's after the after-market, close the connection
			if currentTime.After(endTime) {
				log.Println("After market close. Closing WebSocket connection.")
				conn.Close()
				break
			}

			_, messageBytes, err := conn.ReadMessage()
			if err != nil {
				log.Println("Error reading from StockTitan WebSocket:", err)
				sendStatusMessage(1)
				conn.Close()
				break
			}

			// Process the received message
			if err := json.Unmarshal(messageBytes, &partialMessage); err != nil {
				log.Printf("Error unmarshalling message header: %v", err)
				continue
			}

			if partialMessage.Header.Type == "ping" {
				pongMessage := map[string]interface{}{
					"type": "pong",
				}

				pongBytes, err := json.Marshal(pongMessage)
				if err != nil {
					log.Printf("Error marshalling pong message: %v", err)
					continue
				}

				if err := conn.WriteMessage(websocket.TextMessage, pongBytes); err != nil {
					log.Printf("Error writing pong message: %v", err)
					continue
				}

			}

			if partialMessage.Header.Type == "journal" {
				lastWebSocketMessage = messageBytes
				go sendDiscordMessage(messageBytes)
				go broadcastToClients(messageBytes)
			}
		}
	}
}
