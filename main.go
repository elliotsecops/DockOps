package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver"
	"github.com/spf13/viper"
)

type ImageInfo struct {
	Registry string
	Repo     string
	Tag      string
}

type Config struct {
	GCRAccessToken string        `mapstructure:"gcr_access_token"`
	RateLimit      time.Duration `mapstructure:"rate_limit"`
	MaxRetries     int           `mapstructure:"max_retries"`
	RetryDelay     time.Duration `mapstructure:"retry_delay"`
}

var config Config

func main() {
	if err := loadConfig(); err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	rateLimiter := time.NewTicker(config.RateLimit)
	defer rateLimiter.Stop()

	client := &http.Client{}
	cache := make(map[string][]string)

	localImages, err := getLocalImages()
	if err != nil {
		log.Fatalf("Error getting local images: %v", err)
	}

	var wg sync.WaitGroup
	results := make(chan string, len(localImages))

	for _, image := range localImages {
		wg.Add(1)
		go func(img ImageInfo) {
			defer wg.Done()
			<-rateLimiter.C
			if update, err := checkForUpdates(client, img, cache); err != nil {
				log.Printf("Error checking for updates for %s: %v", img.Repo, err)
			} else if update != "" {
				results <- update
			}
		}(image)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		fmt.Println(result)
	}
}

func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return fmt.Errorf("config file not found, please create config.yaml: %w", err)
		}
		return fmt.Errorf("error reading config file: %w", err)
	}

	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Set defaults if not specified
	if config.RateLimit == 0 {
		config.RateLimit = time.Second / 10
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}

	return nil
}

func checkForUpdates(client *http.Client, image ImageInfo, cache map[string][]string) (string, error) {
	remoteTags, err := getRemoteTags(client, image.Registry, image.Repo, "gcloud", cache)
	if err != nil {
		return "", fmt.Errorf("error getting remote tags: %w", err)
	}

	currentVer, err := semver.NewVersion(image.Tag)
	if err != nil {
		return "", fmt.Errorf("error parsing current version: %w", err)
	}

	var latestVer *semver.Version
	for _, tag := range remoteTags {
		if v, err := semver.NewVersion(tag); err == nil {
			if latestVer == nil || v.GreaterThan(latestVer) {
				latestVer = v
			}
		}
	}

	if latestVer != nil && latestVer.GreaterThan(currentVer) {
		return fmt.Sprintf("Update available for %s: %s -> %s", image.Repo, currentVer, latestVer), nil
	}

	return "", nil
}

func getLocalImages() ([]ImageInfo, error) {
	cmd := exec.Command("docker", "image", "ls", "--format", "{{.Repository}}:{{.Tag}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error executing docker command: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var images []ImageInfo

	for _, line := range lines {
		img, err := parseImageName(line)
		if err != nil {
			log.Printf("Error parsing image name %s: %v", line, err)
			continue
		}
		images = append(images, img)
	}

	return images, nil
}

func getRemoteTags(client *http.Client, registry, repo, authMethod string, cache map[string][]string) ([]string, error) {
	cacheKey := registry + "/" + repo
	if cachedTags, ok := cache[cacheKey]; ok {
		return cachedTags, nil
	}

	var resp *http.Response
	var err error
	retryDelay := config.RetryDelay

	for i := 0; i < config.MaxRetries; i++ {
		url := fmt.Sprintf("%s/repositories/%s/tags/?page_size=1000", registry, repo)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var req *http.Request
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating HTTP request: %w", err)
		}

		if authMethod == "gcloud" {
			if config.GCRAccessToken == "" {
				return nil, fmt.Errorf("GCR_ACCESS_TOKEN not set in config")
			}
			req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(config.GCRAccessToken))
		}

		resp, err = client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		if i < config.MaxRetries-1 {
			log.Printf("Retry %d: Error fetching tags, retrying in %s: %v", i+1, retryDelay, err)
			time.Sleep(retryDelay)
			retryDelay *= 2
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("max retries exceeded: error fetching tags: %w", err)
		}
		return nil, fmt.Errorf("max retries exceeded: unexpected status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	var result struct {
		Tags []string `json:"tags"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding JSON response: %w", err)
	}

	cache[cacheKey] = result.Tags
	return result.Tags, nil
}

func parseImageName(name string) (ImageInfo, error) {
	parts := strings.Split(name, "/")
	if len(parts) < 2 {
		return ImageInfo{}, fmt.Errorf("invalid image name format")
	}

	registry := parts[0]
	repo := strings.Join(parts[1:len(parts)-1], "/")
	tagParts := strings.Split(parts[len(parts)-1], ":")
	if len(tagParts) != 2 {
		return ImageInfo{}, fmt.Errorf("invalid tag format")
	}

	repo += "/" + tagParts[0]
	tag := tagParts[1]

	return ImageInfo{
		Registry: registry,
		Repo:     repo,
		Tag:      tag,
	}, nil
}
