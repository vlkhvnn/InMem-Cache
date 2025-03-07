package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/vlkhvnn/inmemcache/pkg/cache"
)

func handleConnection(conn net.Conn, c *cache.Cache) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		command := strings.ToUpper(parts[0])
		switch command {
		case "SET":
			if len(parts) < 3 {
				fmt.Fprintln(conn, "ERROR: SET requires key and value")
				continue
			}
			key := parts[1]
			value := strings.Join(parts[2:], " ")
			c.Set(key, value)
			fmt.Fprintln(conn, "OK")
		case "GET":
			if len(parts) < 2 {
				fmt.Fprintln(conn, "ERROR: GET requires key")
				continue
			}
			key := parts[1]
			value, err := c.Get(key)
			if err != nil {
				fmt.Fprintln(conn, "ERROR: key not found")
			} else {
				fmt.Fprintln(conn, value)
			}
		case "DEL":
			if len(parts) < 2 {
				fmt.Fprintln(conn, "ERROR: DEL requires key")
				continue
			}
			key := parts[1]
			c.Delete(key)
			fmt.Fprintln(conn, "OK")
		default:
			fmt.Fprintln(conn, "ERROR: unknown command")
		}
	}
}

func main() {
	addr := ":8080"
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}
	log.Printf("Server listening on %s", addr)

	// Create a new cache instance.
	cacheInstance := cache.NewCache()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go handleConnection(conn, cacheInstance)
	}
}
