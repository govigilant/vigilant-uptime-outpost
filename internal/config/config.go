package config

import (
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	VigilantURL           string
	IP                    string
	Port                  int
	Hostname              string
	Country               string
	Latitude              float64
	Longitude             float64
	OutpostSecret         string
	InactivityTimeoutMins int
}

func Load() *Config {
	vigilantURL := os.Getenv("VIGILANT_URL")
	outpostSecret := os.Getenv("OUTPOST_SECRET")
	hostname := getHostname()
	port := getPort(hostname)
	ip := getPublicIP()
	inactivityTimeoutMins := getInactivityTimeoutMins()
	country := strings.TrimSpace(os.Getenv("COUNTRY"))
	latitude := getLatitude()
	longitude := getLongitude()

	if ip == "" {
		log.Printf("IP address could not be determined, exiting")
		os.Exit(1)
	}

	log.Printf("Configuration: IP=%s, Port=%d, Hostname=%s, VigilantURL=%s, Country=%s, Latitude=%f, Longitude=%f, InactivityTimeout=%dmins",
		ip, port, hostname, vigilantURL, country, latitude, longitude, inactivityTimeoutMins)

	return &Config{
		VigilantURL:           vigilantURL,
		IP:                    ip,
		Port:                  port,
		Hostname:              hostname,
		Country:               country,
		Latitude:              latitude,
		Longitude:             longitude,
		OutpostSecret:         outpostSecret,
		InactivityTimeoutMins: inactivityTimeoutMins,
	}
}

func getHostname() string {
	containerName := getDockerContainerName()
	if containerName != "" {
		storeHostname(containerName)
		return containerName
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("failed to get system hostname: %v", err)
		hostname = "unknown"
	}
	storeHostname(hostname)
	return hostname
}

func getDockerContainerName() string {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		hostname, err := os.Hostname()
		if err == nil && hostname != "" {
			return hostname
		}
	}

	data, err := os.ReadFile("/proc/self/cgroup")
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.Contains(line, "docker") {
				parts := strings.Split(line, "/")
				if len(parts) > 0 {
					containerID := parts[len(parts)-1]
					if containerID != "" && containerID != "docker" {
						if len(containerID) > 12 {
							return containerID[:12]
						}
						return containerID
					}
				}
			}
		}
	}

	return ""
}

func storeHostname(hostname string) {
	dataDir := "/var/lib/uptime-outpost"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		dataDir = ".outpost-data"
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Printf("failed to create data directory: %v", err)
			return
		}
	}

	hostnameFile := filepath.Join(dataDir, "hostname")
	if err := os.WriteFile(hostnameFile, []byte(hostname), 0644); err != nil {
		log.Printf("failed to write hostname file: %v", err)
	}
}

func getPort(hostname string) int {
	if p := os.Getenv("PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			return parsed
		}
	}

	rand.Seed(time.Now().UnixNano())
	port := 1000 + rand.Intn(9001)
	return port
}

func getInactivityTimeoutMins() int {
	if t := os.Getenv("INACTIVITY_TIMEOUT_MINS"); t != "" {
		if parsed, err := strconv.Atoi(t); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 60 // Default to 60 minutes (1 hour)
}

func getLatitude() float64 {
	if lat := strings.TrimSpace(os.Getenv("LATITUDE")); lat != "" {
		if parsed, err := strconv.ParseFloat(lat, 64); err == nil {
			if parsed >= -90 && parsed <= 90 {
				return parsed
			}
			log.Printf("invalid latitude value %f, must be between -90 and 90", parsed)
		}
	}
	return 0
}

func getLongitude() float64 {
	if lon := strings.TrimSpace(os.Getenv("LONGITUDE")); lon != "" {
		if parsed, err := strconv.ParseFloat(lon, 64); err == nil {
			if parsed >= -180 && parsed <= 180 {
				return parsed
			}
			log.Printf("invalid longitude value %f, must be between -180 and 180", parsed)
		}
	}
	return 0
}

func getPublicIP() string {
	if ip := os.Getenv("IP"); ip != "" {
		return ip
	}

	ipv4 := fetchPublicIP("https://api.ipify.org")
	if ipv4 != "" && !strings.Contains(ipv4, ":") {
		return ipv4
	}
	ipv6 := fetchPublicIP("https://api64.ipify.org")
	if ipv6 != "" {
		return ipv6
	}

	log.Printf("failed to get public IP")
	return ""
}

func fetchPublicIP(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("failed to fetch IP from %s: %v", url, err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("unexpected status code from %s: %d", url, resp.StatusCode)
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read response from %s: %v", url, err)
		return ""
	}

	ip := strings.TrimSpace(string(body))
	log.Printf("fetched public IP: %s", ip)
	return ip
}
