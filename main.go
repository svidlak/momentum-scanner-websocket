package main

import "fmt"

func main() {
	wsConn, err := GetJWT()
	fmt.Println(wsConn, err)
	// err := godotenv.Load()
	// if err != nil {
	// 	fmt.Println("Error loading .env file")
	// }
	//
	// InitDiscordBot()
	// go startStockTitanConnection()
	//
	// http.HandleFunc("/ws", handleClientConnections)
	// http.HandleFunc("/alive", serverStatusHandler)
	//
	// log.Println("WebSocket proxy is running on :8080")
	// log.Fatal(http.ListenAndServe(":8080", nil))
}
