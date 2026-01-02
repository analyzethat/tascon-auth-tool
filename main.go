package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"powerbi-access-tool/config"
	"powerbi-access-tool/db"
	"powerbi-access-tool/handlers"
	"powerbi-access-tool/repository"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Connecting to database %s on %s...", cfg.Database, cfg.Server)
	log.Println("Using Azure CLI authentication (run 'az login' first if not logged in)")

	// Connect to database (triggers MFA popup)
	database, err := db.Open(db.Config{
		Server:   cfg.Server,
		Database: cfg.Database,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()
	log.Println("Connected to database successfully")

	// Setup repositories
	userRepo := repository.NewUserRepository(database)
	accessRepo := repository.NewAccessRepository(database)
	groupRepo := repository.NewGroupRepository(database)

	// Setup handlers
	h, err := handlers.NewHandler(userRepo, accessRepo, groupRepo, cfg)
	if err != nil {
		log.Fatalf("Failed to setup handlers: %v", err)
	}

	// Setup router
	router := handlers.SetupRoutes(h)

	// Listen on random port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	fmt.Printf("Server running at http://localhost:%d\n", port)

	// Setup server
	server := &http.Server{Handler: router}

	// Start server in goroutine
	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down...")
	if err := server.Shutdown(context.Background()); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}
