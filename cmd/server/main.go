package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vlkhvnn/inmemcache/pkg/cache"
)

// Command-line flags.
var (
	authEnabled  = flag.Bool("auth", false, "Enable authentication")
	authPassword = flag.String("password", "secret", "Authentication password")
	useTLS       = flag.Bool("tls", false, "Enable TLS")
	certFile     = flag.String("cert", "server.crt", "TLS certificate file")
	keyFile      = flag.String("key", "server.key", "TLS key file")
	tcpAddr      = flag.String("tcp", ":8080", "TCP server address")
	metricsAddr  = flag.String("metrics", ":9090", "Metrics HTTP server address")
	workerCount  = flag.Int("workers", 10, "Number of workers in the pool")
)

// Prometheus metrics.
var (
	reqCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mycache_requests_total",
		Help: "Total number of requests processed",
	}, []string{"command"})
	errorCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mycache_errors_total",
		Help: "Total number of errors encountered",
	}, []string{"command"})
	processingDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mycache_processing_seconds",
		Help:    "Histogram of request processing durations",
		Buckets: prometheus.DefBuckets,
	}, []string{"command"})
)

func init() {
	prometheus.MustRegister(reqCounter)
	prometheus.MustRegister(errorCounter)
	prometheus.MustRegister(processingDuration)
}

// handleConnection processes a single connection. If authentication is enabled,
// it requires an "AUTH <password>" command before any other commands are accepted.
// It records metrics for each command processed.
func handleConnection(conn net.Conn, c *cache.Cache) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	authenticated := !*authEnabled // if auth is not enabled, consider the connection authenticated

	for scanner.Scan() {
		start := time.Now()
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		command := strings.ToUpper(parts[0])

		// Require authentication if enabled.
		if *authEnabled && !authenticated {
			if command != "AUTH" {
				fmt.Fprintln(conn, "ERROR: Authentication required. Please use AUTH <password>")
				errorCounter.WithLabelValues("unauthenticated").Inc()
				continue
			}
			if len(parts) < 2 || parts[1] != *authPassword {
				fmt.Fprintln(conn, "ERROR: Invalid password")
				errorCounter.WithLabelValues("AUTH").Inc()
				return // Close connection on failed auth.
			}
			authenticated = true
			fmt.Fprintln(conn, "OK")
			reqCounter.WithLabelValues("AUTH").Inc()
			processingDuration.WithLabelValues("AUTH").Observe(time.Since(start).Seconds())
			continue
		}

		// Process the command.
		switch command {
		case "SET":
			reqCounter.WithLabelValues("SET").Inc()
			if len(parts) < 3 {
				fmt.Fprintln(conn, "ERROR: SET requires key and value")
				errorCounter.WithLabelValues("SET").Inc()
				continue
			}
			key := parts[1]
			value := strings.Join(parts[2:], " ")
			c.Set(key, value)
			fmt.Fprintln(conn, "OK")
		case "GET":
			reqCounter.WithLabelValues("GET").Inc()
			if len(parts) < 2 {
				fmt.Fprintln(conn, "ERROR: GET requires key")
				errorCounter.WithLabelValues("GET").Inc()
				continue
			}
			key := parts[1]
			value, err := c.Get(key)
			if err != nil {
				fmt.Fprintln(conn, "ERROR: key not found")
				errorCounter.WithLabelValues("GET").Inc()
			} else {
				fmt.Fprintln(conn, value)
			}
		case "DEL":
			reqCounter.WithLabelValues("DEL").Inc()
			if len(parts) < 2 {
				fmt.Fprintln(conn, "ERROR: DEL requires key")
				errorCounter.WithLabelValues("DEL").Inc()
				continue
			}
			key := parts[1]
			c.Delete(key)
			fmt.Fprintln(conn, "OK")
		default:
			fmt.Fprintln(conn, "ERROR: unknown command")
			errorCounter.WithLabelValues("unknown").Inc()
		}
		processingDuration.WithLabelValues(command).Observe(time.Since(start).Seconds())
	}
	if err := scanner.Err(); err != nil {
		log.Printf("connection error: %v", err)
	}
}

// worker continuously reads from the connection channel and processes each connection.
func worker(id int, connChan <-chan net.Conn, c *cache.Cache) {
	for conn := range connChan {
		log.Printf("Worker %d handling connection from %s", id, conn.RemoteAddr())
		handleConnection(conn, c)
	}
}

func main() {
	flag.Parse()

	// Start the metrics HTTP server.
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Printf("Metrics server listening on %s", *metricsAddr)
		if err := http.ListenAndServe(*metricsAddr, nil); err != nil {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()

	// Create an instance of the in-memory cache.
	cacheInstance := cache.NewCache()

	// Set up the TCP listener with optional TLS.
	var ln net.Listener
	var err error
	if *useTLS {
		// Load TLS certificate and key.
		cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("Failed to load TLS certificate and key: %v", err)
		}
		tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
		ln, err = tls.Listen("tcp", *tcpAddr, tlsConfig)
		if err != nil {
			log.Fatalf("Failed to listen with TLS on %s: %v", *tcpAddr, err)
		}
		log.Printf("Server (TLS enabled) is listening on %s", *tcpAddr)
	} else {
		ln, err = net.Listen("tcp", *tcpAddr)
		if err != nil {
			log.Fatalf("Failed to listen on %s: %v", *tcpAddr, err)
		}
		log.Printf("Server is listening on %s", *tcpAddr)
	}

	// Create a connection channel (queue) for the worker pool.
	connChan := make(chan net.Conn, 100)

	// Launch the worker pool.
	for i := 0; i < *workerCount; i++ {
		go worker(i, connChan, cacheInstance)
	}

	// Accept incoming connections and send them to the worker pool.
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		connChan <- conn
	}
}
