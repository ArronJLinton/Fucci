package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/ArronJLinton/fucci-api/internal/database"
)

func (c *Config) playerProfileDB() PlayerProfileStore {
	if c.PlayerProfileDB != nil {
		return c.PlayerProfileDB
	}
	return c.DB
}

// PlayerProfileResponse is the DTO for GET/POST/PUT /api/player-profile (007 spec).
type PlayerProfileResponse struct {
	ID          int32                        `json:"id"`
	Age         *int32                       `json:"age"`
	Country     string                       `json:"country"`
	Club        *string                      `json:"club"`
	IsFreeAgent bool                         `json:"is_free_agent"`
	Position    string                       `json:"position"`
	PhotoURL    *string                      `json:"photo_url"`
	Speed       int32                        `json:"speed"`
	Shooting    int32                        `json:"shooting"`
	Passing     int32                        `json:"passing"`
	Dribbling   int32                        `json:"dribbling"`
	Defending   int32                        `json:"defending"`
	Physical    int32                        `json:"physical"`
	Stamina     int32                        `json:"stamina"`
	Traits      []string                     `json:"traits"`
	CareerTeams []PlayerProfileCareerTeamDTO `json:"career_teams"`
}

// PlayerProfileCareerTeamDTO is one career team in the profile response.
type PlayerProfileCareerTeamDTO struct {
	ID        int32  `json:"id"`
	TeamName  string `json:"team_name"`
	StartYear int32  `json:"start_year"`
	EndYear   *int32 `json:"end_year"`
}

// ComparePlayerCatalogItem is one selectable player for compare search.
type ComparePlayerCatalogItem struct {
	ID             string  `json:"id"`
	DisplayName    string  `json:"display_name"`
	Age            *int32  `json:"age"`
	CountryCode    string  `json:"country_code"`
	CountryLabel   string  `json:"country_label"`
	Team           string  `json:"team"`
	PositionAbbrev string  `json:"position_abbrev"`
	PhotoURL       *string `json:"photo_url"`
	Rating         int32   `json:"rating"`
	Speed          int32   `json:"speed"`
	Shooting       int32   `json:"shooting"`
	Passing        int32   `json:"passing"`
	Dribbling      int32   `json:"dribbling"`
	Defending      int32   `json:"defending"`
	Physical       int32   `json:"physical"`
	Stamina        int32   `json:"stamina"`
	ValueLabel     string  `json:"value_label"`
	SeasonGoals    int32   `json:"season_goals"`
	SeasonLabel    string  `json:"season_label"`
}

// PlayerProfileInput is the body for POST/PUT /api/player-profile.
// Core attributes are optional: on first create they default to neutral 50 per stat; when a profile
// already exists, omitted core fields keep the stored values (same for POST upsert and PUT).
type PlayerProfileInput struct {
	Age         *int32  `json:"age"`
	Country     string  `json:"country"`
	Club        *string `json:"club"`
	IsFreeAgent *bool   `json:"is_free_agent"`
	Position    string  `json:"position"`
	Speed       *int32  `json:"speed"`
	Shooting    *int32  `json:"shooting"`
	Passing     *int32  `json:"passing"`
	Dribbling   *int32  `json:"dribbling"`
	Defending   *int32  `json:"defending"`
	Physical    *int32  `json:"physical"`
	Stamina     *int32  `json:"stamina"`
}

// coreAttrsBlock holds the seven persisted core stats (40–99).
type coreAttrsBlock struct {
	Speed     int32
	Shooting  int32
	Passing   int32
	Dribbling int32
	Defending int32
	Physical  int32
	Stamina   int32
}

func validateCoreAttrsOptional(req *PlayerProfileInput) string {
	checks := []struct {
		name string
		v    *int32
	}{
		{"speed", req.Speed},
		{"shooting", req.Shooting},
		{"passing", req.Passing},
		{"dribbling", req.Dribbling},
		{"defending", req.Defending},
		{"physical", req.Physical},
		{"stamina", req.Stamina},
	}
	for _, c := range checks {
		if c.v != nil && (*c.v < 40 || *c.v > 99) {
			return c.name + " must be between 40 and 99"
		}
	}
	return ""
}

func mergeCoreForPut(req *PlayerProfileInput, existing database.PlayerProfile) coreAttrsBlock {
	return coreAttrsBlock{
		Speed:     pickInt32(req.Speed, existing.Speed),
		Shooting:  pickInt32(req.Shooting, existing.Shooting),
		Passing:   pickInt32(req.Passing, existing.Passing),
		Dribbling: pickInt32(req.Dribbling, existing.Dribbling),
		Defending: pickInt32(req.Defending, existing.Defending),
		Physical:  pickInt32(req.Physical, existing.Physical),
		Stamina:   pickInt32(req.Stamina, existing.Stamina),
	}
}

func pickInt32(req *int32, fallback int32) int32 {
	if req != nil {
		return *req
	}
	return fallback
}

// optionalCoreUpsertArg is passed to UpsertPlayerProfile: Valid=false means omitted.
func optionalCoreUpsertArg(p *int32) sql.NullInt32 {
	if p == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: *p, Valid: true}
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

func completionPercentForCompare(row database.ListComparePlayerCatalogRow) int32 {
	count := int32(0)
	if row.Age.Valid {
		count++
	}
	if strings.TrimSpace(row.CountryCode) != "" {
		count++
	}
	if strings.TrimSpace(row.Position) != "" {
		count++
	}
	if row.TraitsCount > 0 {
		count++
	}
	return (count * 100) / 4
}

func compareDisplayLevel(traitsCount, completionPercent int32) int32 {
	level := 38 + traitsCount*9 + int32((float64(completionPercent)*0.2)+0.5)
	if level > 99 {
		return 99
	}
	return level
}

func positionAbbrev(position string) string {
	switch position {
	case "GK":
		return "GK"
	case "DEF":
		return "CB"
	case "MID":
		return "CM"
	case "FWD":
		return "ST"
	default:
		return ""
	}
}

func compareCatalogDisplayName(row database.ListComparePlayerCatalogRow) string {
	if row.DisplayName.Valid && strings.TrimSpace(row.DisplayName.String) != "" {
		return strings.ToUpper(strings.TrimSpace(row.DisplayName.String))
	}
	full := strings.TrimSpace(row.Firstname + " " + row.Lastname)
	if full == "" {
		return "PLAYER"
	}
	return strings.ToUpper(full)
}

func compareCatalogItem(row database.ListComparePlayerCatalogRow) ComparePlayerCatalogItem {
	var age *int32
	if row.Age.Valid {
		age = &row.Age.Int32
	}
	var photoURL *string
	if row.PhotoUrl.Valid {
		photoURL = &row.PhotoUrl.String
	} else if row.AvatarUrl.Valid {
		photoURL = &row.AvatarUrl.String
	}
	team := "—"
	if row.IsFreeAgent {
		team = "Free Agent"
	} else if row.ClubName.Valid && strings.TrimSpace(row.ClubName.String) != "" {
		team = strings.TrimSpace(row.ClubName.String)
	}
	countryCode := strings.ToUpper(strings.TrimSpace(row.CountryCode))
	completion := completionPercentForCompare(row)
	return ComparePlayerCatalogItem{
		ID:             "profile-" + strconv.Itoa(int(row.ID)),
		DisplayName:    compareCatalogDisplayName(row),
		Age:            age,
		CountryCode:    countryCode,
		// Placeholder until API ships authoritative country labels.
		CountryLabel:   "—",
		Team:           team,
		PositionAbbrev: positionAbbrev(row.Position),
		PhotoURL:       photoURL,
		Rating:         compareDisplayLevel(row.TraitsCount, completion),
		Speed:          row.Speed,
		Shooting:       row.Shooting,
		Passing:        row.Passing,
		Dribbling:      row.Dribbling,
		Defending:      row.Defending,
		Physical:       row.Physical,
		Stamina:        row.Stamina,
		ValueLabel:     "—",
		SeasonGoals:    0,
		// Placeholder until season data is available in profile domain.
		SeasonLabel:    "—",
	}
}

func (c *Config) getPlayerProfileCatalog(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int32)
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	ctx := r.Context()
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	const defaultLimit int32 = 80
	rows, err := c.playerProfileDB().ListComparePlayerCatalog(ctx, database.ListComparePlayerCatalogParams{
		Search: query,
		Limit:  defaultLimit,
	})
	if err != nil {
		log.Printf("[player_profile] ListComparePlayerCatalog error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get compare players")
		return
	}
	out := make([]ComparePlayerCatalogItem, 0, len(rows))
	for _, row := range rows {
		if row.UserID == userID {
			continue
		}
		out = append(out, compareCatalogItem(row))
	}
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"players": out})
}

func (c *Config) getPlayerProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int32)
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	ctx := r.Context()

	profile, err := c.playerProfileDB().GetPlayerProfileByUserID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Profile not found")
			return
		}
		log.Printf("[player_profile] GetPlayerProfileByUserID error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile")
		return
	}

	traits, err := c.playerProfileDB().ListPlayerProfileTraits(ctx, profile.ID)
	if err != nil {
		log.Printf("[player_profile] ListPlayerProfileTraits error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile")
		return
	}
	careerRows, err := c.playerProfileDB().ListPlayerProfileCareerTeams(ctx, profile.ID)
	if err != nil {
		log.Printf("[player_profile] ListPlayerProfileCareerTeams error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile")
		return
	}
	careerTeams := make([]PlayerProfileCareerTeamDTO, 0, len(careerRows))
	for _, row := range careerRows {
		var endYear *int32
		if row.EndYear.Valid {
			endYear = &row.EndYear.Int32
		}
		careerTeams = append(careerTeams, PlayerProfileCareerTeamDTO{
			ID:        row.ID,
			TeamName:  row.TeamName,
			StartYear: row.StartYear,
			EndYear:   endYear,
		})
	}

	resp := profileToResponse(profile, traits, careerTeams)
	respondWithJSON(w, http.StatusOK, resp)
}

func (c *Config) postPlayerProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int32)
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	ctx := r.Context()

	var req PlayerProfileInput
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
	if msg := validateCoreAttrsOptional(&req); msg != "" {
		respondWithError(w, http.StatusBadRequest, msg)
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

	profile, err := c.playerProfileDB().UpsertPlayerProfile(ctx, database.UpsertPlayerProfileParams{
		UserID:      userID,
		Age:         age,
		CountryCode: req.Country,
		ClubName:    club,
		IsFreeAgent: isFreeAgent,
		Position:    req.Position,
		Speed:       optionalCoreUpsertArg(req.Speed),
		Shooting:    optionalCoreUpsertArg(req.Shooting),
		Passing:     optionalCoreUpsertArg(req.Passing),
		Dribbling:   optionalCoreUpsertArg(req.Dribbling),
		Defending:   optionalCoreUpsertArg(req.Defending),
		Physical:    optionalCoreUpsertArg(req.Physical),
		Stamina:     optionalCoreUpsertArg(req.Stamina),
	})
	if err != nil {
		log.Printf("[player_profile] UpsertPlayerProfile error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to save profile")
		return
	}

	traits, err := c.playerProfileDB().ListPlayerProfileTraits(ctx, profile.ID)
	if err != nil {
		log.Printf("[player_profile] ListPlayerProfileTraits error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile")
		return
	}
	careerRows, err := c.playerProfileDB().ListPlayerProfileCareerTeams(ctx, profile.ID)
	if err != nil {
		log.Printf("[player_profile] ListPlayerProfileCareerTeams error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile")
		return
	}
	careerTeams := make([]PlayerProfileCareerTeamDTO, 0, len(careerRows))
	for _, row := range careerRows {
		var endYear *int32
		if row.EndYear.Valid {
			endYear = &row.EndYear.Int32
		}
		careerTeams = append(careerTeams, PlayerProfileCareerTeamDTO{ID: row.ID, TeamName: row.TeamName, StartYear: row.StartYear, EndYear: endYear})
	}
	respondWithJSON(w, http.StatusOK, profileToResponse(profile, traits, careerTeams))
}

func (c *Config) putPlayerProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int32)
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	ctx := r.Context()

	profile, err := c.playerProfileDB().GetPlayerProfileByUserID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Profile not found")
			return
		}
		log.Printf("[player_profile] GetPlayerProfileByUserID error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile")
		return
	}

	var req PlayerProfileInput
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
	if msg := validateCoreAttrsOptional(&req); msg != "" {
		respondWithError(w, http.StatusBadRequest, msg)
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
	core := mergeCoreForPut(&req, profile)
	updated, err := c.playerProfileDB().UpdatePlayerProfileRow(ctx, database.UpdatePlayerProfileRowParams{
		ID:          profile.ID,
		Age:         age,
		CountryCode: req.Country,
		ClubName:    club,
		IsFreeAgent: isFreeAgent,
		Position:    req.Position,
		PhotoUrl:    profile.PhotoUrl,
		Speed:       core.Speed,
		Shooting:    core.Shooting,
		Passing:     core.Passing,
		Dribbling:   core.Dribbling,
		Defending:   core.Defending,
		Physical:    core.Physical,
		Stamina:     core.Stamina,
	})
	if err != nil {
		log.Printf("[player_profile] UpdatePlayerProfileRow error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}
	traits, err := c.playerProfileDB().ListPlayerProfileTraits(ctx, updated.ID)
	if err != nil {
		log.Printf("[player_profile] ListPlayerProfileTraits error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile")
		return
	}
	careerRows, err := c.playerProfileDB().ListPlayerProfileCareerTeams(ctx, updated.ID)
	if err != nil {
		log.Printf("[player_profile] ListPlayerProfileCareerTeams error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile")
		return
	}
	careerTeams := make([]PlayerProfileCareerTeamDTO, 0, len(careerRows))
	for _, row := range careerRows {
		var endYear *int32
		if row.EndYear.Valid {
			endYear = &row.EndYear.Int32
		}
		careerTeams = append(careerTeams, PlayerProfileCareerTeamDTO{ID: row.ID, TeamName: row.TeamName, StartYear: row.StartYear, EndYear: endYear})
	}
	respondWithJSON(w, http.StatusOK, profileToResponse(updated, traits, careerTeams))
}

// allowedTraitCodes is the enum for PUT /api/player-profile/traits (007 spec).
var allowedTraitCodes = map[string]bool{
	"LEADERSHIP": true, "FINESSE_SHOT": true, "PLAYMAKER": true,
	"SPEED_DRIBBLER": true, "LONG_SHOT_TAKER": true, "OUTSIDE_FOOT_SHOT": true,
	"POWER_HEADER": true, "FLAIR": true, "POWER_FREE_KICK": true,
}

// dedupeTraitCodesPreserveOrder removes duplicate trait codes so INSERT cannot hit
// UNIQUE(player_profile_id, trait_code); first occurrence wins.
func dedupeTraitCodesPreserveOrder(codes []string) []string {
	if len(codes) == 0 {
		return codes
	}
	seen := make(map[string]struct{}, len(codes))
	out := make([]string, 0, len(codes))
	for _, code := range codes {
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out
}

func (c *Config) putPlayerProfileTraits(w http.ResponseWriter, r *http.Request) {
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
	req.Traits = dedupeTraitCodesPreserveOrder(req.Traits)
	for _, t := range req.Traits {
		if !allowedTraitCodes[t] {
			respondWithError(w, http.StatusBadRequest, "Invalid trait code: "+t)
			return
		}
	}

	profile, err := c.playerProfileDB().GetPlayerProfileByUserID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Profile not found")
			return
		}
		log.Printf("[player_profile] GetPlayerProfileByUserID error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile")
		return
	}

	// PUT semantics: the body is the full desired trait set. We must replace what is stored,
	// not only append—otherwise traits the user removed would never disappear. That requires
	// clearing existing rows (delete-all or a diff); delete-then-insert is simple and correct.
	//
	// Production: one transaction so a failed insert does not leave traits wiped or half-written.
	if c.PlayerProfileDB == nil && c.DBConn != nil && c.DB != nil {
		tx, err := c.DBConn.BeginTx(ctx, nil)
		if err != nil {
			log.Printf("[player_profile] BeginTx (traits) error: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to update traits")
			return
		}
		defer func() { _ = tx.Rollback() }()

		q := c.DB.WithTx(tx)
		if err := q.DeletePlayerProfileTraitsByProfileID(ctx, profile.ID); err != nil {
			log.Printf("[player_profile] DeletePlayerProfileTraitsByProfileID error: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to update traits")
			return
		}
		for _, traitCode := range req.Traits {
			if _, err := q.InsertPlayerProfileTrait(ctx, database.InsertPlayerProfileTraitParams{
				PlayerProfileID: profile.ID,
				TraitCode:       traitCode,
			}); err != nil {
				log.Printf("[player_profile] InsertPlayerProfileTrait error: %v", err)
				respondWithError(w, http.StatusInternalServerError, "Failed to save traits")
				return
			}
		}
		traits, err := q.ListPlayerProfileTraits(ctx, profile.ID)
		if err != nil {
			log.Printf("[player_profile] ListPlayerProfileTraits error: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to update traits")
			return
		}
		if err := tx.Commit(); err != nil {
			log.Printf("[player_profile] Commit (traits) error: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to update traits")
			return
		}
		if traits == nil {
			traits = []string{}
		}
		respondWithJSON(w, http.StatusOK, map[string]interface{}{"traits": traits})
		return
	}

	if err := c.playerProfileDB().DeletePlayerProfileTraitsByProfileID(ctx, profile.ID); err != nil {
		log.Printf("[player_profile] DeletePlayerProfileTraitsByProfileID error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to update traits")
		return
	}
	for _, traitCode := range req.Traits {
		if _, err := c.playerProfileDB().InsertPlayerProfileTrait(ctx, database.InsertPlayerProfileTraitParams{
			PlayerProfileID: profile.ID,
			TraitCode:       traitCode,
		}); err != nil {
			log.Printf("[player_profile] InsertPlayerProfileTrait error: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to save traits")
			return
		}
	}
	traits, err := c.playerProfileDB().ListPlayerProfileTraits(ctx, profile.ID)
	if err != nil {
		log.Printf("[player_profile] ListPlayerProfileTraits error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to update traits")
		return
	}
	if traits == nil {
		traits = []string{}
	}
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"traits": traits})
}

func (c *Config) deletePlayerProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int32)
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	ctx := r.Context()

	profile, err := c.playerProfileDB().GetPlayerProfileByUserID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Profile not found")
			return
		}
		log.Printf("[player_profile] GetPlayerProfileByUserID error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile")
		return
	}
	if err := c.playerProfileDB().DeletePlayerProfileRow(ctx, profile.ID); err != nil {
		log.Printf("[player_profile] DeletePlayerProfileRow error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to delete profile")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func profileToResponse(p database.PlayerProfile, traits []string, careerTeams []PlayerProfileCareerTeamDTO) PlayerProfileResponse {
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
		careerTeams = []PlayerProfileCareerTeamDTO{}
	}
	return PlayerProfileResponse{
		ID:          p.ID,
		Age:         age,
		Country:     p.CountryCode,
		Club:        club,
		IsFreeAgent: p.IsFreeAgent,
		Position:    p.Position,
		PhotoURL:    photoURL,
		Speed:       p.Speed,
		Shooting:    p.Shooting,
		Passing:     p.Passing,
		Dribbling:   p.Dribbling,
		Defending:   p.Defending,
		Physical:    p.Physical,
		Stamina:     p.Stamina,
		Traits:      traits,
		CareerTeams: careerTeams,
	}
}
