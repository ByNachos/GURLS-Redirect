package handler

import (
	"GURLS-Redirect/internal/grpc/client"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ua-parser/uap-go/uaparser"
	"go.uber.org/zap"
	httpSwagger "github.com/swaggo/http-swagger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RedirectHandler handles link redirects using gRPC backend client
func RedirectHandler(log *zap.Logger, backendClient *client.BackendClient, uaParser *uaparser.Parser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		alias := strings.TrimPrefix(r.URL.Path, "/")
		if alias == "" {
			http.NotFound(w, r)
			return
		}

		// Get link info from backend via gRPC
		linkStats, err := backendClient.GetLink(r.Context(), alias)
		if err != nil {
			if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
				http.NotFound(w, r)
				return
			}
			log.Error("failed to get link from backend", zap.Error(err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Check if link has expired
		if linkStats.ExpiresAt != nil && time.Now().After(linkStats.ExpiresAt.AsTime()) {
			log.Warn("attempted to access expired link", zap.String("alias", alias))
			http.Error(w, "Link has expired", http.StatusNotFound)
			return
		}

		// Redirect immediately for maximum speed
		http.Redirect(w, r, linkStats.OriginalUrl, http.StatusFound)

		// Record analytics asynchronously after redirect
		go func() {
			client := uaParser.Parse(r.UserAgent())
			deviceType := "Other"

			if client.UserAgent.Family == "Spider" {
				deviceType = "Bot"
			} else {
				switch client.Device.Family {
				case "iPhone", "Generic Smartphone", "Pixel", "Android":
					deviceType = "Mobile"
				case "iPad", "Generic Tablet":
					deviceType = "Tablet"
				default:
					if client.Os.Family != "Other" && client.Device.Family != "Other" {
						deviceType = "Desktop"
					}
				}
			}

			if err := backendClient.RecordClick(context.Background(), alias, deviceType); err != nil {
				log.Error("failed to record click", zap.Error(err), zap.String("alias", alias))
			}
		}()
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleSwaggerRedirect redirects /api/v1 to /api/v1/
func handleSwaggerRedirect(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1" {
		http.Redirect(w, r, "/api/v1/", http.StatusMovedPermanently)
		return
	}
	http.NotFound(w, r)
}

// handleSwaggerSpec serves the swagger.yaml file
func handleSwaggerSpec(log *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		// Load swagger.yaml content
		content, err := loadSwaggerSpec(log)
		if err != nil {
			log.Error("failed to load swagger spec", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(content))
	}
}

// loadSwaggerSpec loads the swagger.yaml content from file
func loadSwaggerSpec(log *zap.Logger) (string, error) {
	file, err := os.Open("api/swagger.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to open swagger.yaml: %w", err)
	}
	defer file.Close()
	
	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read swagger.yaml: %w", err)
	}
	
	return string(content), nil
}

// NewServer creates HTTP server with gRPC backend client
func NewServer(addr string, log *zap.Logger, backendClient *client.BackendClient, regexesPath string, readTimeout, writeTimeout, idleTimeout time.Duration) (*http.Server, error) {
	// Check if regexes file exists
	if _, err := os.Stat(regexesPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("regexes.yaml not found at path: %s. Please download it from https://github.com/ua-parser/uap-core/blob/master/regexes.yaml and place it in the assets/ directory", regexesPath)
	}

	// Create UA parser from file
	uaParser, err := uaparser.New(regexesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create ua-parser from file: %w", err)
	}

	mux := http.NewServeMux()
	
	// Swagger UI routes
	mux.HandleFunc("/api/v1", handleSwaggerRedirect)
	mux.Handle("/api/v1/", httpSwagger.Handler(
		httpSwagger.URL("/swagger.yaml"),
	))
	mux.HandleFunc("/swagger.yaml", handleSwaggerSpec(log))
	
	// Health check endpoints
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", healthHandler)
	
	// Default redirect handler (should be last to catch all other paths)
	mux.HandleFunc("/", RedirectHandler(log, backendClient, uaParser))

	return &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}, nil
}