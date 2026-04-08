package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"
)

func main() {
	// CLI flags
	webPort := flag.Int("web-port", 8080, "Web UI server port")
	tcpHost := flag.String("tcp-host", "127.0.0.1", "TCP game server host")
	tcpPort := flag.Int("tcp-port", 8010, "TCP game server port")
	flag.Parse()

	// Set log level
	log.SetLevel(log.DebugLevel)

	// Create server
	server := NewWebServer(*webPort, *tcpHost, *tcpPort)

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Info("Shutting down...")
		server.Stop()
	}()

	// Start server
	log.Info("Starting web test server", "web-port", *webPort, "tcp-host", *tcpHost, "tcp-port", *tcpPort)
	if err := server.Start(); err != nil {
		log.Error("Server error", "err", err)
		os.Exit(1)
	}
}