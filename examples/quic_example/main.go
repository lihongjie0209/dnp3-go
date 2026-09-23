package main

import (
	"fmt"
	"time"

	"github.com/lihongjie0209/dnp3-go/pkg/channel"
	"github.com/lihongjie0209/dnp3-go/pkg/dnp3"
)

// Example demonstrating QUIC channel usage with DNP3
func main() {
	fmt.Println("=== QUIC Channel Example ===")

	// Example 1: QUIC Server
	fmt.Println("Example 1: QUIC Server (Outstation)")
	if err := runQUICServer(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}

	fmt.Println()

	// Example 2: QUIC Client
	fmt.Println("Example 2: QUIC Client (Master)")
	if err := runQUICClient(); err != nil {
		fmt.Printf("Client error: %v\n", err)
	}
}

func runQUICServer() error {
	fmt.Println("Starting QUIC server...")

	// Set logging level
	dnp3.SetLogLevel(dnp3.LevelInfo)

	// Create QUIC channel in server mode
	quicChannel, err := channel.NewQUICChannel(channel.QUICChannelConfig{
		Address:        ":20000", // Listen on port 20000
		IsServer:       true,
		ReconnectDelay: 5 * time.Second,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   10 * time.Second,
		TLSConfig:      nil, // Will use self-signed cert
	})
	if err != nil {
		return fmt.Errorf("failed to create QUIC channel: %w", err)
	}
	defer quicChannel.Close()

	fmt.Println("QUIC server listening on :20000")
	fmt.Println("Waiting for connections...")

	// The QUIC channel is now ready to use with DNP3
	// You can create a DNP3 channel using channel.New() from pkg/channel
	// For this example, we'll just demonstrate the QUIC connection is working

	// Keep server running for demo
	time.Sleep(30 * time.Second)

	// Print statistics
	stats := quicChannel.Statistics()
	fmt.Printf("\nQUIC Server Statistics:\n")
	fmt.Printf("  Bytes Sent: %d\n", stats.BytesSent)
	fmt.Printf("  Bytes Received: %d\n", stats.BytesReceived)
	fmt.Printf("  Connects: %d\n", stats.Connects)
	fmt.Printf("  Disconnects: %d\n", stats.Disconnects)
	fmt.Printf("  Read Errors: %d\n", stats.ReadErrors)
	fmt.Printf("  Write Errors: %d\n", stats.WriteErrors)

	return nil
}

func runQUICClient() error {
	fmt.Println("Starting QUIC client...")

	// Set logging level
	dnp3.SetLogLevel(dnp3.LevelInfo)

	// Create QUIC channel in client mode
	quicChannel, err := channel.NewQUICChannel(channel.QUICChannelConfig{
		Address:        "localhost:20000", // Connect to server
		IsServer:       false,
		ReconnectDelay: 5 * time.Second,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   10 * time.Second,
		TLSConfig:      nil, // Will use self-signed cert
	})
	if err != nil {
		return fmt.Errorf("failed to create QUIC channel: %w", err)
	}
	defer quicChannel.Close()

	fmt.Println("QUIC client connecting to localhost:20000")

	// The QUIC channel is now ready to use with DNP3
	// You can create a DNP3 channel using channel.New() from pkg/channel
	// For this example, we'll just demonstrate the QUIC connection is working

	fmt.Println("Connected successfully!")

	// Keep client running for demo
	time.Sleep(30 * time.Second)

	// Print statistics
	stats := quicChannel.Statistics()
	fmt.Printf("\nQUIC Client Statistics:\n")
	fmt.Printf("  Bytes Sent: %d\n", stats.BytesSent)
	fmt.Printf("  Bytes Received: %d\n", stats.BytesReceived)
	fmt.Printf("  Connects: %d\n", stats.Connects)
	fmt.Printf("  Disconnects: %d\n", stats.Disconnects)
	fmt.Printf("  Read Errors: %d\n", stats.ReadErrors)
	fmt.Printf("  Write Errors: %d\n", stats.WriteErrors)

	return nil
}
