package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

// set defaults
const (
	defaultUpdateInterval = "1h"
	defaultUpdateIPv4URL  = "https://www.cloudflare.com/ips-v4"
	defaultServerAddr     = "localhost:8080"
	validTimeUnits        = "'ns', 'us' (or 'µs'), 'ms', 's', 'm', 'h'"
)

// config will hold our configuration
type config struct {
	updateInterval time.Duration // how often we will update ip networks
	updateIPv4URL  string        // source url for updating ip networks
}

// getEnvVars gets config settings from env vars
func getEnvVars(c *config) *config {
	updateIntervalEnv := os.Getenv("UPDATE_INTERVAL")
	if updateIntervalEnv != "" {
		interval, err := time.ParseDuration(updateIntervalEnv)
		if err != nil {
			log.Fatalln("UPDATE_INTERVAL is not a valid. Valid time units are:",
				validTimeUnits)
		}
		c.updateInterval = interval
	}

	updateIPv4URLEnv := os.Getenv("UPDATE_IPV4_URL")
	if updateIPv4URLEnv != "" {
		url, err := url.ParseRequestURI(updateIPv4URLEnv)
		if err != nil {
			log.Fatalln("UPDATE_IPV4_URL is not a valid URL")
		}
		c.updateIPv4URL = url.String()
	}

	return c
}

// updateIPNets reads ip networks from url and writes them to the map
func updateIPNets(url string, ipnets *shmap) {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Unable to get IP networks from URL: %v\n", err)
		return
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	respBytesSlice := bytes.Split(bytes.TrimSpace(respBytes), []byte("\n"))
	// we will clear map before each update
	ipnets.Lock()
	ipnets.m = map[string]struct{}{}
	for _, value := range respBytesSlice {
		_, ipnet, err := net.ParseCIDR(string(value))
		if err != nil {
			log.Printf("Failed to parse CIDR: %v\n", err)
		}
		ipnets.m[ipnet.String()] = struct{}{}
	}
	ipnets.Unlock()
}

// server holds router and ip networks map
type server struct {
	router *http.ServeMux
	ipnets *shmap
}

// shmap holds concurrently safe shared ip networks map
type shmap struct {
	sync.RWMutex
	m map[string]struct{}
}

// getIPNets handler will write a list of ip networks from map
func (s *server) getIPNets(w http.ResponseWriter, r *http.Request) {
	var ipnets []string
	s.ipnets.RLock()
	for ipnet := range s.ipnets.m {
		ipnets = append(ipnets, ipnet)
	}
	s.ipnets.RUnlock()
	bs, err := json.Marshal(ipnets)
	if err != nil {
		log.Printf("Failed to marshal ipnets into JSON: %v\n", err)
	}
	w.Write(bs)
}

// checkIPNet handler will check if ip networks map contains client's ip
func (s *server) checkIPNet(w http.ResponseWriter, r *http.Request) {
	type realip struct {
		RealIP string `json:"real_ip,omitempty"`
	}
	realIP := net.ParseIP(r.Header.Get("X-Real-IP"))
	if realIP == nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		w.Write([]byte("X-Real-IP header is not provided or malformed"))
		return
	}

	match := false
	s.ipnets.RLock()
	for key := range s.ipnets.m {
		_, ipnet, _ := net.ParseCIDR(key)
		if ipnet.Contains(realIP) {
			match = true
			break
		}
	}
	s.ipnets.RUnlock()

	resp := realip{
		RealIP: realIP.String(),
	}

	bs, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal ipnets into JSON: %v\n", err)
	}

	if match {
		w.WriteHeader(http.StatusServiceUnavailable) // 503
	} else {
		w.WriteHeader(http.StatusOK) // 200
	}

	w.Write(bs)
}

func main() {
	// init config with default values
	interval, _ := time.ParseDuration(defaultUpdateInterval)
	url, _ := url.ParseRequestURI(defaultUpdateIPv4URL)

	c := &config{
		updateInterval: interval,
		updateIPv4URL:  url.String(),
	}
	c = getEnvVars(c)

	// init ip networks map
	ipnets := &shmap{}
	updateIPNets(c.updateIPv4URL, ipnets)

	// run IPv4 IP networks updater in a separate goroutine
	go func(c *config) {
		cron := time.Tick(c.updateInterval)
		for range cron {
			updateIPNets(c.updateIPv4URL, ipnets)
		}
	}(c)

	// init server instance
	s := &server{
		router: http.NewServeMux(),
		ipnets: ipnets,
	}

	// init server routes
	s.router.HandleFunc("/", s.getIPNets)
	s.router.HandleFunc("/check", s.checkIPNet)

	// start server
	log.Println("Update interval is set to:", c.updateInterval)
	log.Println("Update IPv4 URL is set to:", c.updateIPv4URL)
	log.Println("Starting server on:", defaultServerAddr)
	log.Fatal(http.ListenAndServe(defaultServerAddr, s.router))
}
