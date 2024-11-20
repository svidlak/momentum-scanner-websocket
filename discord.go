package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
)

const (
	BullChannelId = "1308554651783659591"
	BearChannelId = "1308554686739124344"
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

var dgInstance *discordgo.Session

func sendDiscordMessage(messageBytes []byte) {
	var data WebSocketMessage
	err := json.Unmarshal(messageBytes, &data)
	if err != nil {
		log.Printf("Failed to format json message: %v\n", err)
		return
	}
	msg := formatMessage(data)

	if data.Payload.Volume > 500000 && data.Payload.Price > 5 {
		if data.Payload.PriceChangeRatio > 0 {
			sendMessage(BullChannelId, msg)
		} else {
			sendMessage(BearChannelId, msg)
		}
	}
}

func formatMessage(data WebSocketMessage) *discordgo.MessageEmbed {
	// Format market cap
	marketCapFormatted := fmt.Sprintf("$%.2f", data.Payload.MarketCap)

	// Default news title and URL
	newsTitle := "No news available"
	newsUrl := ""

	if len(data.Payload.News) > 0 {
		newsTitle = data.Payload.News[0].Title
		newsUrl = "https://www.stocktitan.net/news/" + data.Payload.Symbol + "/" + data.Payload.News[0].InternalURL + ".html"
	}

	// Determine the color based on PriceChangeRatio
	embedColor := 0xFF0000
	if data.Payload.PriceChangeRatio > 0 {
		embedColor = 0x00FF00 // Green if price change ratio is positive
	}
	// Create the embed
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("🚨 Stock Alert: %s", data.Payload.Symbol),
		Description: fmt.Sprintf("**Price**: `$%.2f`\n**Price Change**: `%.2f%%`\n**Market Cap**: `%s`\n**Volume**: `%d`\n**Shares Float**: `%d`",
			data.Payload.Price, data.Payload.PriceChangeRatio*100, marketCapFormatted, data.Payload.Volume, int(data.Payload.SharesFloat)),
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

// sendMessage posts a message to a Discord channel
func sendMessage(channelID string, message *discordgo.MessageEmbed) {
	_, err := dgInstance.ChannelMessageSendEmbed(channelID, message)
	if err != nil {
		log.Printf("Failed to send message: %v\n", err)
	}
}

func InitDiscordBot() {
	log.Println("Starting discord integration")
	BotToken := os.Getenv("DISCORD_BOT_TOKEN")
	dg, err := discordgo.New("Bot " + BotToken)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}

	if err != nil {
		log.Fatalf("Error opening Discord connection: %v", err)
	}

	dgInstance = dg
}
