package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

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

// PlayerProfileStore is the DB surface used by /api/player-profile handlers; *database.Queries implements it.
// PlayerProfileDB on Config overrides DB for those handlers when set (unit tests).
type PlayerProfileStore interface {
	GetPlayerProfileByUserID(ctx context.Context, userID int32) (database.PlayerProfile, error)
	UpsertPlayerProfile(ctx context.Context, arg database.UpsertPlayerProfileParams) (database.PlayerProfile, error)
	UpdatePlayerProfileRow(ctx context.Context, arg database.UpdatePlayerProfileRowParams) (database.PlayerProfile, error)
	DeletePlayerProfileRow(ctx context.Context, id int32) error
	ListPlayerProfileTraits(ctx context.Context, playerProfileID int32) ([]string, error)
	ListPlayerProfileCareerTeams(ctx context.Context, playerProfileID int32) ([]database.PlayerProfileCareerTeam, error)
	DeletePlayerProfileTraitsByProfileID(ctx context.Context, playerProfileID int32) error
	InsertPlayerProfileTrait(ctx context.Context, arg database.InsertPlayerProfileTraitParams) (database.PlayerProfileTrait, error)
}

// InitJWT initializes JWT authentication with the provided secret
func InitJWT(secret string) error {
	return auth.InitJWTAuth(secret)
}

type Config struct {
	DB                 *database.Queries
	DBConn             *sql.DB
	FootballAPIKey     string
	RapidAPIKey        string
	Cache              cache.CacheInterface
	APIFootballBaseURL string
	NewsBaseURL        string // optional; when set, news client uses this (e.g. for tests)
	OpenAIKey          string
	OpenAIBaseURL      string
	AIPromptGenerator  *ai.PromptGenerator
	SystemUserEmail    string // Email for Fucci system user (006 seeded comments); default fucci@system.local

	// Optional test doubles; when set, handlers use them instead of DB for the corresponding reads.
	CardVoteReader  CardVoteReader
	CommentReader   CommentReader
	PlayerProfileDB PlayerProfileStore // nil => use DB for /api/player-profile routes
}

func New(c Config) http.Handler {
	router := chi.NewRouter()

	// Initialize AI prompt generator if OpenAI key is provided
	if c.OpenAIKey != "" {
		c.AIPromptGenerator = ai.NewPromptGenerator(c.OpenAIKey, c.OpenAIBaseURL, c.Cache)
	}

	// Initialize services
	teamsService := NewTeamsService(c.DB)
	teamManagersService := NewTeamManagersService(c.DB)
	leaguesService := NewLeaguesService(c.DB)
	playerProfilesService := &PlayerProfileService{DB: c.DB}
	verificationsService := &VerificationService{DB: c.DB, PlayerProfileSvc: &PlayerProfileService{DB: c.DB}}

	// Health check routes
	router.Get("/health", HandleReadiness)
	router.Get("/health/redis", c.HandleRedisHealth)
	router.Get("/health/cache-stats", c.HandleCacheStats)

	// Auth routes (no authentication required)
	authRouter := chi.NewRouter()
	authRouter.Post("/register", c.handleCreateUser)
	authRouter.Post("/login", c.handleLogin)

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
	playerProfileRouter.Post("/", c.postPlayerProfile)
	playerProfileRouter.Put("/", c.putPlayerProfile)
	playerProfileRouter.Put("/traits", c.putPlayerProfileTraits)
	playerProfileRouter.Delete("/", c.deletePlayerProfile)

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

	debateRouter := chi.NewRouter()
	debateRouter.Post("/", c.createDebate)
	debateRouter.Get("/top", c.getTopDebates)
	debateRouter.Get("/generate", c.generateAIPrompt)
	debateRouter.Post("/generate", c.generateDebate)
	debateRouter.Post("/generate-set", c.generateDebateSet)
	debateRouter.Get("/health", c.checkDebateGenerationHealth)
	debateRouter.Get("/match", c.getDebatesByMatch)
	debateRouter.Get("/{id}", c.getDebate)
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

	// Player Profiles routes
	playerProfilesRouter := chi.NewRouter()
	playerProfilesRouter.Post("/", playerProfilesService.CreatePlayerProfile)
	playerProfilesRouter.Get("/{id}", playerProfilesService.GetPlayerProfile)
	playerProfilesRouter.Put("/{id}", playerProfilesService.UpdatePlayerProfile)
	playerProfilesRouter.Delete("/{id}", playerProfilesService.DeletePlayerProfile)

	// Verifications routes
	verificationsRouter := chi.NewRouter()
	verificationsRouter.Post("/", verificationsService.AddVerification)
	verificationsRouter.Delete("/{id}", verificationsService.RemoveVerification)
	verificationsRouter.Get("/player/{playerId}", verificationsService.ListVerifications)

	commentsRouter := chi.NewRouter()
	commentsRouter.With(auth.RequireAuth).Put("/{commentId}/vote", c.SetCommentVote)
	commentsRouter.With(auth.RequireAuth).Post("/{commentId}/reactions", c.AddCommentReaction)
	commentsRouter.With(auth.RequireAuth).Delete("/{commentId}/reactions", c.RemoveCommentReaction)

	router.Mount("/auth", authRouter)
	router.Mount("/users", userRouter)
	router.Mount("/player-profile", playerProfileRouter)
	router.Mount("/comments", commentsRouter)
	router.Mount("/futbol", futbolRouter)
	router.Mount("/google", googleRouter)
	router.Mount("/news", newsRouter)
	router.Mount("/debates", debateRouter)
	router.Mount("/teams", teamsRouter)
	router.Mount("/team-managers", teamManagersRouter)
	router.Mount("/leagues", leaguesRouter)
	router.Mount("/player-profiles", playerProfilesRouter)
	router.Mount("/verifications", verificationsRouter)

	return router
}

// getUserIDFromContext extracts user ID from request context (set by auth middleware)
func getUserIDFromContext(r *http.Request) uuid.UUID {
	userID, ok := r.Context().Value("user_id").(int32)
	if !ok {
		// Return default UUID if no user ID in context (for backward compatibility)
		return uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}
	// Convert int32 to UUID by creating a zero-padded UUID
	return uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012d", userID))
}
