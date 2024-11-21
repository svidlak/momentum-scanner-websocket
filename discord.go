package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

const (
	BullChannelId = "1309202216967602218"
	BearChannelId = "1309202267160838144"
	NewsChannelId = "1309202734863613952"

	ServerStatusChanel = "1308874016005689345"
	userId             = "208296432371761152"
)

type WebSocketMessage struct {
	Header struct {
		Type string `json:"type"`
	} `json:"header"`
	Payload struct {
		Date        string `json:"date"`
		Symbol      string `json:"symbol"`
		InternalURL string `json:"internal_url"`
		News        []struct {
			Summary *struct {
				En struct {
					Summary  string   `json:"summary"`
					Positive []string `json:"positive"`
					Negative []string `json:"negative"`
					FAQ      []struct {
						Q string `json:"q"`
						A string `json:"a"`
					} `json:"faq"`
				} `json:"en"`
			} `json:"summary"`
			InternalURL string `json:"internal_url"`
			Title       string `json:"title"`
		} `json:"news"`
		PriceChangeRatio float64 `json:"price_change_ratio"`
		Price            float64 `json:"price"`
		MarketCap        float64 `json:"market_cap"`
		SharesFloat      float64 `json:"shares_float"`
		Volume           int     `json:"volume"`
		AlertCount       int     `json:"alert_count"` // Nested 'en' for the summary
		// Pointer to handle null or missing
	} `json:"payload"`
}

var (
	childDgInstance  *discordgo.Session
	masterDgInstance *discordgo.Session
)

func sendStatusMessage(messageType int) {
	var message string
	if messageType == 1 {
		message = "⚠️  **WebSocket Disconnected:** Retrying to reconnect...\n<@" + userId + ">!"
	}

	if messageType == 0 {
		message = "✅ **WebSocket Status:** Online."
	}

	masterDgInstance.ChannelMessageSend(ServerStatusChanel, message)
}

func sendDiscordMessage(messageBytes []byte) {
	var data WebSocketMessage
	err := json.Unmarshal(messageBytes, &data)
	if err != nil {
		log.Printf("Failed to format json message: %v\n", err)
		return
	}

	if data.Payload.News != nil && len(data.Payload.News) > 0 && data.Payload.AlertCount == 1 {
		msg := formatMessage(data, 0)
		sendMessage(NewsChannelId, msg)
	}

	if data.Payload.Volume > 500000 && data.Payload.Price > 5 {
		if data.Payload.PriceChangeRatio > 0 {
			msg := formatMessage(data, 2)
			sendMessage(BullChannelId, msg)
		} else {
			msg := formatMessage(data, 1)
			sendMessage(BearChannelId, msg)
		}
	}
}

func formatMessage(data WebSocketMessage, messageType int) *discordgo.MessageEmbed {
	// Default news title and URL
	newsTitle := "No news available"
	newsUrl := ""

	if len(data.Payload.News) > 0 {
		newsTitle = data.Payload.News[0].Title
		newsUrl = "https://www.stocktitan.net/news/" + data.Payload.Symbol + "/" + data.Payload.News[0].InternalURL + ".html"
	}

	var embedColor int

	switch messageType {
	case 1:
		embedColor = 0xFF0000 // Red
	case 2:
		embedColor = 0x00FF00 // Green
	case 0:
		embedColor = 0x0000FF // Blue
	}

	marketCapFormatted := FormatNumber(int64(data.Payload.MarketCap))
	volumeFormatted := FormatNumber(int64(data.Payload.Volume)) // Volume is already int, so just format
	sharesFloatFormatted := FormatNumber(int64(data.Payload.SharesFloat))

	// Create the embed
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("🚨 Stock Alert: %s", data.Payload.Symbol),
		Description: fmt.Sprintf("**Price**: `$%.2f`\n**Price Change**: `%.2f%%`\n**Market Cap**: `%s`\n**Volume**: `%s`\n**Shares Float**: `%s`",
			data.Payload.Price, data.Payload.PriceChangeRatio*100, marketCapFormatted, volumeFormatted, sharesFloatFormatted),
		Color: embedColor, // Set the color of the embed (hexadecimal, e.g., orange)
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "📢 News",
				Value: fmt.Sprintf("[%s](%s)", newsTitle, newsUrl),
			},
			{
				Name:   "🕒 Date",
				Value:  data.Payload.Date,
				Inline: true,
			},
			{
				Name:   "📊 Alerts Triggered",
				Value:  fmt.Sprintf("%d", data.Payload.AlertCount),
				Inline: true,
			},
		},
	}

	return embed
}

func FormatNumber(n int64) string {
	strNum := strconv.FormatInt(n, 10)
	return AddCommas(strNum)
}

func AddCommas(numStr string) string {
	result := ""
	count := 0

	for i := len(numStr) - 1; i >= 0; i-- {
		count++
		result = string(numStr[i]) + result
		if count%3 == 0 && i != 0 {
			result = "," + result
		}
	}
	return result
}

// sendMessage posts a message to a Discord channel
func sendMessage(channelID string, message *discordgo.MessageEmbed) {
	_, err := childDgInstance.ChannelMessageSendEmbed(channelID, message)
	if err != nil {
		log.Printf("Failed to send message: %v\n", err)
	}
}

func InitDiscordBot() {
	log.Println("Starting discord integration")
	childBotToken := os.Getenv("CHILD_DISCORD_BOT_TOKEN")
	masterBotToken := os.Getenv("MAIN_DISCORD_BOT_TOKEN")

	childDgInstanceCon, err := discordgo.New("Bot " + childBotToken)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}

	if err != nil {
		log.Fatalf("Error opening Discord connection: %v", err)
	}

	masterDgInstanceCon, err := discordgo.New("Bot " + masterBotToken)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}

	if err != nil {
		log.Fatalf("Error opening Discord connection: %v", err)
	}

	masterDgInstance = masterDgInstanceCon
	childDgInstance = childDgInstanceCon
}
