package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	http_adapter "github.com/ishaf/cubit/internal/adapters/in/http"
	"github.com/ishaf/cubit/internal/adapters/out/db"
	"github.com/ishaf/cubit/internal/adapters/out/docker"
	"github.com/ishaf/cubit/internal/adapters/out/storage"
	"github.com/ishaf/cubit/internal/adapters/out/traefik"
	"github.com/ishaf/cubit/internal/usecase"
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

	// 2. Initialize Repositories (Outbound Adapters)
	nodeRepo := db.NewNodeRepo(database)
	appRepo := db.NewAppRepo(database)
	depRepo := db.NewDeploymentRepo(database)
	domRepo := db.NewDomainRepo(database)

	// 3. Initialize Outbound Infrastructure Adapters
	proxyAdapter := traefik.NewFileProvider(*traefikOut, "letsencrypt")
	storageAdapter, err := storage.NewLocalStorageAdapter(*storageDir, string(storage.DriverGarageLocal))
	if err != nil {
		log.Fatalf("Fatal: Storage initialization failed: %v", err)
	}
	dockerSupervisor := docker.NewCelldSupervisor("ghcr.io/denoland/celld")

	// 4. Initialize Use Cases (Application Layer)
	nodeUsecase := usecase.NewNodeUsecase(nodeRepo, dockerSupervisor, fmt.Sprintf("s3://%s", *bucketName))
	appUsecase := usecase.NewAppUsecase(appRepo, depRepo, nodeRepo, domRepo, storageAdapter, proxyAdapter, *bucketName)
	domainUsecase := usecase.NewDomainUsecase(domRepo, appRepo, appUsecase)
	runtimeUsecase := usecase.NewRuntimeUsecase(nodeRepo, dockerSupervisor, storageAdapter, fmt.Sprintf("s3://%s", *bucketName), "0.2.0")

	// 5. Initialize Inbound HTTP Adapter
	apiHandler := http_adapter.NewAPIHandler(nodeUsecase, appUsecase, domainUsecase, runtimeUsecase, depRepo)
	sseStreamer := http_adapter.NewSSELogStreamer(depRepo)

	// 6. Configure Chi Router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// CORS for frontend web dashboard
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Subdomain routing middleware: routes requests with *.localhost or *.cubit.local to deployed worker
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			host := req.Host
			if h, _, err := net.SplitHostPort(host); err == nil {
				host = h
			}

			var subdomain string
			if strings.HasSuffix(host, ".localhost") {
				subdomain = strings.TrimSuffix(host, ".localhost")
			} else if strings.HasSuffix(host, ".cubit.local") {
				subdomain = strings.TrimSuffix(host, ".cubit.local")
			}

			// If request is directed to a subdomain application (e.g. hello-world-api.localhost)
			if subdomain != "" && subdomain != "api" && subdomain != "dashboard" && subdomain != "localhost" {
				app, err := appUsecase.GetApplicationBySubdomain(req.Context(), subdomain)
				if err == nil && app != nil {
					if app.ActiveDeploymentID == "" {
						http.Error(w, fmt.Sprintf("Application %q has no active deployment", app.Name), http.StatusServiceUnavailable)
						return
					}

					reqBody, _ := io.ReadAll(req.Body)
					headers := make(map[string]string)
					for k, v := range req.Header {
						if len(v) > 0 {
							headers[k] = v[0]
						}
					}

					status, respHeaders, respBody, err := appUsecase.InvokeApplication(req.Context(), app.ID, req.Method, req.URL.RequestURI(), headers, reqBody)
					if err != nil {
						http.Error(w, fmt.Sprintf("Worker invocation error: %v", err), http.StatusInternalServerError)
						return
					}

					for k, v := range respHeaders {
						w.Header().Set(k, v)
					}
					w.WriteHeader(status)
					_, _ = w.Write(respBody)
					return
				}
			}

			next.ServeHTTP(w, req)
		})
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Mount Generated OpenAPI Routes under /api/v1
	r.Route("/api/v1", func(sub chi.Router) {
		http_adapter.HandlerFromMux(apiHandler, sub)
		sub.Get("/deployments/{id}/logs/stream", sseStreamer.HandleStream)
	})

	// Static SPA file server
	staticDir := *webDir
	if staticDir == "" {
		staticDir = os.Getenv("CUBIT_WEB_DIR")
	}
	if staticDir == "" {
		if _, err := os.Stat("./web/dist"); err == nil {
			staticDir = "./web/dist"
		}
	}
	if staticDir != "" {
		absDir, err := filepath.Abs(staticDir)
		if err == nil {
			if _, err := os.Stat(absDir); err == nil {
				r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
					if strings.HasPrefix(req.URL.Path, "/api") || req.URL.Path == "/health" {
						http.NotFound(w, req)
						return
					}
					cleanPath := filepath.Clean(req.URL.Path)
					fpath := filepath.Join(absDir, cleanPath)

					info, err := os.Stat(fpath)
					if os.IsNotExist(err) || (err == nil && info.IsDir()) {
						fpath = filepath.Join(absDir, "index.html")
					}

					data, err := os.ReadFile(fpath)
					if err != nil {
						http.NotFound(w, req)
						return
					}

					w.Header().Set("Content-Type", getMimeType(fpath))
					w.Header().Set("Content-Length", strconv.Itoa(len(data)))
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write(data)
				})
				log.Printf("Serving web dashboard from %s", absDir)
			}
		}
	}

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

func getMimeType(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".ico":
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}

