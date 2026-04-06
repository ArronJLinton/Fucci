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

// TeamsService handles team-related operations
type TeamsService struct {
	db *database.Queries
}

// NewTeamsService creates a new teams service
func NewTeamsService(db *database.Queries) *TeamsService {
	return &TeamsService{db: db}
}

// CreateTeam creates a new team
func (s *TeamsService) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		LeagueID    string `json:"league_id"`
		ManagerID   string `json:"manager_id"`
		LogoURL     string `json:"logo_url"`
		City        string `json:"city"`
		Country     string `json:"country"`
		Founded     int32  `json:"founded"`
		Stadium     string `json:"stadium"`
		Capacity    int32  `json:"capacity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.LeagueID == "" {
		http.Error(w, "name and league_id are required", http.StatusBadRequest)
		return
	}

	// Parse UUIDs
	leagueID, err := uuid.Parse(req.LeagueID)
	if err != nil {
		http.Error(w, "Invalid league_id format", http.StatusBadRequest)
		return
	}

	var managerID *uuid.UUID
	if req.ManagerID != "" {
		parsedManagerID, err := uuid.Parse(req.ManagerID)
		if err != nil {
			http.Error(w, "Invalid manager_id format", http.StatusBadRequest)
			return
		}
		managerID = &parsedManagerID
	}

	// Check if user has permission to create teams in this league
	// if !s.hasTeamPermission(r.Context(), userID, leagueID, "create") {
	// 	http.Error(w, "Insufficient permissions", http.StatusForbidden)
	// 	return
	// }

	// Check if league exists
	// league, err := s.db.GetLeague(r.Context(), leagueID)
	// if err != nil {
	// 	if err == sql.ErrNoRows {
	// 		http.Error(w, "League not found", http.StatusNotFound)
	// 		return
	// 	}
	// 	http.Error(w, "Failed to get league", http.StatusInternalServerError)
	// 	return
	// }

	// Check if manager exists (if provided)
	if managerID != nil {
		_, err := s.db.GetTeamManager(r.Context(), *managerID)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Manager not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to get manager", http.StatusInternalServerError)
			return
		}
	}

	// Fix managerID in CreateTeam
	var managerIDNull uuid.NullUUID
	if managerID != nil {
		managerIDNull = uuid.NullUUID{UUID: *managerID, Valid: true}
	}

	// Create team
	team, err := s.db.CreateTeam(r.Context(), database.CreateTeamParams{
		Name:        req.Name,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
		LeagueID:    uuid.NullUUID{UUID: leagueID, Valid: true},
		ManagerID:   managerIDNull,
		LogoUrl:     sql.NullString{String: req.LogoURL, Valid: req.LogoURL != ""},
		City:        sql.NullString{String: req.City, Valid: req.City != ""},
		Country:     req.Country,
		Founded:     sql.NullInt32{Int32: req.Founded, Valid: req.Founded > 0},
		Stadium:     sql.NullString{String: req.Stadium, Valid: req.Stadium != ""},
		Capacity:    sql.NullInt32{Int32: req.Capacity, Valid: req.Capacity > 0},
	})
	if err != nil {
		http.Error(w, "Failed to create team", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(team)
}

// GetTeam retrieves a team by ID
func (s *TeamsService) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid team ID format", http.StatusBadRequest)
		return
	}

	team, err := s.db.GetTeam(r.Context(), teamID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get team", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

// ListTeams retrieves all teams with optional filtering
func (s *TeamsService) ListTeams(w http.ResponseWriter, r *http.Request) {
	leagueIDStr := r.URL.Query().Get("league_id")
	managerIDStr := r.URL.Query().Get("manager_id")
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

	var managerID *uuid.UUID
	if managerIDStr != "" {
		parsedManagerID, err := uuid.Parse(managerIDStr)
		if err != nil {
			http.Error(w, "Invalid manager_id format", http.StatusBadRequest)
			return
		}
		managerID = &parsedManagerID
	}

	var leagueIDFilter uuid.UUID = uuid.Nil
	if leagueID != nil {
		leagueIDFilter = *leagueID
	}
	var managerIDFilter uuid.UUID = uuid.Nil
	if managerID != nil {
		managerIDFilter = *managerID
	}
	teams, err := s.db.ListTeams(r.Context(), database.ListTeamsParams{
		Column1: leagueIDFilter,
		Column2: managerIDFilter,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		http.Error(w, "Failed to list teams", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(teams)
}

// UpdateTeam updates a team
func (s *TeamsService) UpdateTeam(w http.ResponseWriter, r *http.Request) {
	teamID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid team ID format", http.StatusBadRequest)
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		LeagueID    *string `json:"league_id"`
		ManagerID   *string `json:"manager_id"`
		LogoURL     *string `json:"logo_url"`
		City        *string `json:"city"`
		Country     *string `json:"country"`
		Founded     *int32  `json:"founded"`
		Stadium     *string `json:"stadium"`
		Capacity    *int32  `json:"capacity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get current team
	team, err := s.db.GetTeam(r.Context(), teamID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get team", http.StatusInternalServerError)
		return
	}

	// Check permissions
	// if !s.hasTeamPermission(r.Context(), userID, team.LeagueID, "update") {
	// 	http.Error(w, "Insufficient permissions", http.StatusForbidden)
	// 	return
	// }

	// Prepare update parameters
	params := database.UpdateTeamParams{
		ID: teamID,
	}

	// Update fields if provided
	if req.Name != nil {
		params.Name = *req.Name
	} else {
		params.Name = team.Name
	}

	if req.Description != nil {
		params.Description = sql.NullString{String: *req.Description, Valid: *req.Description != ""}
	} else {
		params.Description = team.Description
	}

	if req.LeagueID != nil {
		parsedLeagueID, err := uuid.Parse(*req.LeagueID)
		if err != nil {
			http.Error(w, "Invalid league_id format", http.StatusBadRequest)
			return
		}
		params.LeagueID = uuid.NullUUID{UUID: parsedLeagueID, Valid: true}
	} else {
		params.LeagueID = team.LeagueID
	}

	if req.ManagerID != nil {
		if *req.ManagerID == "" {
			params.ManagerID = uuid.NullUUID{Valid: false}
		} else {
			parsedManagerID, err := uuid.Parse(*req.ManagerID)
			if err != nil {
				http.Error(w, "Invalid manager_id format", http.StatusBadRequest)
				return
			}
			params.ManagerID = uuid.NullUUID{UUID: parsedManagerID, Valid: true}
		}
	} else {
		params.ManagerID = team.ManagerID
	}

	if req.LogoURL != nil {
		params.LogoUrl = sql.NullString{String: *req.LogoURL, Valid: *req.LogoURL != ""}
	} else {
		params.LogoUrl = team.LogoUrl
	}

	if req.City != nil {
		params.City = sql.NullString{String: *req.City, Valid: *req.City != ""}
	} else {
		params.City = team.City
	}

	if req.Country != nil {
		params.Country = *req.Country
	} else {
		params.Country = team.Country
	}

	if req.Founded != nil {
		params.Founded = sql.NullInt32{Int32: *req.Founded, Valid: *req.Founded > 0}
	} else {
		params.Founded = team.Founded
	}

	if req.Stadium != nil {
		params.Stadium = sql.NullString{String: *req.Stadium, Valid: *req.Stadium != ""}
	} else {
		params.Stadium = team.Stadium
	}

	if req.Capacity != nil {
		params.Capacity = sql.NullInt32{Int32: *req.Capacity, Valid: *req.Capacity > 0}
	} else {
		params.Capacity = team.Capacity
	}

	// Validate league exists if changing
	if req.LeagueID != nil {
		_, err := s.db.GetLeague(r.Context(), params.LeagueID.UUID)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "League not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to get league", http.StatusInternalServerError)
			return
		}
	}

	// Validate manager exists if changing
	if req.ManagerID != nil && params.ManagerID.UUID != uuid.Nil {
		_, err := s.db.GetTeamManager(r.Context(), params.ManagerID.UUID)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Manager not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to get manager", http.StatusInternalServerError)
			return
		}
	}

	// Update team
	updatedTeam, err := s.db.UpdateTeam(r.Context(), params)
	if err != nil {
		http.Error(w, "Failed to update team", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedTeam)
}

// DeleteTeam deletes a team
func (s *TeamsService) DeleteTeam(w http.ResponseWriter, r *http.Request) {
	teamID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid team ID format", http.StatusBadRequest)
		return
	}

	// Check permissions
	// if !s.hasTeamPermission(r.Context(), userID, team.LeagueID, "delete") {
	// 	http.Error(w, "Insufficient permissions", http.StatusForbidden)
	// 	return
	// }

	err = s.db.DeleteTeam(r.Context(), teamID)
	if err != nil {
		http.Error(w, "Failed to delete team", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Team deleted successfully"})
}

// GetTeamStats retrieves team statistics
func (s *TeamsService) GetTeamStats(w http.ResponseWriter, r *http.Request) {
	teamID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid team ID format", http.StatusBadRequest)
		return
	}

	// Get team
	team, err := s.db.GetTeam(r.Context(), teamID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get team", http.StatusInternalServerError)
		return
	}

	// Legacy player_profiles-backed team player stats were retired with table cleanup.
	// TODO: re-introduce team-scoped player stats from canonical player_profile model.
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
		"team_id":             team.ID,
		"team_name":           team.Name,
		"player_count":        playerCount,
		"average_rating":      avgRating,
		"total_verifications": totalVerifications,
		"players":             players,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Simplify hasTeamPermission function
func (s *TeamsService) hasTeamPermission(ctx context.Context, userID, leagueID uuid.UUID, operation string) bool {
	// For now, return true for all operations since we're using placeholder user IDs
	// In a real implementation, this would check user permissions properly
	return true
}
