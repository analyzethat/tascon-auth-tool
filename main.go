package main

import (
	"context"
	"database/sql"
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
	// Log security configuration
	logSecurityConfig()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var database *sql.DB

	// Only connect if credentials are configured
	if cfg.Username != "" && cfg.Password != "" {
		log.Printf("Connecting to database %s on %s...", cfg.Database, cfg.Server)

		database, err = db.Open(db.Config{
			Server:   cfg.Server,
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		})
		if err != nil {
			log.Printf("Warning: Failed to connect to database: %v", err)
			log.Println("Start the application and configure credentials in Settings")
		} else {
			defer database.Close()
			log.Println("Connected to database successfully")
		}
	} else {
		log.Println("No database credentials configured. Please configure in Settings.")
	}

	// Setup repositories (may be nil if no database connection)
	var userRepo *repository.UserRepository
	var accessRepo *repository.AccessRepository
	var groupRepo *repository.GroupRepository

	if database != nil {
		userRepo = repository.NewUserRepository(database)
		accessRepo = repository.NewAccessRepository(database)
		groupRepo = repository.NewGroupRepository(database)
	}

	// Setup handlers
	h, err := handlers.NewHandler(userRepo, accessRepo, groupRepo, cfg)
	if err != nil {
		log.Fatalf("Failed to setup handlers: %v", err)
	}

	// Setup router
	router := handlers.SetupRoutes(h)

	// Get port from environment (Railway sets PORT) or use random port
	port := os.Getenv("PORT")
	if port == "" {
		port = "0"
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port

	fmt.Printf("Server running at http://localhost:%d\n", actualPort)

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

func logSecurityConfig() {
	if _, hasKey := config.GetMasterKey(); hasKey {
		log.Println("Config encryption: ENABLED (POWERBI_MASTER_KEY set)")
	} else {
		log.Println("Config encryption: DISABLED (set POWERBI_MASTER_KEY for encryption)")
	}

	if _, hasPassword := handlers.GetAdminPassword(); hasPassword {
		log.Println("Authentication: ENABLED (POWERBI_ADMIN_PASSWORD set)")
	} else {
		log.Println("Authentication: DISABLED (set POWERBI_ADMIN_PASSWORD to enable)")
	}
}
