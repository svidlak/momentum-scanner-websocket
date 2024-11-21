package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func GetJWT() (string, error) {
	// Context for chromedp
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Timeout for operations

	// Variables
	loginURL := "https://www.stocktitan.net/"
	redirectURL := "https://www.stocktitan.net/scanner/momentum"
	email := "svid320@gmail.com" // Replace with your email
	password := "8506214Ms"      // Replace with your password

	// Result to store the jsGlobals content
	var jsGlobalsContent string

	// Run chromedp tasks
	err := chromedp.Run(ctx,
		chromedp.Navigate(loginURL),                                 // Navigate to the login page
		chromedp.Click(`a.nav-link[data-bs-target="#login-modal"]`), // Click the login button
		chromedp.WaitVisible(`#login-modal`, chromedp.ByID),         // Wait for the login modal
		chromedp.WaitVisible(`#login-email`, chromedp.ByID),         // Wait for the email input field
		chromedp.WaitVisible(`#login-password`, chromedp.ByID),      // Wait for the password input field
		chromedp.WaitVisible(`#login-submit`, chromedp.ByID),        // Wait for the submit button
		chromedp.SendKeys(`#login-email`, email),                    // Enter email
		chromedp.SendKeys(`#login-password`, password),              // Enter password
		chromedp.Click(`#login-submit`),                             // Click the login button
		chromedp.Sleep(10*time.Second),
		chromedp.Navigate(redirectURL), // Navigate to the redirected page

		chromedp.Sleep(10*time.Second),
		chromedp.WaitReady(`body`, chromedp.ByQuery),                      // Wait for the page to be fully loaded after redirect
		chromedp.Evaluate(`JSON.stringify(jsGlobals)`, &jsGlobalsContent), // Get the jsGlobals variable
	)
	if err != nil {
		log.Fatalf("Chromedp failed: %v", err)
	}

	// // Save jsGlobals to a JSON file
	// err = saveToJSON("variables.json", jsGlobalsContent)
	// if err != nil {
	// 	log.Fatalf("Failed to save JSON: %v", err)
	// }

	return extractWSUrl(jsGlobalsContent)
}

// // saveToJSON saves a JSON string to a file
// func saveToJSON(filename, jsonContent string) error {
// 	var formattedJSON map[string]interface{}
// 	if err := json.Unmarshal([]byte(jsonContent), &formattedJSON); err != nil {
// 		return fmt.Errorf("invalid JSON format: %w", err)
// 	}
//
// 	file, err := os.Create(filename)
// 	if err != nil {
// 		return fmt.Errorf("failed to create file: %w", err)
// 	}
// 	defer file.Close()
//
// 	encoder := json.NewEncoder(file)
// 	encoder.SetIndent("", "  ") // Pretty print the JSON
// 	return encoder.Encode(formattedJSON)
// }

func extractWSUrl(jsonContent string) (string, error) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonContent), &data)
	if err != nil {
		log.Fatal("Error unmarshalling JSON: ", err)
		return "", err
	}

	// Extract the "jwt" and "ws" values
	// jwt, jwtOk := data["jwt"].(string)
	ws, wsOk := data["ws"].(string)

	// Check if the values exist and print them
	// if jwtOk && wsOk {
	// 	return ws + "/" + jwt, nil
	// } else {
	// 	log.Fatal("JWT or WS value not found")
	// 	return "", errors.New("failed to refresh JWT")
	// }
	//
	if wsOk {
		return ws, nil
	}
	return "", errors.New("parsing failed")
}
