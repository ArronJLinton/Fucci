package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ArronJLinton/fucci-api/internal/database"
)

// MePlayerProfileResponse is the DTO for GET/POST/PUT /api/me/player-profile (007 spec).
type MePlayerProfileResponse struct {
	ID          int32                    `json:"id"`
	Age         *int32                   `json:"age,omitempty"`
	Country     string                   `json:"country"`
	Club        *string                  `json:"club,omitempty"`
	IsFreeAgent bool                     `json:"is_free_agent"`
	Position    string                   `json:"position"`
	PhotoURL    *string                  `json:"photo_url,omitempty"`
	Traits      []string                 `json:"traits"`
	CareerTeams []MePlayerProfileCareerTeamDTO `json:"career_teams"`
}

// MePlayerProfileCareerTeamDTO is one career team in the profile response.
type MePlayerProfileCareerTeamDTO struct {
	ID        int32   `json:"id"`
	TeamName  string  `json:"team_name"`
	StartYear int32   `json:"start_year"`
	EndYear   *int32  `json:"end_year,omitempty"`
}

// MePlayerProfileInput is the body for POST/PUT /api/me/player-profile.
type MePlayerProfileInput struct {
	Age         *int32  `json:"age"`
	Country     string  `json:"country"`
	Club        *string `json:"club"`
	IsFreeAgent *bool   `json:"is_free_agent"`
	Position    string  `json:"position"`
}

func (c *Config) getMePlayerProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int32)
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	ctx := r.Context()

	profile, err := c.DB.GetMePlayerProfileByUserID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Printf("[me_player_profile] GetMePlayerProfileByUserID error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile")
		return
	}

	traits, _ := c.DB.ListMePlayerProfileTraits(ctx, profile.ID)
	careerRows, _ := c.DB.ListMePlayerProfileCareerTeams(ctx, profile.ID)
	careerTeams := make([]MePlayerProfileCareerTeamDTO, 0, len(careerRows))
	for _, row := range careerRows {
		var endYear *int32
		if row.EndYear.Valid {
			endYear = &row.EndYear.Int32
		}
		careerTeams = append(careerTeams, MePlayerProfileCareerTeamDTO{
			ID:        row.ID,
			TeamName:  row.TeamName,
			StartYear: row.StartYear,
			EndYear:   endYear,
		})
	}

	resp := meProfileToResponse(profile, traits, careerTeams)
	respondWithJSON(w, http.StatusOK, resp)
}

func (c *Config) postMePlayerProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int32)
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	ctx := r.Context()

	var req MePlayerProfileInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Country == "" || req.Position == "" {
		respondWithError(w, http.StatusBadRequest, "country and position are required")
		return
	}
	if req.Position != "GK" && req.Position != "DEF" && req.Position != "MID" && req.Position != "FWD" {
		respondWithError(w, http.StatusBadRequest, "position must be GK, DEF, MID, or FWD")
		return
	}
	if req.Age != nil && (*req.Age < 13 || *req.Age > 60) {
		respondWithError(w, http.StatusBadRequest, "age must be between 13 and 60")
		return
	}

	// Upsert: if profile exists, update; else create.
	existing, err := c.DB.GetMePlayerProfileByUserID(ctx, userID)
	if err == nil {
		// Update existing
		age := sql.NullInt32{}
		if req.Age != nil {
			age.Int32 = *req.Age
			age.Valid = true
		}
		club := sql.NullString{}
		if req.Club != nil {
			club.String = *req.Club
			club.Valid = true
		}
		isFreeAgent := false
		if req.IsFreeAgent != nil {
			isFreeAgent = *req.IsFreeAgent
		}
		_, err = c.DB.UpdateMePlayerProfile(ctx, database.UpdateMePlayerProfileParams{
			ID:          existing.ID,
			Age:         age,
			CountryCode: req.Country,
			ClubName:    club,
			IsFreeAgent: isFreeAgent,
			Position:    req.Position,
			PhotoUrl:    existing.PhotoUrl,
		})
		if err != nil {
			log.Printf("[me_player_profile] UpdateMePlayerProfile error: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to update profile")
			return
		}
		profile, _ := c.DB.GetMePlayerProfileByUserID(ctx, userID)
		traits, _ := c.DB.ListMePlayerProfileTraits(ctx, profile.ID)
		careerRows, _ := c.DB.ListMePlayerProfileCareerTeams(ctx, profile.ID)
		careerTeams := make([]MePlayerProfileCareerTeamDTO, 0, len(careerRows))
		for _, row := range careerRows {
			var endYear *int32
			if row.EndYear.Valid {
				endYear = &row.EndYear.Int32
			}
			careerTeams = append(careerTeams, MePlayerProfileCareerTeamDTO{ID: row.ID, TeamName: row.TeamName, StartYear: row.StartYear, EndYear: endYear})
		}
		respondWithJSON(w, http.StatusOK, meProfileToResponse(profile, traits, careerTeams))
		return
	}
	if err != sql.ErrNoRows {
		log.Printf("[me_player_profile] GetMePlayerProfileByUserID error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile")
		return
	}

	// Create new
	age := sql.NullInt32{}
	if req.Age != nil {
		age.Int32 = *req.Age
		age.Valid = true
	}
	club := sql.NullString{}
	if req.Club != nil {
		club.String = *req.Club
		club.Valid = true
	}
	isFreeAgent := false
	if req.IsFreeAgent != nil {
		isFreeAgent = *req.IsFreeAgent
	}
	profile, err := c.DB.CreateMePlayerProfile(ctx, database.CreateMePlayerProfileParams{
		UserID:      userID,
		Age:         age,
		CountryCode: req.Country,
		ClubName:    club,
		IsFreeAgent: isFreeAgent,
		Position:    req.Position,
	})
	if err != nil {
		log.Printf("[me_player_profile] CreateMePlayerProfile error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to create profile")
		return
	}
	resp := meProfileToResponse(profile, nil, nil)
	respondWithJSON(w, http.StatusOK, resp)
}

func (c *Config) putMePlayerProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int32)
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	ctx := r.Context()

	profile, err := c.DB.GetMePlayerProfileByUserID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Profile not found")
			return
		}
		log.Printf("[me_player_profile] GetMePlayerProfileByUserID error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile")
		return
	}

	var req MePlayerProfileInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Country == "" || req.Position == "" {
		respondWithError(w, http.StatusBadRequest, "country and position are required")
		return
	}
	if req.Position != "GK" && req.Position != "DEF" && req.Position != "MID" && req.Position != "FWD" {
		respondWithError(w, http.StatusBadRequest, "position must be GK, DEF, MID, or FWD")
		return
	}
	if req.Age != nil && (*req.Age < 13 || *req.Age > 60) {
		respondWithError(w, http.StatusBadRequest, "age must be between 13 and 60")
		return
	}

	age := sql.NullInt32{}
	if req.Age != nil {
		age.Int32 = *req.Age
		age.Valid = true
	}
	club := sql.NullString{}
	if req.Club != nil {
		club.String = *req.Club
		club.Valid = true
	}
	isFreeAgent := false
	if req.IsFreeAgent != nil {
		isFreeAgent = *req.IsFreeAgent
	}
	updated, err := c.DB.UpdateMePlayerProfile(ctx, database.UpdateMePlayerProfileParams{
		ID:          profile.ID,
		Age:         age,
		CountryCode: req.Country,
		ClubName:    club,
		IsFreeAgent: isFreeAgent,
		Position:    req.Position,
		PhotoUrl:    profile.PhotoUrl,
	})
	if err != nil {
		log.Printf("[me_player_profile] UpdateMePlayerProfile error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}
	traits, _ := c.DB.ListMePlayerProfileTraits(ctx, updated.ID)
	careerRows, _ := c.DB.ListMePlayerProfileCareerTeams(ctx, updated.ID)
	careerTeams := make([]MePlayerProfileCareerTeamDTO, 0, len(careerRows))
	for _, row := range careerRows {
		var endYear *int32
		if row.EndYear.Valid {
			endYear = &row.EndYear.Int32
		}
		careerTeams = append(careerTeams, MePlayerProfileCareerTeamDTO{ID: row.ID, TeamName: row.TeamName, StartYear: row.StartYear, EndYear: endYear})
	}
	respondWithJSON(w, http.StatusOK, meProfileToResponse(updated, traits, careerTeams))
}

func (c *Config) deleteMePlayerProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int32)
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	ctx := r.Context()

	profile, err := c.DB.GetMePlayerProfileByUserID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Printf("[me_player_profile] GetMePlayerProfileByUserID error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile")
		return
	}
	if err := c.DB.DeleteMePlayerProfile(ctx, profile.ID); err != nil {
		log.Printf("[me_player_profile] DeleteMePlayerProfile error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to delete profile")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func meProfileToResponse(p database.MePlayerProfile, traits []string, careerTeams []MePlayerProfileCareerTeamDTO) MePlayerProfileResponse {
	var age *int32
	if p.Age.Valid {
		age = &p.Age.Int32
	}
	var club *string
	if p.ClubName.Valid {
		club = &p.ClubName.String
	}
	var photoURL *string
	if p.PhotoUrl.Valid {
		photoURL = &p.PhotoUrl.String
	}
	if traits == nil {
		traits = []string{}
	}
	if careerTeams == nil {
		careerTeams = []MePlayerProfileCareerTeamDTO{}
	}
	return MePlayerProfileResponse{
		ID:          p.ID,
		Age:         age,
		Country:     p.CountryCode,
		Club:        club,
		IsFreeAgent: p.IsFreeAgent,
		Position:    p.Position,
		PhotoURL:    photoURL,
		Traits:      traits,
		CareerTeams: careerTeams,
	}
}
