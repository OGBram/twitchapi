package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
)

const channelName = "zackrawrr" // Channel to monitor

// getOAuthToken fetches an OAuth token from Twitch.
func getOAuthToken(clientID, clientSecret string) (string, error) {
	client := resty.New()

	resp, err := client.R().
		SetFormData(map[string]string{
			"client_id":     clientID,
			"client_secret": clientSecret,
			"grant_type":    "client_credentials",
		}).
		Post("https://id.twitch.tv/oauth2/token")

	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return "", err
	}

	token, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("failed to retrieve access token")
	}

	return token, nil
}

// getChannelData retrieves data about the specified channel.
func getChannelData(accessToken, clientID string) (map[string]interface{}, error) {
	client := resty.New()

	resp, err := client.R().
		SetHeader("Client-ID", clientID).
		SetHeader("Authorization", "Bearer "+accessToken).
		SetQueryParam("login", channelName).
		Get("https://api.twitch.tv/helix/users")

	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}

	data, ok := result["data"].([]interface{})
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("no data found for channel")
	}

	channelData, ok := data[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data format for channel")
	}

	// Print the channel data
	fmt.Printf("Channel Data: %+v\n", channelData)

	return channelData, nil
}

// isChannelLive checks if the specified channel is currently live.
func isChannelLive(accessToken, clientID string) (bool, error) {
	client := resty.New()

	channelData, err := getChannelData(accessToken, clientID)
	if err != nil {
		return false, err
	}

	userID, ok := channelData["id"].(string)
	if !ok {
		return false, fmt.Errorf("unable to retrieve channel ID")
	}

	resp, err := client.R().
		SetHeader("Client-ID", clientID).
		SetHeader("Authorization", "Bearer "+accessToken).
		SetQueryParam("user_id", userID).
		Get("https://api.twitch.tv/helix/streams")

	if err != nil {
		return false, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return false, err
	}

	data, ok := result["data"].([]interface{})
	if !ok {
		return false, fmt.Errorf("invalid data format for streams")
	}

	return len(data) > 0, nil
}

// monitorChannel checks if the channel is live and logs the status.
func monitorChannel(clientID, clientSecret string) {
	accessToken, err := getOAuthToken(clientID, clientSecret)
	if err != nil {
		log.Printf("Error fetching OAuth token: %v", err)
		return
	}

	live, err := isChannelLive(accessToken, clientID)
	if err != nil {
		log.Printf("Error checking if channel is live: %v", err)
		return
	}

	if live {
		fmt.Println("The channel is live!")
	} else {
		fmt.Println("The channel is not live.")
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Fatalf("CLIENT_ID and CLIENT_SECRET must be set in the environment")
	}

	// Monitor the channel every minute
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Start monitoring the channel in a separate goroutine
	go func() {
		for range ticker.C {
			monitorChannel(clientID, clientSecret)
		}
	}()

	// Initial run
	monitorChannel(clientID, clientSecret)
	getChannelData(clientID, clientSecret)

	// Block forever
	select {}
}
