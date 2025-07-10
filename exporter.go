package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config holds all configuration variables for convenience.
type Config struct {
	ApiURL         string
	User           string
	Password       string
	Port           string
	ScrapeInterval time.Duration
}

var (
	// httpClient is a reusable HTTP client.
	httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}

	// blockedIP is the Prometheus metric definition.
	blockedIP = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "fastnetmon_blocked_ip", // A more unique name to avoid conflicts.
			Help: "Represents a currently blocked IP address by FastNetMon.",
		},
		[]string{"ip", "uuid"}, // Adding UUID can be useful.
	)
)

// loadConfig loads configuration from environment variables.
func loadConfig() (*Config, error) {
	// Load .env file. Log a warning if it fails, but don't stop.
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: Could not load .env file. Using environment variables.", err)
	}

	cfg := &Config{
		ApiURL:   os.Getenv("EXPORTER_API_URL"),
		User:     os.Getenv("EXPORTER_USER"),
		Password: os.Getenv("EXPORTER_PASSWORD"),
		Port:     os.Getenv("EXPORTER_PORT"),
	}

	// Check for mandatory environment variables.
	if cfg.ApiURL == "" || cfg.User == "" || cfg.Password == "" {
		return nil, fmt.Errorf("error: missing required environment variables: EXPORTER_API_URL, EXPORTER_USER, EXPORTER_PASSWORD")
	}

	if cfg.Port == "" {
		cfg.Port = ":9898" // Default port.
	}

	// Make the scrape interval configurable.
	intervalStr := os.Getenv("EXPORTER_SCRAPE_INTERVAL_SECONDS")
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval <= 0 {
		interval = 60 // Default interval is 60 seconds.
	}
	cfg.ScrapeInterval = time.Duration(interval) * time.Second

	return cfg, nil
}

// Structs for parsing the JSON response.
type BlockedValue struct {
	UUID string `json:"uuid"`
	IP   string `json:"ip"`
}

type BlackholeResponse struct {
	Success bool           `json:"success"`
	Values  []BlockedValue `json:"values"`
}

// fetchBlockedIPs performs a request to the FastNetMon API.
func fetchBlockedIPs(cfg *Config) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.ApiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.SetBasicAuth(cfg.User, cfg.Password)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	return body, nil
}

// updateMetrics parses the response and updates Prometheus metrics.
func updateMetrics(body []byte) {
	var resp BlackholeResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		log.Printf("Error decoding JSON: %v", err)
		return
	}

	if !resp.Success {
		log.Println("API request was not successful according to response body")
		return
	}

	// Reset all old metrics before updating.
	blockedIP.Reset()
	for _, v := range resp.Values {
		blockedIP.With(prometheus.Labels{"ip": v.IP, "uuid": v.UUID}).Set(1)
	}
	log.Printf("Successfully updated metrics. Found %d blocked IPs.", len(resp.Values))
}

// startScrapingLoop starts the endless loop for API scraping.
func startScrapingLoop(cfg *Config) {
	ticker := time.NewTicker(cfg.ScrapeInterval)
	defer ticker.Stop()

	// Run immediately for the first time without waiting for the ticker.
	for ; ; <-ticker.C {
		log.Println("Scraping FastNetMon API...")
		body, err := fetchBlockedIPs(cfg)
		if err != nil {
			log.Printf("Error during scrape: %v", err)
		} else {
			updateMetrics(body)
		}
	}
}

// healthCheckHandler is the handler for the exporter's own health check.
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// Just respond with OK if the service is up and responding to requests.
	io.WriteString(w, `{"status": "ok"}`)
}

func main() {
	// Load configuration on startup.
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Register the metric with Prometheus.
	prometheus.MustRegister(blockedIP)

	// Start the metric update loop in a separate goroutine.
	go startScrapingLoop(cfg)

	// Register HTTP handlers.
	http.HandleFunc("/health", healthCheckHandler) // <-- NEW ENDPOINT
	http.Handle("/metrics", promhttp.Handler())

	log.Printf("Starting exporter on port %s", cfg.Port)
	log.Printf("Scraping API every %v", cfg.ScrapeInterval)
	log.Println("Health check available at /health")
	if err := http.ListenAndServe(cfg.Port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
