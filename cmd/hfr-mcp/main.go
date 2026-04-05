package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/XaaT/hfr-mcp/internal/config"
	"github.com/XaaT/hfr-mcp/internal/hfr"
	hfrmcp "github.com/XaaT/hfr-mcp/internal/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	cfg := config.Load()

	if cfg.Login == "" || cfg.Passwd == "" {
		fmt.Fprintln(os.Stderr, "credentials required: set login/passwd in hfr.conf or ~/.config/hfr/config, or HFR_LOGIN/HFR_PASSWD env vars")
		os.Exit(1)
	}

	// Create HFR client (login will happen lazily on first tool call)
	client := hfr.NewClient()

	// Lazy login: only login once, on first tool call
	var loginOnce sync.Once
	var loginErr error
	lazyLogin := func() error {
		loginOnce.Do(func() {
			fmt.Fprintln(os.Stderr, "HFR: logging in as", cfg.Login)
			loginErr = client.Login(cfg.Login, cfg.Passwd)
			if loginErr != nil {
				fmt.Fprintln(os.Stderr, "HFR: login failed:", loginErr)
			} else {
				fmt.Fprintln(os.Stderr, "HFR: logged in as", cfg.Login)
			}
		})
		return loginErr
	}

	// Create MCP server
	srv := mcp.NewServer(
		&mcp.Implementation{
			Name:    "hfr-mcp",
			Version: "0.1.0",
		},
		nil,
	)

	// Register tools with lazy login
	hfrmcp.RegisterTools(srv, client, lazyLogin)

	// Run over stdio
	fmt.Fprintln(os.Stderr, "HFR MCP server starting (stdio)")
	if err := srv.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}
