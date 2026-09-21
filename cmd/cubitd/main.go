package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/adapters/out/docker"
	"github.com/ishaf/cubit/internal/adapters/out/storage"
	"github.com/ishaf/cubit/internal/adapters/out/traefik"
	"github.com/ishaf/cubit/internal/core/middleware"
	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/infrastructure/db"
	appModule "github.com/ishaf/cubit/internal/modules/application"
	depModule "github.com/ishaf/cubit/internal/modules/deployment"
	domModule "github.com/ishaf/cubit/internal/modules/domain"
	ghModule "github.com/ishaf/cubit/internal/modules/github"
	nodeModule "github.com/ishaf/cubit/internal/modules/node"
	runtimeModule "github.com/ishaf/cubit/internal/modules/runtime"
	srvModule "github.com/ishaf/cubit/internal/modules/service"
)

func main() {
	port := flag.Int("port", 8000, "Port to listen on")
	dbPath := flag.String("db", "cubit.db", "SQLite database path")
	traefikOut := flag.String("traefik-config", "/etc/traefik/dynamic/cubit.yaml", "Traefik dynamic configuration output path")
	storageDir := flag.String("storage-dir", ".data/storage", "Storage directory for local S3 / Garage")
	bucketName := flag.String("bucket", "cubit-fleet", "Default fleet bucket name")
	webDir := flag.String("web-dir", "", "Directory containing compiled frontend SPA assets (defaults to CUBIT_WEB_DIR or ./web/dist)")
	flag.Parse()

	log.Printf("Starting Cubit Control Plane on port %d...", *port)

	// 1. Initialize SQLite Database
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", *dbPath)
	database, err := db.OpenSQLite(dsn)
	if err != nil {
		log.Fatalf("Fatal: Database initialization failed: %v", err)
	}
	defer database.Close()
	log.Println("SQLite database initialized in WAL mode.")

	// 2. Initialize Repositories (Module Data Access)
	appRepo := appModule.NewRepository(database)
	depRepo := depModule.NewRepository(database)
	nodeRepo := nodeModule.NewRepository(database)
	domRepo := domModule.NewRepository(database)
	servicesRepo := srvModule.NewRepository(database, filepath.Join(*storageDir, "d1"))
	githubRepo := ghModule.NewRepository(database)

	// 3. Initialize Outbound Infrastructure Adapters
	proxyProvider := traefik.NewFileProvider(*traefikOut, "letsencrypt")
	routeSyncer := traefik.NewRouteSyncer(proxyProvider, appRepo, domRepo, nodeRepo)
	storageAdapter, err := storage.NewLocalStorageAdapter(*storageDir, string(storage.DriverGarageLocal))
	if err != nil {
		log.Fatalf("Fatal: Storage initialization failed: %v", err)
	}
	dockerSupervisor := docker.NewCelldSupervisor("ghcr.io/denoland/celld")

	// 4. Initialize Modular Domain Services
	nodeService := nodeModule.NewService(nodeRepo, dockerSupervisor, routeSyncer, fmt.Sprintf("s3://%s", *bucketName))
	appService := appModule.NewService(appRepo, storageAdapter, routeSyncer, *bucketName)
	depService := depModule.NewService(depRepo, appService, storageAdapter, routeSyncer, *bucketName)
	domainService := domModule.NewService(domRepo, appRepo, routeSyncer)
	runtimeService := runtimeModule.NewService(nodeRepo, dockerSupervisor, storageAdapter, fmt.Sprintf("s3://%s", *bucketName), domain.DefaultCelldVersion)
	servicesService := srvModule.NewService(servicesRepo, appRepo, appService, storageAdapter, *bucketName)
	githubService := ghModule.NewService(githubRepo, appRepo, depService)

	// 5. Initialize Modular HTTP Handlers
	nodeHandler := nodeModule.NewHandler(nodeService)
	appHandler := appModule.NewHandler(appService)
	depHandler := depModule.NewHandler(depService)
	domHandler := domModule.NewHandler(domainService)
	runtimeHandler := runtimeModule.NewHandler(runtimeService)
	servicesHandler := srvModule.NewHandler(servicesService)
	githubHandler := ghModule.NewHandler(githubService)

	// 6. Configure Gin Engine and Middlewares
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.SubdomainRouter(appService))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 7. Mount Module Routes under /api/v1
	api := r.Group("/api/v1")
	{
		nodeHandler.RegisterRoutes(api)
		appHandler.RegisterRoutes(api)
		depHandler.RegisterRoutes(api)
		domHandler.RegisterRoutes(api)
		runtimeHandler.RegisterRoutes(api)
		servicesHandler.RegisterRoutes(api)
		githubHandler.RegisterRoutes(api)
	}

	// 8. Static SPA file server
	middleware.ServeSPA(r, *webDir)

	srv := &http.Server{
		Addr:        fmt.Sprintf(":%d", *port),
		Handler:     r,
		ReadTimeout: 15 * time.Second,
		IdleTimeout: 60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	log.Printf("Cubit Control Plane is listening on http://localhost:%d", *port)

	// Graceful shutdown on SIGINT / SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Cubit Control Plane...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Cubit Control Plane cleanly stopped.")
}
