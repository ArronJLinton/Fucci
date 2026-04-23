package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/ArronJLinton/fucci-api/internal/ai"
	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/ArronJLinton/fucci-api/internal/cache"
	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
)

// CardVoteReader is used for unit tests; when set, setCardVote uses it for GetUser and GetDebateCard.
// Production leaves this nil (handler falls back to DB).
type CardVoteReader interface {
	GetUser(ctx context.Context, id int32) (database.Users, error)
	GetDebateCard(ctx context.Context, id int32) (database.DebateCards, error)
}

// CommentReader is used for unit tests; when set, comment handlers use it for GetDebate, GetComments, GetComment.
// Production leaves this nil (handler falls back to DB).
type CommentReader interface {
	GetDebate(ctx context.Context, id int32) (database.Debates, error)
	GetComments(ctx context.Context, debateID sql.NullInt32) ([]database.GetCommentsRow, error)
	GetComment(ctx context.Context, id int32) (database.GetCommentRow, error)
}

// DebatesFeedStore is the DB surface for GET /debates/public-feed and GET /debates/feed.
// *database.Queries implements it; DebatesFeedDB on Config overrides for unit tests.
type DebatesFeedStore interface {
	ListDebatesPublicFeed(ctx context.Context, limit int32) ([]database.ListDebatesPublicFeedRow, error)
	ListDebatesFeedNewForUser(ctx context.Context, arg database.ListDebatesFeedNewForUserParams) ([]database.ListDebatesFeedNewForUserRow, error)
	ListDebatesFeedVotedForUser(ctx context.Context, arg database.ListDebatesFeedVotedForUserParams) ([]database.ListDebatesFeedVotedForUserRow, error)
}

// PlayerProfileStore is the DB surface used by /api/player-profile handlers; *database.Queries implements it.
// PlayerProfileDB on Config overrides DB for those handlers when set (unit tests).
type PlayerProfileStore interface {
	GetPlayerProfileByUserID(ctx context.Context, userID int32) (database.PlayerProfile, error)
	UpsertPlayerProfile(ctx context.Context, arg database.UpsertPlayerProfileParams) (database.PlayerProfile, error)
	UpdatePlayerProfileRow(ctx context.Context, arg database.UpdatePlayerProfileRowParams) (database.PlayerProfile, error)
	DeletePlayerProfileRow(ctx context.Context, id int32) error
	ListComparePlayerCatalog(ctx context.Context, arg database.ListComparePlayerCatalogParams) ([]database.ListComparePlayerCatalogRow, error)
	ListPlayerProfileTraits(ctx context.Context, playerProfileID int32) ([]string, error)
	ListPlayerProfileCareerTeams(ctx context.Context, playerProfileID int32) ([]database.PlayerProfileCareerTeam, error)
	DeletePlayerProfileTraitsByProfileID(ctx context.Context, playerProfileID int32) error
	InsertPlayerProfileTrait(ctx context.Context, arg database.InsertPlayerProfileTraitParams) (database.PlayerProfileTrait, error)
}

type GoogleVerifier interface {
	ExchangeCodeForIDToken(ctx context.Context, code, redirectURI string) (string, error)
	VerifyIDToken(ctx context.Context, token string) (auth.GoogleIDTokenClaims, error)
}

// InitJWT initializes JWT authentication with the provided secret
func InitJWT(secret string) error {
	return auth.InitJWTAuth(secret)
}

type Config struct {
	DB             *database.Queries
	DBConn         *sql.DB
	FootballAPIKey string
	// RapidAPIKey is RAPID_API_KEY: used for Google News (RapidAPI host/headers in google.go), Snapchat stories, and as fallback for the news client if NewsAPIKey is empty.
	RapidAPIKey string
	// NewsAPIKey is NEWS_API_KEY: X-API-Key for the Open Web Ninja realtime news HTTP client (see newsXAPIKey).
	NewsAPIKey              string
	GoogleOAuthClientID     string
	GoogleOAuthClientSecret string
	GoogleOAuthRedirectURIs string // comma-separated list of allowed callback URIs
	// GoogleOAuthCallbackURL is the full URL registered with Google for GET /auth/google/callback (server-side code exchange).
	GoogleOAuthCallbackURL string
	// GoogleOAuthAllowDevReturnURLs enables exp:// and http dev return URLs for GET /auth/google/start (must stay false in production).
	GoogleOAuthAllowDevReturnURLs bool
	CloudinaryCloudName           string
	CloudinaryAPIKey              string
	CloudinaryAPISecret           string
	CloudinaryUploadPreset        string
	Cache                         cache.CacheInterface
	APIFootballBaseURL            string
	NewsBaseURL                   string // optional; when set, news client uses this (e.g. for tests)
	OpenAIKey                     string
	OpenAIBaseURL                 string
	AIPromptGenerator             *ai.PromptGenerator
	SystemUserEmail               string // Email for Fucci system user (006 seeded comments); default fucci@system.local
	GoogleVerifier                GoogleVerifier

	// lazyGoogleVerifier is the default *auth.GoogleOAuthVerifier when GoogleVerifier is nil (production).
	// Initialized once via googleVerifierOnce to avoid new http.Client allocations per request.
	lazyGoogleVerifier GoogleVerifier
	googleVerifierOnce sync.Once

	// Optional test doubles; when set, handlers use them instead of DB for the corresponding reads.
	CardVoteReader  CardVoteReader
	CommentReader   CommentReader
	DebatesFeedDB   DebatesFeedStore   // nil => use DB for debate feed GETs
	PlayerProfileDB PlayerProfileStore // nil => use DB for /api/player-profile routes

	// ProfileUpdateDB optional fake for PUT /users/profile persistence; nil => DBConn + sqlc (production).
	ProfileUpdateDB ProfileUpdatePersistence
}

// newsXAPIKey is the key passed to the Open Web Ninja news HTTP client. When NewsAPIKey is empty, falls back to RapidAPIKey.
func (a *Config) newsXAPIKey() string {
	if a.NewsAPIKey != "" {
		return a.NewsAPIKey
	}
	return a.RapidAPIKey
}

func New(c *Config) http.Handler {
	if c == nil {
		panic("api.New: nil Config")
	}
	router := chi.NewRouter()

	// Do not silently populate OAuth redirect URI allowlists here.
	// Redirect URIs must be explicitly configured by the operator so auth flows fail closed when unset.

	// Initialize AI prompt generator if OpenAI key is provided
	if c.OpenAIKey != "" {
		c.AIPromptGenerator = ai.NewPromptGenerator(c.OpenAIKey, c.OpenAIBaseURL, c.Cache)
	}

	// Initialize services
	teamsService := NewTeamsService(c.DB)
	teamManagersService := NewTeamManagersService(c.DB)
	leaguesService := NewLeaguesService(c.DB)

	// Health check routes
	router.Get("/health", HandleReadiness)
	router.Get("/health/redis", c.HandleRedisHealth)
	router.Get("/health/cache-stats", c.HandleCacheStats)

	// Auth routes (no authentication required)
	authRouter := chi.NewRouter()
	authRouter.Post("/register", c.handleCreateUser)
	authRouter.Post("/login", c.handleLogin)
	authRouter.Post("/google", c.handleGoogleAuth)
	authRouter.Get("/google/start", c.handleGoogleOAuthStart)
	authRouter.Get("/google/callback", c.handleGoogleOAuthCallback)
	authRouter.Post("/google/exchange", c.handleGoogleOAuthExchange)

	// User routes (authentication required)
	userRouter := chi.NewRouter()
	userRouter.Use(auth.RequireAuth)
	userRouter.Get("/profile", c.handleGetProfile)
	userRouter.Put("/profile", c.handleUpdateProfile)
	userRouter.Get("/me/following", c.handleGetFollowing)

	// Temp route for listing all users
	userRouter.Get("/all", c.handleListAllUsers)

	// 007: signed-in user's player profile (GET/POST/PUT/DELETE /api/player-profile, traits at /traits)
	playerProfileRouter := chi.NewRouter()
	playerProfileRouter.Use(auth.RequireAuth)
	playerProfileRouter.Get("/", c.getPlayerProfile)
	playerProfileRouter.Get("/catalog", c.getPlayerProfileCatalog)
	playerProfileRouter.Post("/", c.postPlayerProfile)
	playerProfileRouter.Put("/", c.putPlayerProfile)
	playerProfileRouter.Put("/traits", c.putPlayerProfileTraits)
	playerProfileRouter.Delete("/", c.deletePlayerProfile)

	uploadRouter := chi.NewRouter()
	uploadRouter.Use(auth.RequireAuth)
	uploadRouter.Post("/cloudinary/signature", c.postCloudinarySignature)

	futbolRouter := chi.NewRouter()
	futbolRouter.Get("/matches", c.getMatches)
	futbolRouter.Get("/lineup", c.getMatchLineup)
	futbolRouter.Get("/leagues", c.getLeagues)
	futbolRouter.Get("/team_standings", c.getLeagueStandingsByTeamId)
	futbolRouter.Get("/league_standings", c.getLeagueStandingsByLeagueId)

	googleRouter := chi.NewRouter()
	googleRouter.Get("/search", c.search)

	newsRouter := chi.NewRouter()
	// Register more specific route first to avoid route conflicts
	newsRouter.Get("/football/match", c.getMatchNews)
	newsRouter.Get("/football", c.getFootballNews)

	snapchatRouter := chi.NewRouter()
	snapchatRouter.Get("/stories", c.getSnapchatUserStories)

	debateRouter := chi.NewRouter()
	debateRouter.Post("/", c.createDebate)
	debateRouter.Get("/public-feed", c.getDebatesPublicFeed)
	debateRouter.With(auth.RequireAuth).Get("/feed", c.getDebatesFeed)
	debateRouter.Get("/top", c.getTopDebates)
	debateRouter.Get("/generate", c.generateAIPrompt)
	debateRouter.Post("/generate", c.generateDebate)
	debateRouter.Post("/generate-set", c.generateDebateSet)
	debateRouter.Get("/health", c.checkDebateGenerationHealth)
	debateRouter.Get("/match", c.getDebatesByMatch)
	debateRouter.With(auth.OptionalAuth).Get("/{id}", c.getDebate)
	debateRouter.With(auth.RequireAuth).Put("/{debateId}/cards/{cardId}/vote", c.setCardVote)
	debateRouter.Post("/cards", c.createDebateCard)
	// Legacy POST /debates/votes and POST /debates/comments removed: they were unauthenticated and used hardcoded user_id. Use PUT /debates/{id}/cards/{cardId}/vote and POST /debates/{id}/comments (auth required) instead.
	debateRouter.Get("/{debateId}/comments", c.ListDebateComments)
	debateRouter.With(auth.RequireAuth).Post("/{debateId}/comments", c.CreateDebateComment)
	// Admin routes for soft delete management
	debateRouter.Delete("/{id}/hard", c.hardDeleteDebate) // Permanent deletion
	debateRouter.Post("/{id}/restore", c.restoreDebate)   // Restore soft-deleted debate

	// Teams routes
	teamsRouter := chi.NewRouter()
	teamsRouter.Post("/", teamsService.CreateTeam)
	teamsRouter.Get("/", teamsService.ListTeams)
	teamsRouter.Get("/{id}", teamsService.GetTeam)
	teamsRouter.Put("/{id}", teamsService.UpdateTeam)
	teamsRouter.Delete("/{id}", teamsService.DeleteTeam)
	teamsRouter.Get("/{id}/stats", teamsService.GetTeamStats)

	// Team Managers routes
	teamManagersRouter := chi.NewRouter()
	teamManagersRouter.Post("/", teamManagersService.CreateTeamManager)
	teamManagersRouter.Get("/", teamManagersService.ListTeamManagers)
	teamManagersRouter.Get("/{id}", teamManagersService.GetTeamManager)
	teamManagersRouter.Put("/{id}", teamManagersService.UpdateTeamManager)
	teamManagersRouter.Delete("/{id}", teamManagersService.DeleteTeamManager)
	teamManagersRouter.Get("/{id}/stats", teamManagersService.GetManagerStats)

	// Leagues routes
	leaguesRouter := chi.NewRouter()
	leaguesRouter.Post("/", leaguesService.CreateLeague)
	leaguesRouter.Get("/", leaguesService.ListLeagues)
	leaguesRouter.Get("/{id}", leaguesService.GetLeague)
	leaguesRouter.Put("/{id}", leaguesService.UpdateLeague)
	leaguesRouter.Delete("/{id}", leaguesService.DeleteLeague)
	leaguesRouter.Get("/{id}/stats", leaguesService.GetLeagueStats)

	commentsRouter := chi.NewRouter()
	commentsRouter.With(auth.RequireAuth).Put("/{commentId}/vote", c.SetCommentVote)
	commentsRouter.With(auth.RequireAuth).Post("/{commentId}/reactions", c.AddCommentReaction)
	commentsRouter.With(auth.RequireAuth).Delete("/{commentId}/reactions", c.RemoveCommentReaction)

	router.Mount("/auth", authRouter)
	router.Mount("/users", userRouter)
	router.Mount("/player-profile", playerProfileRouter)
	router.Mount("/upload", uploadRouter)
	router.Mount("/comments", commentsRouter)
	router.Mount("/futbol", futbolRouter)
	router.Mount("/google", googleRouter)
	router.Mount("/news", newsRouter)
	router.Mount("/snapchat", snapchatRouter)
	router.Mount("/debates", debateRouter)
	router.Mount("/teams", teamsRouter)
	router.Mount("/team-managers", teamManagersRouter)
	router.Mount("/leagues", leaguesRouter)
	// Legacy /player-profiles and /verifications routes intentionally not mounted.
	// Canonical profile surface is /player-profile (singular) + /player-profile/traits.

	return router
}

func (c *Config) googleAllowedRedirectURIs() []string {
	raw := strings.Split(c.GoogleOAuthRedirectURIs, ",")
	allowed := make([]string, 0, len(raw)+1)
	seen := map[string]struct{}{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		allowed = append(allowed, s)
	}
	for _, v := range raw {
		add(v)
	}
	add(c.GoogleOAuthCallbackURL)
	return allowed
}

func (c *Config) googleVerifier() GoogleVerifier {
	if c.GoogleVerifier != nil {
		return c.GoogleVerifier
	}
	c.googleVerifierOnce.Do(func() {
		c.lazyGoogleVerifier = auth.NewGoogleOAuthVerifier(
			c.GoogleOAuthClientID,
			c.GoogleOAuthClientSecret,
			c.googleAllowedRedirectURIs(),
		)
	})
	return c.lazyGoogleVerifier
}

// debatesFeedStore returns the querier for debate feed handlers (test mock or DB).
func (c *Config) debatesFeedStore() DebatesFeedStore {
	if c.DebatesFeedDB != nil {
		return c.DebatesFeedDB
	}
	if c.DB == nil {
		return nil
	}
	return c.DB
}

// getUserIDFromContext extracts user ID from request context (set by auth middleware)
func getUserIDFromContext(r *http.Request) uuid.UUID {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		// Return default UUID if no user ID in context (for backward compatibility)
		return uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}
	// Convert int32 to UUID by creating a zero-padded UUID
	return uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012d", userID))
}
