package helpers

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

// maxImageSize is the maximum allowed image size in bytes (5MB).
const maxImageSize = 5 * 1024 * 1024

// DownloadImageAsBase64 downloads an image from URL and returns it as base64 string
func DownloadImageAsBase64(imageURL string) (string, error) {
	if imageURL == "" {
		return "", nil
	}

	// Validate the URL before fetching: require https and a non-empty host.
	u, err := url.Parse(imageURL)
	if err != nil {
		log.Printf("Error parsing image URL: %v", err)
		return "", fmt.Errorf("invalid image URL: %w", err)
	}
	if u.Scheme != "https" {
		log.Printf("Rejected image URL with disallowed scheme %q", u.Scheme)
		return "", fmt.Errorf("invalid image URL scheme: %q (only https is allowed)", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		log.Printf("Rejected image URL with empty host")
		return "", fmt.Errorf("invalid image URL: empty host")
	}

	// Block SSRF: resolve the host and reject any IP that maps to loopback,
	// private (RFC1918), link-local (incl. the 169.254.169.254 metadata IP) or
	// unspecified ranges. Literal-IP hosts are returned as-is by LookupIP and
	// checked the same way. Every resolved IP is checked to defeat multi-record
	// / DNS-rebinding bypasses.
	ips, err := net.LookupIP(host)
	if err != nil {
		log.Printf("Error resolving image host %s: %v", host, err)
		return "", fmt.Errorf("failed to resolve image host: %w", err)
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			log.Printf("Rejected image host %s: resolves to disallowed address", host)
			return "", fmt.Errorf("image host %q resolves to a disallowed address", host)
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make request
	resp, err := client.Get(imageURL)
	if err != nil {
		log.Printf("Error downloading image from host %s: %v", host, err)
		return "", err
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error downloading image: status code %d", resp.StatusCode)
		return "", fmt.Errorf("failed to download image: status code %d", resp.StatusCode)
	}

	// Enforce the 5MB cap before buffering: read at most maxImageSize+1 bytes so
	// an oversized body is never fully buffered.
	limited := io.LimitReader(resp.Body, maxImageSize+1)
	imageData, err := io.ReadAll(limited)
	if err != nil {
		log.Printf("Error reading image data: %v", err)
		return "", err
	}
	if len(imageData) > maxImageSize {
		log.Printf("Image too large: exceeds %d bytes", maxImageSize)
		return "", fmt.Errorf("image too large")
	}

	// Convert to base64
	base64String := base64.StdEncoding.EncodeToString(imageData)
	return fmt.Sprintf("data:%s;base64,%s", resp.Header.Get("Content-Type"), base64String), nil
}
