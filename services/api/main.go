package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/ArronJLinton/fucci-api/internal/api"
	"github.com/ArronJLinton/fucci-api/internal/cache"
	"github.com/ArronJLinton/fucci-api/internal/config"
	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	_ "github.com/lib/pq"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
)

type Config struct {
	DB *database.Queries
}

var (
	version = "dev"
)

func main() {
	// Initialize the logger
	zlog, _ := zap.NewProduction(
		zap.Fields(
			zap.String("version", version),
		),
	)
	defer func() {
		_ = zlog.Sync()
	}()
	logger := otelzap.New(zlog)

	// Initialize the configuration
	c := config.InitConfig(logger)

	// Initialize JWT authentication
	if err := api.InitJWT(c.JWT_SECRET); err != nil {
		log.Printf("Warning: Failed to initialize JWT auth: %v (auth features may not work)\n", err)
	}

	conn, err := sql.Open("postgres", c.DB_URL)
	if err != nil {
		log.Fatal("Failed to connect to Database - ", err)
	}

	// Initialize Redis cache
	redisCache, err := cache.NewCache(c.REDIS_URL)
	if err != nil {
		log.Fatal("Failed to connect to Redis - ", err)
	}

	router := chi.NewRouter()
	// Tells browsers how this api can be used
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", ";http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"string"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	v1Router := chi.NewRouter()
	dbQueries := database.New(conn)
	apiCfg := api.Config{
		DB:                            dbQueries,
		DBConn:                        conn,
		FootballAPIKey:                c.FOOTBALL_API_KEY,
		RapidAPIKey:                   c.RAPID_API_KEY,
		NewsBaseURL:                   c.NewsBaseURL,
		GoogleOAuthClientID:           c.GOOGLE_OAUTH_CLIENT_ID,
		GoogleOAuthClientSecret:       c.GOOGLE_OAUTH_CLIENT_SECRET,
		GoogleOAuthRedirectURIs:       c.GOOGLE_OAUTH_REDIRECT_URIS,
		GoogleOAuthCallbackURL:        c.GOOGLE_OAUTH_CALLBACK_URL,
		GoogleOAuthAllowDevReturnURLs: c.GOOGLE_OAUTH_ALLOW_DEV_RETURN_URLS,
		CloudinaryCloudName:           c.CLOUDINARY_CLOUD_NAME,
		CloudinaryAPIKey:              c.CLOUDINARY_API_KEY,
		CloudinaryAPISecret:           c.CLOUDINARY_API_SECRET,
		CloudinaryUploadPreset:        c.CLOUDINARY_UPLOAD_PRESET,
		Cache:                         redisCache,
		OpenAIKey:                     c.OPENAI_API_KEY,
		OpenAIBaseURL:                 c.OPENAI_BASE_URL,
		SystemUserEmail:               c.SYSTEM_USER_EMAIL,
	}
	apiRouter := api.New(&apiCfg)
	v1Router.Mount("/api", apiRouter)
	router.Mount("/v1", v1Router)

	// Get port from environment variable with fallback
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Always bind to 0.0.0.0 for both local and production
	bindAddr := "0.0.0.0"
	serverAddr := fmt.Sprintf("%s:%s", bindAddr, port)
	fmt.Printf("Server starting on %s\n", serverAddr)

	server := &http.Server{
		Handler: router,
		Addr:    serverAddr,
	}

	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
