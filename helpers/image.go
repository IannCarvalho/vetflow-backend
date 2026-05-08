package helpers

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// DownloadImageAsBase64 downloads an image from URL and returns it as base64 string
func DownloadImageAsBase64(imageURL string) (string, error) {
	if imageURL == "" {
		return "", nil
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make request
	resp, err := client.Get(imageURL)
	if err != nil {
		log.Printf("Error downloading image from %s: %v", imageURL, err)
		return "", err
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error downloading image: status code %d", resp.StatusCode)
		return "", fmt.Errorf("failed to download image: status code %d", resp.StatusCode)
	}

	// Read image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading image data: %v", err)
		return "", err
	}

	// Limit size to 5MB
	if len(imageData) > 5*1024*1024 {
		log.Printf("Image too large: %d bytes", len(imageData))
		return "", fmt.Errorf("image too large")
	}

	// Convert to base64
	base64String := base64.StdEncoding.EncodeToString(imageData)
	return fmt.Sprintf("data:%s;base64,%s", resp.Header.Get("Content-Type"), base64String), nil
}
