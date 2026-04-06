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

// TeamManagersService handles team manager-related operations
type TeamManagersService struct {
	db *database.Queries
}

// NewTeamManagersService creates a new team managers service
func NewTeamManagersService(db *database.Queries) *TeamManagersService {
	return &TeamManagersService{db: db}
}

// CreateTeamManager creates a new team manager
func (s *TeamManagersService) CreateTeamManager(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID     int32  `json:"user_id"`
		LeagueID   string `json:"league_id"`
		TeamID     string `json:"team_id"`
		Title      string `json:"title"`
		Experience int32  `json:"experience"`
		Bio        string `json:"bio"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.LeagueID == "" {
		http.Error(w, "league_id is required", http.StatusBadRequest)
		return
	}

	// Parse UUIDs
	var leagueIDParsed uuid.UUID
	if req.LeagueID != "" {
		var err error
		leagueIDParsed, err = uuid.Parse(req.LeagueID)
		if err != nil {
			http.Error(w, "Invalid league_id format", http.StatusBadRequest)
			return
		}
	}

	var teamID *uuid.UUID
	if req.TeamID != "" {
		parsedTeamID, err := uuid.Parse(req.TeamID)
		if err != nil {
			http.Error(w, "Invalid team_id format", http.StatusBadRequest)
			return
		}
		teamID = &parsedTeamID
	}

	// Check if current user has permission to create managers in this league
	currentUserID := getUserIDFromContext(r)
	if !s.hasManagerPermission(r.Context(), currentUserID, leagueIDParsed, "create") {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	// Check if user exists
	_, err := s.db.GetUser(r.Context(), req.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	// Check if team exists (if provided)
	if teamID != nil {
		_, err := s.db.GetTeam(r.Context(), *teamID)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Team not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to get team", http.StatusInternalServerError)
			return
		}
	}

	// Check if user is already a manager in this league
	existingManagers, err := s.db.GetTeamManagersByLeague(r.Context(), leagueIDParsed)
	if err != nil {
		http.Error(w, "Failed to check existing managers", http.StatusInternalServerError)
		return
	}

	// Convert *uuid.UUID to uuid.NullUUID for teamID
	var teamIDNull uuid.NullUUID
	if teamID != nil {
		teamIDNull = uuid.NullUUID{UUID: *teamID, Valid: true}
	}

	// Check if user is already a manager in this league
	for _, manager := range existingManagers {
		if manager.UserID == req.UserID {
			http.Error(w, "User is already a manager in this league", http.StatusConflict)
			return
		}
	}

	// Create team manager
	manager, err := s.db.CreateTeamManager(r.Context(), database.CreateTeamManagerParams{
		UserID:     req.UserID,
		LeagueID:   leagueIDParsed,
		TeamID:     teamIDNull,
		Title:      sql.NullString{String: req.Title, Valid: req.Title != ""},
		Experience: sql.NullInt32{Int32: req.Experience, Valid: req.Experience > 0},
		Bio:        sql.NullString{String: req.Bio, Valid: req.Bio != ""},
	})
	if err != nil {
		http.Error(w, "Failed to create team manager", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(manager)
}

// GetTeamManager retrieves a team manager by ID
func (s *TeamManagersService) GetTeamManager(w http.ResponseWriter, r *http.Request) {
	managerID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid manager ID format", http.StatusBadRequest)
		return
	}

	manager, err := s.db.GetTeamManager(r.Context(), managerID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Team manager not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get team manager", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manager)
}

// ListTeamManagers retrieves all team managers with optional filtering
func (s *TeamManagersService) ListTeamManagers(w http.ResponseWriter, r *http.Request) {
	leagueIDStr := r.URL.Query().Get("league_id")
	teamIDStr := r.URL.Query().Get("team_id")
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

	var leagueID *uuid.UUID
	if leagueIDStr != "" {
		parsedLeagueID, err := uuid.Parse(leagueIDStr)
		if err != nil {
			http.Error(w, "Invalid league_id format", http.StatusBadRequest)
			return
		}
		leagueID = &parsedLeagueID
	}

	var teamID *uuid.UUID
	if teamIDStr != "" {
		parsedTeamID, err := uuid.Parse(teamIDStr)
		if err != nil {
			http.Error(w, "Invalid team_id format", http.StatusBadRequest)
			return
		}
		teamID = &parsedTeamID
	}

	// In ListTeamManagers, use uuid.Nil for columns that can't be filtered properly
	var leagueIDFilter uuid.UUID = uuid.Nil
	if leagueID != nil {
		leagueIDFilter = *leagueID
	}
	var teamIDFilter uuid.UUID = uuid.Nil
	if teamID != nil {
		teamIDFilter = *teamID
	}
	managers, err := s.db.ListTeamManagers(r.Context(), database.ListTeamManagersParams{
		Column1: leagueIDFilter,
		Column2: teamIDFilter,
		Column3: uuid.Nil, // Don't filter by user for now since DB expects UUID but we have int32
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		http.Error(w, "Failed to list team managers", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(managers)
}

// UpdateTeamManager updates a team manager
func (s *TeamManagersService) UpdateTeamManager(w http.ResponseWriter, r *http.Request) {
	managerID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid manager ID format", http.StatusBadRequest)
		return
	}

	var req struct {
		TeamID     *string `json:"team_id"`
		Title      *string `json:"title"`
		Experience *int32  `json:"experience"`
		Bio        *string `json:"bio"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get current manager
	manager, err := s.db.GetTeamManager(r.Context(), managerID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Team manager not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get team manager", http.StatusInternalServerError)
		return
	}

	// Check permissions
	currentUserID := getUserIDFromContext(r)
	if !s.hasManagerPermission(r.Context(), currentUserID, manager.LeagueID, "update") {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	// Prepare update parameters
	params := database.UpdateTeamManagerParams{
		ID: managerID,
	}

	// Update fields if provided
	if req.TeamID != nil {
		if *req.TeamID == "" {
			params.TeamID = uuid.NullUUID{Valid: false}
		} else {
			parsedTeamID, err := uuid.Parse(*req.TeamID)
			if err != nil {
				http.Error(w, "Invalid team_id format", http.StatusBadRequest)
				return
			}
			params.TeamID = uuid.NullUUID{UUID: parsedTeamID, Valid: true}
		}
	} else {
		params.TeamID = manager.TeamID
	}

	if req.Title != nil {
		params.Title = sql.NullString{String: *req.Title, Valid: *req.Title != ""}
	} else {
		params.Title = manager.Title
	}

	if req.Experience != nil {
		params.Experience = sql.NullInt32{Int32: *req.Experience, Valid: *req.Experience > 0}
	} else {
		params.Experience = manager.Experience
	}

	if req.Bio != nil {
		params.Bio = sql.NullString{String: *req.Bio, Valid: *req.Bio != ""}
	} else {
		params.Bio = manager.Bio
	}

	// Validate team exists if changing
	if req.TeamID != nil && params.TeamID.Valid {
		_, err := s.db.GetTeam(r.Context(), params.TeamID.UUID)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Team not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to get team", http.StatusInternalServerError)
			return
		}
	}

	// Update team manager
	updatedManager, err := s.db.UpdateTeamManager(r.Context(), params)
	if err != nil {
		http.Error(w, "Failed to update team manager", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedManager)
}

// DeleteTeamManager deletes a team manager
func (s *TeamManagersService) DeleteTeamManager(w http.ResponseWriter, r *http.Request) {
	managerID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid manager ID format", http.StatusBadRequest)
		return
	}

	// Get manager to check permissions
	manager, err := s.db.GetTeamManager(r.Context(), managerID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Team manager not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get team manager", http.StatusInternalServerError)
		return
	}

	// Check permissions
	currentUserID := getUserIDFromContext(r)
	if !s.hasManagerPermission(r.Context(), currentUserID, manager.LeagueID, "delete") {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	err = s.db.DeleteTeamManager(r.Context(), managerID)
	if err != nil {
		http.Error(w, "Failed to delete team manager", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Team manager deleted successfully"})
}

// GetManagerStats retrieves manager statistics
func (s *TeamManagersService) GetManagerStats(w http.ResponseWriter, r *http.Request) {
	managerID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid manager ID format", http.StatusBadRequest)
		return
	}

	// Get manager
	manager, err := s.db.GetTeamManager(r.Context(), managerID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Team manager not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get team manager", http.StatusInternalServerError)
		return
	}

	// Get manager's team (if assigned)
	var team *database.Teams
	if manager.TeamID.Valid {
		teamData, err := s.db.GetTeam(r.Context(), manager.TeamID.UUID)
		if err == nil {
			team = &teamData
		}
	}

	// Legacy player_profiles-backed team player stats were retired with table cleanup.
	// TODO: re-introduce manager/team player stats from canonical player_profile model.
	players := []interface{}{}

	// Calculate stats
	var totalRating float64
	var totalVerifications int32
	playerCount := int32(len(players))

	for range players {
		// No-op until canonical team-linked player stats are implemented.
	}

	var avgRating float64
	if playerCount > 0 {
		avgRating = totalRating / float64(playerCount)
	}

	stats := map[string]interface{}{
		"manager_id":          manager.ID,
		"user_id":             manager.UserID,
		"league_id":           manager.LeagueID,
		"team":                team,
		"player_count":        playerCount,
		"average_rating":      avgRating,
		"total_verifications": totalVerifications,
		"players":             players,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// hasManagerPermission checks if user has permission for manager operations
func (s *TeamManagersService) hasManagerPermission(ctx context.Context, userID, leagueID uuid.UUID, operation string) bool {
	// For now, return true for all operations since we're using placeholder user IDs
	// In a real implementation, this would check user permissions properly
	return true
}
