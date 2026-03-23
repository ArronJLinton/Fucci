package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/ArronJLinton/fucci-api/internal/database"
)

func (c *Config) mePlayerProfileDB() MePlayerProfileStore {
	if c.MePlayerProfileDB != nil {
		return c.MePlayerProfileDB
	}
	return c.DB
}

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

// normalizeCountryCode validates ISO 3166-1 alpha-2 for VARCHAR(2): exactly two ASCII A–Z letters (case-insensitive input is uppercased).
func normalizeCountryCode(country string) (string, bool) {
	s := strings.ToUpper(strings.TrimSpace(country))
	if len(s) != 2 {
		return "", false
	}
	for i := 0; i < 2; i++ {
		c := s[i]
		if c < 'A' || c > 'Z' {
			return "", false
		}
	}
	return s, true
}

func (c *Config) getMePlayerProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int32)
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	ctx := r.Context()

	profile, err := c.mePlayerProfileDB().GetMePlayerProfileByUserID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Profile not found")
			return
		}
		log.Printf("[me_player_profile] GetMePlayerProfileByUserID error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile")
		return
	}

	traits, _ := c.mePlayerProfileDB().ListMePlayerProfileTraits(ctx, profile.ID)
	careerRows, _ := c.mePlayerProfileDB().ListMePlayerProfileCareerTeams(ctx, profile.ID)
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
	countryCode, ok := normalizeCountryCode(req.Country)
	if !ok {
		respondWithError(w, http.StatusBadRequest, "country must be a 2-letter ISO 3166-1 alpha-2 code (A-Z)")
		return
	}
	req.Country = countryCode
	if req.Position != "GK" && req.Position != "DEF" && req.Position != "MID" && req.Position != "FWD" {
		respondWithError(w, http.StatusBadRequest, "position must be GK, DEF, MID, or FWD")
		return
	}
	if req.Age != nil && (*req.Age < 13 || *req.Age > 60) {
		respondWithError(w, http.StatusBadRequest, "age must be between 13 and 60")
		return
	}

	// Upsert: if profile exists, update; else create.
	existing, err := c.mePlayerProfileDB().GetMePlayerProfileByUserID(ctx, userID)
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
		_, err = c.mePlayerProfileDB().UpdateMePlayerProfile(ctx, database.UpdateMePlayerProfileParams{
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
		profile, _ := c.mePlayerProfileDB().GetMePlayerProfileByUserID(ctx, userID)
		traits, _ := c.mePlayerProfileDB().ListMePlayerProfileTraits(ctx, profile.ID)
		careerRows, _ := c.mePlayerProfileDB().ListMePlayerProfileCareerTeams(ctx, profile.ID)
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
	profile, err := c.mePlayerProfileDB().CreateMePlayerProfile(ctx, database.CreateMePlayerProfileParams{
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

	profile, err := c.mePlayerProfileDB().GetMePlayerProfileByUserID(ctx, userID)
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
	countryCode, ok := normalizeCountryCode(req.Country)
	if !ok {
		respondWithError(w, http.StatusBadRequest, "country must be a 2-letter ISO 3166-1 alpha-2 code (A-Z)")
		return
	}
	req.Country = countryCode
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
	updated, err := c.mePlayerProfileDB().UpdateMePlayerProfile(ctx, database.UpdateMePlayerProfileParams{
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
	traits, _ := c.mePlayerProfileDB().ListMePlayerProfileTraits(ctx, updated.ID)
	careerRows, _ := c.mePlayerProfileDB().ListMePlayerProfileCareerTeams(ctx, updated.ID)
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

// allowedTraitCodes is the enum for PUT /api/me/player-profile/traits (007 spec).
var allowedTraitCodes = map[string]bool{
	"LEADERSHIP": true, "FINESSE_SHOT": true, "PLAYMAKER": true,
	"SPEED_DRIBBLER": true, "LONG_SHOT_TAKER": true, "OUTSIDE_FOOT_SHOT": true,
	"POWER_HEADER": true, "FLAIR": true, "POWER_FREE_KICK": true,
}

func (c *Config) putMePlayerProfileTraits(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int32)
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	ctx := r.Context()

	var req struct {
		Traits []string `json:"traits"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if len(req.Traits) > 5 {
		respondWithError(w, http.StatusBadRequest, "Maximum 5 traits allowed")
		return
	}
	for _, t := range req.Traits {
		if !allowedTraitCodes[t] {
			respondWithError(w, http.StatusBadRequest, "Invalid trait code: "+t)
			return
		}
	}

	profile, err := c.mePlayerProfileDB().GetMePlayerProfileByUserID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Printf("[me_player_profile] GetMePlayerProfileByUserID error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile")
		return
	}

	if err := c.mePlayerProfileDB().DeleteMePlayerProfileTraitsByProfileID(ctx, profile.ID); err != nil {
		log.Printf("[me_player_profile] DeleteMePlayerProfileTraitsByProfileID error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to update traits")
		return
	}
	for _, traitCode := range req.Traits {
		if _, err := c.mePlayerProfileDB().InsertMePlayerProfileTrait(ctx, database.InsertMePlayerProfileTraitParams{
			MePlayerProfileID: profile.ID,
			TraitCode:         traitCode,
		}); err != nil {
			log.Printf("[me_player_profile] InsertMePlayerProfileTrait error: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to save traits")
			return
		}
	}
	traits, _ := c.mePlayerProfileDB().ListMePlayerProfileTraits(ctx, profile.ID)
	if traits == nil {
		traits = []string{}
	}
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"traits": traits})
}

func (c *Config) deleteMePlayerProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int32)
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	ctx := r.Context()

	profile, err := c.mePlayerProfileDB().GetMePlayerProfileByUserID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Printf("[me_player_profile] GetMePlayerProfileByUserID error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile")
		return
	}
	if err := c.mePlayerProfileDB().DeleteMePlayerProfile(ctx, profile.ID); err != nil {
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
