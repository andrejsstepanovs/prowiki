package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/andrejsstepanovs/prowiki/internal/db"
	"github.com/andrejsstepanovs/prowiki/internal/di"
	"github.com/andrejsstepanovs/prowiki/internal/migrate"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	
	targetDir := "."
	if len(os.Args) >= 3 {
		targetDir = os.Args[2]
	}

	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		fmt.Printf("failed to get absolute path: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Println("\nShutting down gracefully...")
		cancel()
	}()

	switch cmd {
	case "init":
		if err := runInit(absDir); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case "ingest":
		if err := runIngest(ctx, absDir); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case "daemon":
		if err := runDaemon(ctx, absDir); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case "server":
		if err := runServer(ctx, absDir); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`ProWiki CLI

Usage:
  prowiki init [path]    - Initialize the database (.prowiki.db)
  prowiki ingest [path]  - Run the ingestion scanner
  prowiki daemon [path]  - Start the background queue worker
  prowiki server [path]  - Start the web dashboard and API`)
}

func runInit(dir string) error {
	dbPath := filepath.Join(dir, ".prowiki.db")
	database, err := db.Open(db.Config{Path: dbPath})
	if err != nil {
		return fmt.Errorf("failed to create db: %w", err)
	}
	defer database.Close()

	fmt.Println("Running migrations...")
	if err := migrate.Up(database); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	fmt.Printf("ProWiki initialized successfully at %s\n", dbPath)
	return nil
}

func runIngest(ctx context.Context, dir string) error {
	c, err := di.NewContainer(ctx, dir)
	if err != nil {
		return err
	}
	defer c.DB.Close()

	fmt.Printf("Starting ingestion in %s...\n", dir)
	if err := c.IngestionService.Run(ctx); err != nil {
		return fmt.Errorf("ingestion failed: %w", err)
	}
	fmt.Println("Ingestion complete.")
	return nil
}

func runDaemon(ctx context.Context, dir string) error {
	c, err := di.NewContainer(ctx, dir)
	if err != nil {
		return err
	}
	defer c.DB.Close()

	fmt.Printf("Starting ProWiki daemon in %s...\n", dir)
	c.Daemon.Start(ctx) // Blocks until ctx is canceled
	return nil
}

func runServer(ctx context.Context, dir string) error {
	c, err := di.NewContainer(ctx, dir)
	if err != nil {
		return err
	}
	defer c.DB.Close()

	fmt.Printf("Starting ProWiki server in %s...\n", dir)
	return c.Server.Start(ctx)
}
