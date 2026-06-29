package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/api"
	"github.com/ArronJLinton/fucci-api/internal/cache"
	"github.com/ArronJLinton/fucci-api/internal/config"
	"github.com/ArronJLinton/fucci-api/internal/database"
	pushpkg "github.com/ArronJLinton/fucci-api/internal/push"
	"github.com/ArronJLinton/fucci-api/internal/scheduler"
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
		NewsAPIKey:                    c.NEWS_API_KEY,
		NewsBaseURL:                   c.NEWS_BASE_URL,
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
		YouTubeAPIKey:                 c.YOUTUBE_API_KEY,
		YouTubeCacheTTLHours:          c.YOUTUBE_CACHE_TTL_HOURS,
		ExpoAccessToken:               c.EXPO_ACCESS_TOKEN,
		Environment:                   c.ENVIRONMENT,
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

	// Daily 04:00 UTC pre-match debate generator (+ news cache warm). Runs as a
	// goroutine in this process; cross-machine deduplication uses Redis SetNX
	// inside the job so multi-machine deploys still only generate once per day.
	// See services/api/internal/api/prewarm.go for the job body, and the
	// PREWARM_LEAGUE_IDS env var to extend or disable.
	schedCtx, schedCancel := context.WithCancel(context.Background())
	var prewarmScheduler *scheduler.Scheduler
	if leagueIDs := api.ParsePrewarmLeagueIDs(c.PREWARM_LEAGUE_IDS); len(leagueIDs) > 0 {
		prewarmJob := api.NewPrewarmJob(&apiCfg, leagueIDs, port)
		prewarmScheduler = scheduler.New(prewarmJob, scheduler.Options{
			DailyAtUTC: time.Date(0, 1, 1, 4, 0, 0, 0, time.UTC),
			RunOnStart: true,
		})
		prewarmScheduler.Start(schedCtx)
		log.Printf("[main] pre-warm scheduler started (leagues=%v)", leagueIDs)
	} else {
		log.Printf("[main] PREWARM_LEAGUE_IDS empty; pre-match debate scheduler disabled")
	}

	pushScanCtx := schedCtx
	pushSlotScanner := &pushpkg.SlotScanner{Cache: redisCache}
	pushIntervalScheduler := scheduler.NewInterval(pushSlotScanner, scheduler.IntervalOptions{
		Every: 15 * time.Minute,
	})
	pushIntervalScheduler.Start(pushScanCtx)
	log.Printf("[main] push slot scanner started (every 15m, Phase 1 no-op)")

	// Trap SIGINT/SIGTERM so we can stop the scheduler and drain HTTP cleanly.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Printf("[main] shutdown signal received; stopping scheduler and HTTP server")
		schedCancel()
		if prewarmScheduler != nil {
			go prewarmScheduler.Stop()
		}
		pushIntervalScheduler.Stop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
