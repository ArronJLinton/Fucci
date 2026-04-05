package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// upsertArgCoreInt32 interprets nullable core args from UpsertPlayerProfile (nil = omitted).
func upsertArgCoreInt32(tb testing.TB, v interface{}, ifNull int32) int32 {
	tb.Helper()
	if v == nil {
		return ifNull
	}
	switch x := v.(type) {
	case int32:
		return x
	case int64:
		return int32(x)
	case int:
		return int32(x)
	default:
		tb.Fatalf("upsertArgCoreInt32: unexpected type %T for UpsertPlayerProfile core arg: %#v", v, v)
		return 0
	}
}

// stubPlayerProfileStore implements PlayerProfileStore with per-method func hooks (nil => safe default).
type stubPlayerProfileStore struct {
	GetPlayerProfileByUserIDFn             func(ctx context.Context, userID int32) (database.PlayerProfile, error)
	UpsertPlayerProfileFn                  func(ctx context.Context, arg database.UpsertPlayerProfileParams) (database.PlayerProfile, error)
	UpdatePlayerProfileRowFn               func(ctx context.Context, arg database.UpdatePlayerProfileRowParams) (database.PlayerProfile, error)
	DeletePlayerProfileRowFn               func(ctx context.Context, id int32) error
	ListPlayerProfileTraitsFn              func(ctx context.Context, playerProfileID int32) ([]string, error)
	ListPlayerProfileCareerTeamsFn         func(ctx context.Context, playerProfileID int32) ([]database.PlayerProfileCareerTeam, error)
	DeletePlayerProfileTraitsByProfileIDFn func(ctx context.Context, playerProfileID int32) error
	InsertPlayerProfileTraitFn             func(ctx context.Context, arg database.InsertPlayerProfileTraitParams) (database.PlayerProfileTrait, error)
}

func (s *stubPlayerProfileStore) GetPlayerProfileByUserID(ctx context.Context, userID int32) (database.PlayerProfile, error) {
	if s.GetPlayerProfileByUserIDFn != nil {
		return s.GetPlayerProfileByUserIDFn(ctx, userID)
	}
	return database.PlayerProfile{}, sql.ErrNoRows
}

func (s *stubPlayerProfileStore) UpsertPlayerProfile(ctx context.Context, arg database.UpsertPlayerProfileParams) (database.PlayerProfile, error) {
	if s.UpsertPlayerProfileFn != nil {
		return s.UpsertPlayerProfileFn(ctx, arg)
	}
	return database.PlayerProfile{}, assert.AnError
}

func (s *stubPlayerProfileStore) UpdatePlayerProfileRow(ctx context.Context, arg database.UpdatePlayerProfileRowParams) (database.PlayerProfile, error) {
	if s.UpdatePlayerProfileRowFn != nil {
		return s.UpdatePlayerProfileRowFn(ctx, arg)
	}
	return database.PlayerProfile{}, assert.AnError
}

func (s *stubPlayerProfileStore) DeletePlayerProfileRow(ctx context.Context, id int32) error {
	if s.DeletePlayerProfileRowFn != nil {
		return s.DeletePlayerProfileRowFn(ctx, id)
	}
	return assert.AnError
}

func (s *stubPlayerProfileStore) ListPlayerProfileTraits(ctx context.Context, playerProfileID int32) ([]string, error) {
	if s.ListPlayerProfileTraitsFn != nil {
		return s.ListPlayerProfileTraitsFn(ctx, playerProfileID)
	}
	return nil, nil
}

func (s *stubPlayerProfileStore) ListPlayerProfileCareerTeams(ctx context.Context, playerProfileID int32) ([]database.PlayerProfileCareerTeam, error) {
	if s.ListPlayerProfileCareerTeamsFn != nil {
		return s.ListPlayerProfileCareerTeamsFn(ctx, playerProfileID)
	}
	return nil, nil
}

func (s *stubPlayerProfileStore) DeletePlayerProfileTraitsByProfileID(ctx context.Context, playerProfileID int32) error {
	if s.DeletePlayerProfileTraitsByProfileIDFn != nil {
		return s.DeletePlayerProfileTraitsByProfileIDFn(ctx, playerProfileID)
	}
	return nil
}

func (s *stubPlayerProfileStore) InsertPlayerProfileTrait(ctx context.Context, arg database.InsertPlayerProfileTraitParams) (database.PlayerProfileTrait, error) {
	if s.InsertPlayerProfileTraitFn != nil {
		return s.InsertPlayerProfileTraitFn(ctx, arg)
	}
	return database.PlayerProfileTrait{}, assert.AnError
}

func playerProfileTestRequest(method, path string, body interface{}, userID int32) *http.Request {
	var r *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		r = httptest.NewRequest(method, path, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	ctx := context.WithValue(r.Context(), "user_id", userID)
	return r.WithContext(ctx)
}

func sampleProfile(userID int32, id int32) database.PlayerProfile {
	// Neutral core block (matches POST/PUT default 50 for omitted cores).
	return database.PlayerProfile{
		ID:          id,
		UserID:      userID,
		Age:         sql.NullInt32{Int32: 22, Valid: true},
		CountryCode: "US",
		ClubName:    sql.NullString{String: "Test FC", Valid: true},
		IsFreeAgent: false,
		Position:    "MID",
		Speed:       50,
		Shooting:    50,
		Passing:     50,
		Dribbling:   50,
		Defending:   50,
		Physical:    50,
		Stamina:     50,
		CreatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
	}
}

func TestGetMyPlayerProfile_NotFound(t *testing.T) {
	uid := int32(42)
	stub := &stubPlayerProfileStore{
		GetPlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.PlayerProfile, error) {
			assert.Equal(t, uid, userID)
			return database.PlayerProfile{}, sql.ErrNoRows
		},
	}
	cfg := &Config{PlayerProfileDB: stub}
	rec := httptest.NewRecorder()
	cfg.getPlayerProfile(rec, playerProfileTestRequest(http.MethodGet, "/player-profile", nil, uid))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "not found")
}

func TestGetMyPlayerProfile_OK(t *testing.T) {
	uid := int32(7)
	p := sampleProfile(uid, 100)
	stub := &stubPlayerProfileStore{
		GetPlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.PlayerProfile, error) {
			return p, nil
		},
		ListPlayerProfileTraitsFn: func(ctx context.Context, pid int32) ([]string, error) {
			assert.Equal(t, p.ID, pid)
			return []string{"FLAIR", "PLAYMAKER"}, nil
		},
		ListPlayerProfileCareerTeamsFn: func(ctx context.Context, pid int32) ([]database.PlayerProfileCareerTeam, error) {
			return []database.PlayerProfileCareerTeam{
				{ID: 1, PlayerProfileID: pid, TeamName: "Old Club", StartYear: 2020, EndYear: sql.NullInt32{Int32: 2023, Valid: true}},
			}, nil
		},
	}
	cfg := &Config{PlayerProfileDB: stub}
	rec := httptest.NewRecorder()
	cfg.getPlayerProfile(rec, playerProfileTestRequest(http.MethodGet, "/player-profile", nil, uid))

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp PlayerProfileResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, p.ID, resp.ID)
	assert.Equal(t, "US", resp.Country)
	assert.Equal(t, "MID", resp.Position)
	assert.Equal(t, []string{"FLAIR", "PLAYMAKER"}, resp.Traits)
	assert.Equal(t, int32(50), resp.Speed)
	assert.Equal(t, int32(50), resp.Stamina)
	require.Len(t, resp.CareerTeams, 1)
	assert.Equal(t, "Old Club", resp.CareerTeams[0].TeamName)
}

func TestGetMyPlayerProfile_ListTraitsError(t *testing.T) {
	uid := int32(7)
	p := sampleProfile(uid, 100)
	stub := &stubPlayerProfileStore{
		GetPlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.PlayerProfile, error) {
			return p, nil
		},
		ListPlayerProfileTraitsFn: func(ctx context.Context, pid int32) ([]string, error) {
			return nil, assert.AnError
		},
	}
	cfg := &Config{PlayerProfileDB: stub}
	rec := httptest.NewRecorder()
	cfg.getPlayerProfile(rec, playerProfileTestRequest(http.MethodGet, "/player-profile", nil, uid))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetMyPlayerProfile_ListCareerTeamsError(t *testing.T) {
	uid := int32(7)
	p := sampleProfile(uid, 100)
	stub := &stubPlayerProfileStore{
		GetPlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.PlayerProfile, error) {
			return p, nil
		},
		ListPlayerProfileTraitsFn: func(ctx context.Context, pid int32) ([]string, error) {
			return nil, nil
		},
		ListPlayerProfileCareerTeamsFn: func(ctx context.Context, pid int32) ([]database.PlayerProfileCareerTeam, error) {
			return nil, assert.AnError
		},
	}
	cfg := &Config{PlayerProfileDB: stub}
	rec := httptest.NewRecorder()
	cfg.getPlayerProfile(rec, playerProfileTestRequest(http.MethodGet, "/player-profile", nil, uid))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestPostMyPlayerProfile_InvalidCoreAttr(t *testing.T) {
	cfg := &Config{PlayerProfileDB: &stubPlayerProfileStore{}}
	body := map[string]interface{}{"country": "GB", "position": "DEF", "speed": 39}
	rec := httptest.NewRecorder()
	cfg.postPlayerProfile(rec, playerProfileTestRequest(http.MethodPost, "/player-profile", body, 1))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPostMyPlayerProfile_Create(t *testing.T) {
	uid := int32(5)
	var upsertArg database.UpsertPlayerProfileParams
	stub := &stubPlayerProfileStore{
		UpsertPlayerProfileFn: func(ctx context.Context, arg database.UpsertPlayerProfileParams) (database.PlayerProfile, error) {
			upsertArg = arg
			const d = int32(50)
			return database.PlayerProfile{
				ID: 1, UserID: arg.UserID, CountryCode: arg.CountryCode, Position: arg.Position,
				IsFreeAgent: arg.IsFreeAgent, Age: arg.Age, ClubName: arg.ClubName,
				Speed: upsertArgCoreInt32(t, arg.Speed, d), Shooting: upsertArgCoreInt32(t, arg.Shooting, d),
				Passing: upsertArgCoreInt32(t, arg.Passing, d), Dribbling: upsertArgCoreInt32(t, arg.Dribbling, d),
				Defending: upsertArgCoreInt32(t, arg.Defending, d), Physical: upsertArgCoreInt32(t, arg.Physical, d),
				Stamina: upsertArgCoreInt32(t, arg.Stamina, d),
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}
	cfg := &Config{PlayerProfileDB: stub}
	body := map[string]interface{}{"country": "gb", "position": "DEF", "age": 18}
	rec := httptest.NewRecorder()
	cfg.postPlayerProfile(rec, playerProfileTestRequest(http.MethodPost, "/player-profile", body, uid))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, uid, upsertArg.UserID)
	assert.Equal(t, "GB", upsertArg.CountryCode)
	assert.Equal(t, "DEF", upsertArg.Position)
	// Omitted cores → NULL; SQL applies COALESCE to 50 on insert.
	assert.Nil(t, upsertArg.Speed)
	assert.Nil(t, upsertArg.Shooting)
	assert.Nil(t, upsertArg.Passing)
	assert.Nil(t, upsertArg.Dribbling)
	assert.Nil(t, upsertArg.Defending)
	assert.Nil(t, upsertArg.Physical)
	assert.Nil(t, upsertArg.Stamina)
	var resp PlayerProfileResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "GB", resp.Country)
	assert.Equal(t, int32(1), resp.ID)
}

func TestPostMyPlayerProfile_UpdateExisting(t *testing.T) {
	uid := int32(3)
	existing := sampleProfile(uid, 99)
	// Non-default cores so we assert POST preserves them when body omits core fields.
	existing.Speed, existing.Shooting = 72, 65
	existing.Passing, existing.Dribbling = 58, 61
	existing.Defending, existing.Physical, existing.Stamina = 80, 77, 88
	updated := existing
	updated.CountryCode = "DE"
	updated.Position = "FWD"
	var upsertArg database.UpsertPlayerProfileParams

	stub := &stubPlayerProfileStore{
		UpsertPlayerProfileFn: func(ctx context.Context, arg database.UpsertPlayerProfileParams) (database.PlayerProfile, error) {
			upsertArg = arg
			return updated, nil
		},
		ListPlayerProfileTraitsFn: func(ctx context.Context, pid int32) ([]string, error) {
			assert.Equal(t, updated.ID, pid)
			return []string{}, nil
		},
		ListPlayerProfileCareerTeamsFn: func(ctx context.Context, pid int32) ([]database.PlayerProfileCareerTeam, error) {
			return nil, nil
		},
	}
	cfg := &Config{PlayerProfileDB: stub}
	body := map[string]interface{}{"country": "DE", "position": "FWD"}
	rec := httptest.NewRecorder()
	cfg.postPlayerProfile(rec, playerProfileTestRequest(http.MethodPost, "/player-profile", body, uid))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, uid, upsertArg.UserID)
	assert.Equal(t, "DE", upsertArg.CountryCode)
	assert.Equal(t, "FWD", upsertArg.Position)
	// Omitted cores → NULL; DB merges with existing row (no pre-read in handler).
	assert.Nil(t, upsertArg.Speed)
	assert.Nil(t, upsertArg.Shooting)
	assert.Nil(t, upsertArg.Passing)
	assert.Nil(t, upsertArg.Dribbling)
	assert.Nil(t, upsertArg.Defending)
	assert.Nil(t, upsertArg.Physical)
	assert.Nil(t, upsertArg.Stamina)
	var resp PlayerProfileResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "DE", resp.Country)
	assert.Equal(t, "FWD", resp.Position)
}

func TestPutMyPlayerProfile_NotFound(t *testing.T) {
	stub := &stubPlayerProfileStore{
		GetPlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.PlayerProfile, error) {
			return database.PlayerProfile{}, sql.ErrNoRows
		},
	}
	cfg := &Config{PlayerProfileDB: stub}
	body := map[string]interface{}{"country": "FR", "position": "GK"}
	rec := httptest.NewRecorder()
	cfg.putPlayerProfile(rec, playerProfileTestRequest(http.MethodPut, "/player-profile", body, 1))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPutMyPlayerProfile_InvalidCoreAttr(t *testing.T) {
	uid := int32(1)
	p := sampleProfile(uid, 40)
	stub := &stubPlayerProfileStore{
		GetPlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.PlayerProfile, error) {
			assert.Equal(t, uid, userID)
			return p, nil
		},
	}
	cfg := &Config{PlayerProfileDB: stub}
	body := map[string]interface{}{"country": "GB", "position": "DEF", "speed": 39}
	rec := httptest.NewRecorder()
	cfg.putPlayerProfile(rec, playerProfileTestRequest(http.MethodPut, "/player-profile", body, uid))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPutMyPlayerProfile_OK(t *testing.T) {
	uid := int32(2)
	p := sampleProfile(uid, 55)
	updated := p
	updated.CountryCode = "CA"
	updated.Position = "GK"
	var sawUpdate bool
	stub := &stubPlayerProfileStore{
		GetPlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.PlayerProfile, error) {
			return p, nil
		},
		UpdatePlayerProfileRowFn: func(ctx context.Context, arg database.UpdatePlayerProfileRowParams) (database.PlayerProfile, error) {
			sawUpdate = true
			assert.Equal(t, p.ID, arg.ID)
			assert.Equal(t, "CA", arg.CountryCode)
			assert.Equal(t, "GK", arg.Position)
			// No core fields in body: preserve existing from sampleProfile (MID block)
			assert.Equal(t, p.Speed, arg.Speed)
			assert.Equal(t, p.Stamina, arg.Stamina)
			return updated, nil
		},
		ListPlayerProfileTraitsFn: func(ctx context.Context, pid int32) ([]string, error) {
			return []string{"LEADERSHIP"}, nil
		},
		ListPlayerProfileCareerTeamsFn: func(ctx context.Context, pid int32) ([]database.PlayerProfileCareerTeam, error) {
			return nil, nil
		},
	}
	cfg := &Config{PlayerProfileDB: stub}
	body := map[string]interface{}{"country": "ca", "position": "GK"}
	rec := httptest.NewRecorder()
	cfg.putPlayerProfile(rec, playerProfileTestRequest(http.MethodPut, "/player-profile", body, uid))

	assert.True(t, sawUpdate)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp PlayerProfileResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "CA", resp.Country)
	assert.Equal(t, []string{"LEADERSHIP"}, resp.Traits)
}

func TestDeletePlayerProfileRow_OK(t *testing.T) {
	uid := int32(8)
	p := sampleProfile(uid, 200)
	var deletedID int32
	stub := &stubPlayerProfileStore{
		GetPlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.PlayerProfile, error) {
			return p, nil
		},
		DeletePlayerProfileRowFn: func(ctx context.Context, id int32) error {
			deletedID = id
			return nil
		},
	}
	cfg := &Config{PlayerProfileDB: stub}
	rec := httptest.NewRecorder()
	cfg.deletePlayerProfile(rec, playerProfileTestRequest(http.MethodDelete, "/player-profile", nil, uid))

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, p.ID, deletedID)
}

func TestDeletePlayerProfileRow_NotFound(t *testing.T) {
	stub := &stubPlayerProfileStore{
		GetPlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.PlayerProfile, error) {
			return database.PlayerProfile{}, sql.ErrNoRows
		},
	}
	cfg := &Config{PlayerProfileDB: stub}
	rec := httptest.NewRecorder()
	cfg.deletePlayerProfile(rec, playerProfileTestRequest(http.MethodDelete, "/player-profile", nil, 1))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPutMyPlayerProfileTraits_Replacement(t *testing.T) {
	uid := int32(1)
	p := sampleProfile(uid, 30)
	var deleteCalls, insertOrder []string
	traitsState := []string{"OLD_TRAIT"}

	stub := &stubPlayerProfileStore{
		GetPlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.PlayerProfile, error) {
			return p, nil
		},
		DeletePlayerProfileTraitsByProfileIDFn: func(ctx context.Context, playerProfileID int32) error {
			deleteCalls = append(deleteCalls, "ok")
			traitsState = nil
			return nil
		},
		InsertPlayerProfileTraitFn: func(ctx context.Context, arg database.InsertPlayerProfileTraitParams) (database.PlayerProfileTrait, error) {
			insertOrder = append(insertOrder, arg.TraitCode)
			traitsState = append(traitsState, arg.TraitCode)
			return database.PlayerProfileTrait{ID: int32(len(insertOrder)), PlayerProfileID: arg.PlayerProfileID, TraitCode: arg.TraitCode}, nil
		},
		ListPlayerProfileTraitsFn: func(ctx context.Context, playerProfileID int32) ([]string, error) {
			out := make([]string, len(traitsState))
			copy(out, traitsState)
			return out, nil
		},
	}
	cfg := &Config{PlayerProfileDB: stub}
	body := map[string]interface{}{"traits": []string{"FLAIR", "PLAYMAKER", "SPEED_DRIBBLER"}}
	rec := httptest.NewRecorder()
	cfg.putPlayerProfileTraits(rec, playerProfileTestRequest(http.MethodPut, "/player-profile/traits", body, uid))

	assert.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, deleteCalls, 1)
	assert.Equal(t, []string{"FLAIR", "PLAYMAKER", "SPEED_DRIBBLER"}, insertOrder)
	var out map[string][]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, []string{"FLAIR", "PLAYMAKER", "SPEED_DRIBBLER"}, out["traits"])
}

func TestPutMyPlayerProfileTraits_AllAllowedCodes_OK(t *testing.T) {
	uid := int32(1)
	p := sampleProfile(uid, 30)
	var insertOrder []string
	traitsState := []string(nil)

	stub := &stubPlayerProfileStore{
		GetPlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.PlayerProfile, error) {
			return p, nil
		},
		DeletePlayerProfileTraitsByProfileIDFn: func(ctx context.Context, playerProfileID int32) error {
			traitsState = nil
			return nil
		},
		InsertPlayerProfileTraitFn: func(ctx context.Context, arg database.InsertPlayerProfileTraitParams) (database.PlayerProfileTrait, error) {
			insertOrder = append(insertOrder, arg.TraitCode)
			traitsState = append(traitsState, arg.TraitCode)
			return database.PlayerProfileTrait{ID: int32(len(insertOrder)), PlayerProfileID: arg.PlayerProfileID, TraitCode: arg.TraitCode}, nil
		},
		ListPlayerProfileTraitsFn: func(ctx context.Context, playerProfileID int32) ([]string, error) {
			out := make([]string, len(traitsState))
			copy(out, traitsState)
			return out, nil
		},
	}
	allNine := []string{
		"LEADERSHIP", "FINESSE_SHOT", "PLAYMAKER", "SPEED_DRIBBLER", "LONG_SHOT_TAKER",
		"OUTSIDE_FOOT_SHOT", "POWER_HEADER", "FLAIR", "POWER_FREE_KICK",
	}
	cfg := &Config{PlayerProfileDB: stub}
	body := map[string]interface{}{"traits": allNine}
	rec := httptest.NewRecorder()
	cfg.putPlayerProfileTraits(rec, playerProfileTestRequest(http.MethodPut, "/player-profile/traits", body, uid))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, allNine, insertOrder)
	var out map[string][]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, allNine, out["traits"])
}

func TestDedupeTraitCodesPreserveOrder(t *testing.T) {
	assert.Nil(t, dedupeTraitCodesPreserveOrder(nil))
	assert.Equal(t, []string{}, dedupeTraitCodesPreserveOrder([]string{}))
	assert.Equal(t, []string{"A"}, dedupeTraitCodesPreserveOrder([]string{"A"}))
	assert.Equal(t, []string{"A", "B", "C"}, dedupeTraitCodesPreserveOrder([]string{"A", "B", "A", "C", "B"}))
}

func TestPutMyPlayerProfileTraits_DedupesDuplicates(t *testing.T) {
	uid := int32(1)
	p := sampleProfile(uid, 30)
	var insertOrder []string
	traitsState := []string(nil)

	stub := &stubPlayerProfileStore{
		GetPlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.PlayerProfile, error) {
			return p, nil
		},
		DeletePlayerProfileTraitsByProfileIDFn: func(ctx context.Context, playerProfileID int32) error {
			traitsState = nil
			return nil
		},
		InsertPlayerProfileTraitFn: func(ctx context.Context, arg database.InsertPlayerProfileTraitParams) (database.PlayerProfileTrait, error) {
			insertOrder = append(insertOrder, arg.TraitCode)
			traitsState = append(traitsState, arg.TraitCode)
			return database.PlayerProfileTrait{ID: int32(len(insertOrder)), PlayerProfileID: arg.PlayerProfileID, TraitCode: arg.TraitCode}, nil
		},
		ListPlayerProfileTraitsFn: func(ctx context.Context, playerProfileID int32) ([]string, error) {
			out := make([]string, len(traitsState))
			copy(out, traitsState)
			return out, nil
		},
	}
	cfg := &Config{PlayerProfileDB: stub}
	body := map[string]interface{}{
		"traits": []string{"FLAIR", "PLAYMAKER", "FLAIR", "PLAYMAKER", "SPEED_DRIBBLER"},
	}
	rec := httptest.NewRecorder()
	cfg.putPlayerProfileTraits(rec, playerProfileTestRequest(http.MethodPut, "/player-profile/traits", body, uid))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, []string{"FLAIR", "PLAYMAKER", "SPEED_DRIBBLER"}, insertOrder)
	var out map[string][]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, []string{"FLAIR", "PLAYMAKER", "SPEED_DRIBBLER"}, out["traits"])
}

func TestPutMyPlayerProfileTraits_SixRawFiveUniqueAfterDedupe_OK(t *testing.T) {
	uid := int32(1)
	p := sampleProfile(uid, 30)
	stub := &stubPlayerProfileStore{
		GetPlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.PlayerProfile, error) {
			return p, nil
		},
		DeletePlayerProfileTraitsByProfileIDFn: func(ctx context.Context, playerProfileID int32) error {
			return nil
		},
		InsertPlayerProfileTraitFn: func(ctx context.Context, arg database.InsertPlayerProfileTraitParams) (database.PlayerProfileTrait, error) {
			return database.PlayerProfileTrait{}, nil
		},
		ListPlayerProfileTraitsFn: func(ctx context.Context, playerProfileID int32) ([]string, error) {
			return []string{"LEADERSHIP", "FINESSE_SHOT", "PLAYMAKER", "SPEED_DRIBBLER", "LONG_SHOT_TAKER"}, nil
		},
	}
	cfg := &Config{PlayerProfileDB: stub}
	body := map[string]interface{}{
		"traits": []string{
			"LEADERSHIP", "LEADERSHIP", "FINESSE_SHOT", "PLAYMAKER", "SPEED_DRIBBLER", "LONG_SHOT_TAKER",
		},
	}
	rec := httptest.NewRecorder()
	cfg.putPlayerProfileTraits(rec, playerProfileTestRequest(http.MethodPut, "/player-profile/traits", body, uid))

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestPutMyPlayerProfileTraits_InvalidTrait(t *testing.T) {
	stub := &stubPlayerProfileStore{
		GetPlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.PlayerProfile, error) {
			return sampleProfile(1, 1), nil
		},
	}
	cfg := &Config{PlayerProfileDB: stub}
	body := map[string]interface{}{"traits": []string{"NOT_A_REAL_TRAIT"}}
	rec := httptest.NewRecorder()
	cfg.putPlayerProfileTraits(rec, playerProfileTestRequest(http.MethodPut, "/player-profile/traits", body, 1))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var errResp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&errResp))
	assert.Contains(t, errResp["error"], "Invalid trait")
	assert.Contains(t, errResp["error"], "NOT_A_REAL_TRAIT")
}

func TestPostMyPlayerProfile_InvalidCountry(t *testing.T) {
	cfg := &Config{PlayerProfileDB: &stubPlayerProfileStore{}}
	body := map[string]interface{}{"country": "USA", "position": "MID"}
	rec := httptest.NewRecorder()
	cfg.postPlayerProfile(rec, playerProfileTestRequest(http.MethodPost, "/player-profile", body, 1))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetMyPlayerProfile_Unauthorized(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/player-profile", nil)
	rec := httptest.NewRecorder()
	(&Config{PlayerProfileDB: &stubPlayerProfileStore{}}).getPlayerProfile(rec, r)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
