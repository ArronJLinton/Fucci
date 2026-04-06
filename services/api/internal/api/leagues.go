package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
)

// LeaguesService handles league-related operations
type LeaguesService struct {
	db *database.Queries
}

// NewLeaguesService creates a new leagues service
func NewLeaguesService(db *database.Queries) *LeaguesService {
	return &LeaguesService{db: db}
}

// CreateLeague creates a new league
func (s *LeaguesService) CreateLeague(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Country     string `json:"country"`
		Level       int32  `json:"level"`
		LogoURL     string `json:"logo_url"`
		Website     string `json:"website"`
		Founded     int32  `json:"founded"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	// For now, use a default user ID since getUserIDFromContext returns UUID but DB expects int32
	// In a real implementation, you'd need to fix this type mismatch
	defaultUserID := int32(1) // Using a default user ID for testing

	// Create league
	league, err := s.db.CreateLeague(r.Context(), database.CreateLeagueParams{
		Name:        req.Name,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
		OwnerID:     defaultUserID,
		Country:     sql.NullString{String: req.Country, Valid: req.Country != ""},
		Level:       sql.NullInt32{Int32: req.Level, Valid: req.Level > 0},
		LogoUrl:     sql.NullString{String: req.LogoURL, Valid: req.LogoURL != ""},
		Website:     sql.NullString{String: req.Website, Valid: req.Website != ""},
		Founded:     sql.NullInt32{Int32: req.Founded, Valid: req.Founded > 0},
	})
	if err != nil {
		http.Error(w, "Failed to create league", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(league)
}

// GetLeague retrieves a league by ID
func (s *LeaguesService) GetLeague(w http.ResponseWriter, r *http.Request) {
	leagueID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid league ID format", http.StatusBadRequest)
		return
	}

	league, err := s.db.GetLeague(r.Context(), leagueID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "League not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get league", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(league)
}

// ListLeagues retrieves all leagues with optional filtering
func (s *LeaguesService) ListLeagues(w http.ResponseWriter, r *http.Request) {
	ownerIDStr := r.URL.Query().Get("owner_id")
	countryStr := r.URL.Query().Get("country")
	levelStr := r.URL.Query().Get("level")
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limitStr = "50"
	}
	offsetStr := r.URL.Query().Get("offset")
	if offsetStr == "" {
		offsetStr = "0"
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	var ownerID int32
	if ownerIDStr != "" {
		parsedOwnerID, err := strconv.Atoi(ownerIDStr)
		if err != nil {
			http.Error(w, "Invalid owner_id format", http.StatusBadRequest)
			return
		}
		ownerID = int32(parsedOwnerID)
	}

	var level int32
	if levelStr != "" {
		parsedLevel, err := strconv.Atoi(levelStr)
		if err != nil {
			http.Error(w, "Invalid level format", http.StatusBadRequest)
			return
		}
		level = int32(parsedLevel)
	}

	leagues, err := s.db.ListLeagues(r.Context(), database.ListLeaguesParams{
		Column1: ownerID,
		Column2: countryStr,
		Column3: level,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		http.Error(w, "Failed to list leagues", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(leagues)
}

// UpdateLeague updates a league
func (s *LeaguesService) UpdateLeague(w http.ResponseWriter, r *http.Request) {
	leagueID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid league ID format", http.StatusBadRequest)
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Country     *string `json:"country"`
		Level       *int32  `json:"level"`
		LogoURL     *string `json:"logo_url"`
		Website     *string `json:"website"`
		Founded     *int32  `json:"founded"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get current league
	league, err := s.db.GetLeague(r.Context(), leagueID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "League not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get league", http.StatusInternalServerError)
		return
	}

	// Check permissions
	userID := getUserIDFromContext(r)
	if !s.hasLeaguePermission(r.Context(), userID, leagueID, "update") {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	// Prepare update parameters
	params := database.UpdateLeagueParams{
		ID: leagueID,
	}

	// Update fields if provided
	if req.Name != nil {
		params.Name = *req.Name
	} else {
		params.Name = league.Name
	}

	if req.Description != nil {
		params.Description = sql.NullString{String: *req.Description, Valid: *req.Description != ""}
	} else {
		params.Description = league.Description
	}

	if req.Country != nil {
		params.Country = sql.NullString{String: *req.Country, Valid: *req.Country != ""}
	} else {
		params.Country = league.Country
	}

	if req.Level != nil {
		params.Level = sql.NullInt32{Int32: *req.Level, Valid: *req.Level > 0}
	} else {
		params.Level = league.Level
	}

	if req.LogoURL != nil {
		params.LogoUrl = sql.NullString{String: *req.LogoURL, Valid: *req.LogoURL != ""}
	} else {
		params.LogoUrl = league.LogoUrl
	}

	if req.Website != nil {
		params.Website = sql.NullString{String: *req.Website, Valid: *req.Website != ""}
	} else {
		params.Website = league.Website
	}

	if req.Founded != nil {
		params.Founded = sql.NullInt32{Int32: *req.Founded, Valid: *req.Founded > 0}
	} else {
		params.Founded = league.Founded
	}

	// Update league
	updatedLeague, err := s.db.UpdateLeague(r.Context(), params)
	if err != nil {
		http.Error(w, "Failed to update league", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedLeague)
}

// DeleteLeague deletes a league
func (s *LeaguesService) DeleteLeague(w http.ResponseWriter, r *http.Request) {
	leagueID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid league ID format", http.StatusBadRequest)
		return
	}

	// Check permissions
	userID := getUserIDFromContext(r)
	if !s.hasLeaguePermission(r.Context(), userID, leagueID, "delete") {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	err = s.db.DeleteLeague(r.Context(), leagueID)
	if err != nil {
		http.Error(w, "Failed to delete league", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "League deleted successfully"})
}

// GetLeagueStats retrieves league statistics
func (s *LeaguesService) GetLeagueStats(w http.ResponseWriter, r *http.Request) {
	leagueID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid league ID format", http.StatusBadRequest)
		return
	}

	// Get league
	league, err := s.db.GetLeague(r.Context(), leagueID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "League not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get league", http.StatusInternalServerError)
		return
	}

	// Get league teams
	teams, err := s.db.GetTeamsByLeague(r.Context(), uuid.NullUUID{UUID: leagueID, Valid: true})
	if err != nil {
		http.Error(w, "Failed to get league teams", http.StatusInternalServerError)
		return
	}

	// Get league managers
	managers, err := s.db.GetTeamManagersByLeague(r.Context(), leagueID)
	if err != nil {
		http.Error(w, "Failed to get league managers", http.StatusInternalServerError)
		return
	}

	// Legacy player_profiles-backed league player stats were retired with table cleanup.
	// TODO: re-introduce league player stats from canonical player_profile model.
	players := []interface{}{}

	// Calculate stats
	var totalRating float64
	var totalVerifications int32
	playerCount := int32(len(players))
	teamCount := int32(len(teams))
	managerCount := int32(len(managers))

	for range players {
		// No-op until canonical league-linked player stats are implemented.
	}

	var avgRating float64
	if playerCount > 0 {
		avgRating = totalRating / float64(playerCount)
	}

	stats := map[string]interface{}{
		"league_id":           league.ID,
		"league_name":         league.Name,
		"team_count":          teamCount,
		"manager_count":       managerCount,
		"player_count":        playerCount,
		"average_rating":      avgRating,
		"total_verifications": totalVerifications,
		"teams":               teams,
		"managers":            managers,
		"players":             players,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// hasLeaguePermission checks if user has permission for league operations
func (s *LeaguesService) hasLeaguePermission(ctx context.Context, userID, leagueID uuid.UUID, operation string) bool {
	// For now, return true for all operations since we're using a default user ID
	// In a real implementation, this would check user permissions properly
	return true
}

// Permission check: Is user an admin?
func (svc *LeaguesService) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	// TODO: Implement using DB.GetUser
	return false, nil
}
