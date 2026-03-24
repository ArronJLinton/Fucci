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

// stubMePlayerStore implements MePlayerProfileStore with per-method func hooks (nil => safe default).
type stubMePlayerStore struct {
	GetMePlayerProfileByUserIDFn             func(ctx context.Context, userID int32) (database.MePlayerProfile, error)
	UpsertMePlayerProfileFn                  func(ctx context.Context, arg database.UpsertMePlayerProfileParams) (database.MePlayerProfile, error)
	UpdateMePlayerProfileFn                  func(ctx context.Context, arg database.UpdateMePlayerProfileParams) (database.MePlayerProfile, error)
	DeleteMePlayerProfileFn                  func(ctx context.Context, id int32) error
	ListMePlayerProfileTraitsFn              func(ctx context.Context, mePlayerProfileID int32) ([]string, error)
	ListMePlayerProfileCareerTeamsFn         func(ctx context.Context, mePlayerProfileID int32) ([]database.MePlayerProfileCareerTeam, error)
	DeleteMePlayerProfileTraitsByProfileIDFn func(ctx context.Context, mePlayerProfileID int32) error
	InsertMePlayerProfileTraitFn             func(ctx context.Context, arg database.InsertMePlayerProfileTraitParams) (database.MePlayerProfileTrait, error)
}

func (s *stubMePlayerStore) GetMePlayerProfileByUserID(ctx context.Context, userID int32) (database.MePlayerProfile, error) {
	if s.GetMePlayerProfileByUserIDFn != nil {
		return s.GetMePlayerProfileByUserIDFn(ctx, userID)
	}
	return database.MePlayerProfile{}, sql.ErrNoRows
}

func (s *stubMePlayerStore) UpsertMePlayerProfile(ctx context.Context, arg database.UpsertMePlayerProfileParams) (database.MePlayerProfile, error) {
	if s.UpsertMePlayerProfileFn != nil {
		return s.UpsertMePlayerProfileFn(ctx, arg)
	}
	return database.MePlayerProfile{}, assert.AnError
}

func (s *stubMePlayerStore) UpdateMePlayerProfile(ctx context.Context, arg database.UpdateMePlayerProfileParams) (database.MePlayerProfile, error) {
	if s.UpdateMePlayerProfileFn != nil {
		return s.UpdateMePlayerProfileFn(ctx, arg)
	}
	return database.MePlayerProfile{}, assert.AnError
}

func (s *stubMePlayerStore) DeleteMePlayerProfile(ctx context.Context, id int32) error {
	if s.DeleteMePlayerProfileFn != nil {
		return s.DeleteMePlayerProfileFn(ctx, id)
	}
	return assert.AnError
}

func (s *stubMePlayerStore) ListMePlayerProfileTraits(ctx context.Context, mePlayerProfileID int32) ([]string, error) {
	if s.ListMePlayerProfileTraitsFn != nil {
		return s.ListMePlayerProfileTraitsFn(ctx, mePlayerProfileID)
	}
	return nil, nil
}

func (s *stubMePlayerStore) ListMePlayerProfileCareerTeams(ctx context.Context, mePlayerProfileID int32) ([]database.MePlayerProfileCareerTeam, error) {
	if s.ListMePlayerProfileCareerTeamsFn != nil {
		return s.ListMePlayerProfileCareerTeamsFn(ctx, mePlayerProfileID)
	}
	return nil, nil
}

func (s *stubMePlayerStore) DeleteMePlayerProfileTraitsByProfileID(ctx context.Context, mePlayerProfileID int32) error {
	if s.DeleteMePlayerProfileTraitsByProfileIDFn != nil {
		return s.DeleteMePlayerProfileTraitsByProfileIDFn(ctx, mePlayerProfileID)
	}
	return nil
}

func (s *stubMePlayerStore) InsertMePlayerProfileTrait(ctx context.Context, arg database.InsertMePlayerProfileTraitParams) (database.MePlayerProfileTrait, error) {
	if s.InsertMePlayerProfileTraitFn != nil {
		return s.InsertMePlayerProfileTraitFn(ctx, arg)
	}
	return database.MePlayerProfileTrait{}, assert.AnError
}

func mePlayerTestRequest(method, path string, body interface{}, userID int32) *http.Request {
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

func sampleProfile(userID int32, id int32) database.MePlayerProfile {
	return database.MePlayerProfile{
		ID:          id,
		UserID:      userID,
		Age:         sql.NullInt32{Int32: 22, Valid: true},
		CountryCode: "US",
		ClubName:    sql.NullString{String: "Test FC", Valid: true},
		IsFreeAgent: false,
		Position:    "MID",
		CreatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
	}
}

func TestGetMePlayerProfile_NotFound(t *testing.T) {
	uid := int32(42)
	stub := &stubMePlayerStore{
		GetMePlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.MePlayerProfile, error) {
			assert.Equal(t, uid, userID)
			return database.MePlayerProfile{}, sql.ErrNoRows
		},
	}
	cfg := &Config{MePlayerProfileDB: stub}
	rec := httptest.NewRecorder()
	cfg.getMePlayerProfile(rec, mePlayerTestRequest(http.MethodGet, "/me/player-profile", nil, uid))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	var out map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Contains(t, out["error"], "not found")
}

func TestGetMePlayerProfile_OK(t *testing.T) {
	uid := int32(7)
	p := sampleProfile(uid, 100)
	stub := &stubMePlayerStore{
		GetMePlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.MePlayerProfile, error) {
			return p, nil
		},
		ListMePlayerProfileTraitsFn: func(ctx context.Context, pid int32) ([]string, error) {
			assert.Equal(t, p.ID, pid)
			return []string{"FLAIR", "PLAYMAKER"}, nil
		},
		ListMePlayerProfileCareerTeamsFn: func(ctx context.Context, pid int32) ([]database.MePlayerProfileCareerTeam, error) {
			return []database.MePlayerProfileCareerTeam{
				{ID: 1, MePlayerProfileID: pid, TeamName: "Old Club", StartYear: 2020, EndYear: sql.NullInt32{Int32: 2023, Valid: true}},
			}, nil
		},
	}
	cfg := &Config{MePlayerProfileDB: stub}
	rec := httptest.NewRecorder()
	cfg.getMePlayerProfile(rec, mePlayerTestRequest(http.MethodGet, "/me/player-profile", nil, uid))

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp MePlayerProfileResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, p.ID, resp.ID)
	assert.Equal(t, "US", resp.Country)
	assert.Equal(t, "MID", resp.Position)
	assert.Equal(t, []string{"FLAIR", "PLAYMAKER"}, resp.Traits)
	require.Len(t, resp.CareerTeams, 1)
	assert.Equal(t, "Old Club", resp.CareerTeams[0].TeamName)
}

func TestPostMePlayerProfile_Create(t *testing.T) {
	uid := int32(5)
	var upsertArg database.UpsertMePlayerProfileParams
	stub := &stubMePlayerStore{
		UpsertMePlayerProfileFn: func(ctx context.Context, arg database.UpsertMePlayerProfileParams) (database.MePlayerProfile, error) {
			upsertArg = arg
			return database.MePlayerProfile{
				ID: 1, UserID: arg.UserID, CountryCode: arg.CountryCode, Position: arg.Position,
				IsFreeAgent: arg.IsFreeAgent, Age: arg.Age, ClubName: arg.ClubName,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}
	cfg := &Config{MePlayerProfileDB: stub}
	body := map[string]interface{}{"country": "gb", "position": "DEF", "age": 18}
	rec := httptest.NewRecorder()
	cfg.postMePlayerProfile(rec, mePlayerTestRequest(http.MethodPost, "/me/player-profile", body, uid))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, uid, upsertArg.UserID)
	assert.Equal(t, "GB", upsertArg.CountryCode)
	assert.Equal(t, "DEF", upsertArg.Position)
	var resp MePlayerProfileResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "GB", resp.Country)
	assert.Equal(t, int32(1), resp.ID)
}

func TestPostMePlayerProfile_UpdateExisting(t *testing.T) {
	uid := int32(3)
	existing := sampleProfile(uid, 99)
	updated := existing
	updated.CountryCode = "DE"
	updated.Position = "FWD"
	var upsertArg database.UpsertMePlayerProfileParams

	stub := &stubMePlayerStore{
		UpsertMePlayerProfileFn: func(ctx context.Context, arg database.UpsertMePlayerProfileParams) (database.MePlayerProfile, error) {
			upsertArg = arg
			return updated, nil
		},
		ListMePlayerProfileTraitsFn: func(ctx context.Context, pid int32) ([]string, error) {
			assert.Equal(t, updated.ID, pid)
			return []string{}, nil
		},
		ListMePlayerProfileCareerTeamsFn: func(ctx context.Context, pid int32) ([]database.MePlayerProfileCareerTeam, error) {
			return nil, nil
		},
	}
	cfg := &Config{MePlayerProfileDB: stub}
	body := map[string]interface{}{"country": "DE", "position": "FWD"}
	rec := httptest.NewRecorder()
	cfg.postMePlayerProfile(rec, mePlayerTestRequest(http.MethodPost, "/me/player-profile", body, uid))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, uid, upsertArg.UserID)
	assert.Equal(t, "DE", upsertArg.CountryCode)
	assert.Equal(t, "FWD", upsertArg.Position)
	var resp MePlayerProfileResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "DE", resp.Country)
	assert.Equal(t, "FWD", resp.Position)
}

func TestPutMePlayerProfile_NotFound(t *testing.T) {
	stub := &stubMePlayerStore{
		GetMePlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.MePlayerProfile, error) {
			return database.MePlayerProfile{}, sql.ErrNoRows
		},
	}
	cfg := &Config{MePlayerProfileDB: stub}
	body := map[string]interface{}{"country": "FR", "position": "GK"}
	rec := httptest.NewRecorder()
	cfg.putMePlayerProfile(rec, mePlayerTestRequest(http.MethodPut, "/me/player-profile", body, 1))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPutMePlayerProfile_OK(t *testing.T) {
	uid := int32(2)
	p := sampleProfile(uid, 55)
	updated := p
	updated.CountryCode = "CA"
	updated.Position = "GK"
	var sawUpdate bool
	stub := &stubMePlayerStore{
		GetMePlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.MePlayerProfile, error) {
			return p, nil
		},
		UpdateMePlayerProfileFn: func(ctx context.Context, arg database.UpdateMePlayerProfileParams) (database.MePlayerProfile, error) {
			sawUpdate = true
			assert.Equal(t, p.ID, arg.ID)
			assert.Equal(t, "CA", arg.CountryCode)
			assert.Equal(t, "GK", arg.Position)
			return updated, nil
		},
		ListMePlayerProfileTraitsFn: func(ctx context.Context, pid int32) ([]string, error) {
			return []string{"LEADERSHIP"}, nil
		},
		ListMePlayerProfileCareerTeamsFn: func(ctx context.Context, pid int32) ([]database.MePlayerProfileCareerTeam, error) {
			return nil, nil
		},
	}
	cfg := &Config{MePlayerProfileDB: stub}
	body := map[string]interface{}{"country": "ca", "position": "GK"}
	rec := httptest.NewRecorder()
	cfg.putMePlayerProfile(rec, mePlayerTestRequest(http.MethodPut, "/me/player-profile", body, uid))

	assert.True(t, sawUpdate)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp MePlayerProfileResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "CA", resp.Country)
	assert.Equal(t, []string{"LEADERSHIP"}, resp.Traits)
}

func TestDeleteMePlayerProfile_OK(t *testing.T) {
	uid := int32(8)
	p := sampleProfile(uid, 200)
	var deletedID int32
	stub := &stubMePlayerStore{
		GetMePlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.MePlayerProfile, error) {
			return p, nil
		},
		DeleteMePlayerProfileFn: func(ctx context.Context, id int32) error {
			deletedID = id
			return nil
		},
	}
	cfg := &Config{MePlayerProfileDB: stub}
	rec := httptest.NewRecorder()
	cfg.deleteMePlayerProfile(rec, mePlayerTestRequest(http.MethodDelete, "/me/player-profile", nil, uid))

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, p.ID, deletedID)
}

func TestDeleteMePlayerProfile_NotFound(t *testing.T) {
	stub := &stubMePlayerStore{
		GetMePlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.MePlayerProfile, error) {
			return database.MePlayerProfile{}, sql.ErrNoRows
		},
	}
	cfg := &Config{MePlayerProfileDB: stub}
	rec := httptest.NewRecorder()
	cfg.deleteMePlayerProfile(rec, mePlayerTestRequest(http.MethodDelete, "/me/player-profile", nil, 1))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPutMePlayerProfileTraits_Replacement(t *testing.T) {
	uid := int32(1)
	p := sampleProfile(uid, 30)
	var deleteCalls, insertOrder []string
	traitsState := []string{"OLD_TRAIT"}

	stub := &stubMePlayerStore{
		GetMePlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.MePlayerProfile, error) {
			return p, nil
		},
		DeleteMePlayerProfileTraitsByProfileIDFn: func(ctx context.Context, mePlayerProfileID int32) error {
			deleteCalls = append(deleteCalls, "ok")
			traitsState = nil
			return nil
		},
		InsertMePlayerProfileTraitFn: func(ctx context.Context, arg database.InsertMePlayerProfileTraitParams) (database.MePlayerProfileTrait, error) {
			insertOrder = append(insertOrder, arg.TraitCode)
			traitsState = append(traitsState, arg.TraitCode)
			return database.MePlayerProfileTrait{ID: int32(len(insertOrder)), MePlayerProfileID: arg.MePlayerProfileID, TraitCode: arg.TraitCode}, nil
		},
		ListMePlayerProfileTraitsFn: func(ctx context.Context, mePlayerProfileID int32) ([]string, error) {
			out := make([]string, len(traitsState))
			copy(out, traitsState)
			return out, nil
		},
	}
	cfg := &Config{MePlayerProfileDB: stub}
	body := map[string]interface{}{"traits": []string{"FLAIR", "PLAYMAKER", "SPEED_DRIBBLER"}}
	rec := httptest.NewRecorder()
	cfg.putMePlayerProfileTraits(rec, mePlayerTestRequest(http.MethodPut, "/me/player-profile/traits", body, uid))

	assert.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, deleteCalls, 1)
	assert.Equal(t, []string{"FLAIR", "PLAYMAKER", "SPEED_DRIBBLER"}, insertOrder)
	var out map[string][]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&out))
	assert.Equal(t, []string{"FLAIR", "PLAYMAKER", "SPEED_DRIBBLER"}, out["traits"])
}

func TestPutMePlayerProfileTraits_MaxFive(t *testing.T) {
	stub := &stubMePlayerStore{
		GetMePlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.MePlayerProfile, error) {
			return sampleProfile(1, 1), nil
		},
	}
	cfg := &Config{MePlayerProfileDB: stub}
	body := map[string]interface{}{
		"traits": []string{"LEADERSHIP", "FINESSE_SHOT", "PLAYMAKER", "SPEED_DRIBBLER", "LONG_SHOT_TAKER", "FLAIR"},
	}
	rec := httptest.NewRecorder()
	cfg.putMePlayerProfileTraits(rec, mePlayerTestRequest(http.MethodPut, "/me/player-profile/traits", body, 1))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var errResp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&errResp))
	assert.Contains(t, errResp["error"], "5 traits")
}

func TestPutMePlayerProfileTraits_InvalidTrait(t *testing.T) {
	stub := &stubMePlayerStore{
		GetMePlayerProfileByUserIDFn: func(ctx context.Context, userID int32) (database.MePlayerProfile, error) {
			return sampleProfile(1, 1), nil
		},
	}
	cfg := &Config{MePlayerProfileDB: stub}
	body := map[string]interface{}{"traits": []string{"NOT_A_REAL_TRAIT"}}
	rec := httptest.NewRecorder()
	cfg.putMePlayerProfileTraits(rec, mePlayerTestRequest(http.MethodPut, "/me/player-profile/traits", body, 1))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var errResp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&errResp))
	assert.Contains(t, errResp["error"], "Invalid trait")
	assert.Contains(t, errResp["error"], "NOT_A_REAL_TRAIT")
}

func TestPostMePlayerProfile_InvalidCountry(t *testing.T) {
	cfg := &Config{MePlayerProfileDB: &stubMePlayerStore{}}
	body := map[string]interface{}{"country": "USA", "position": "MID"}
	rec := httptest.NewRecorder()
	cfg.postMePlayerProfile(rec, mePlayerTestRequest(http.MethodPost, "/me/player-profile", body, 1))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetMePlayerProfile_Unauthorized(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/me/player-profile", nil)
	rec := httptest.NewRecorder()
	(&Config{MePlayerProfileDB: &stubMePlayerStore{}}).getMePlayerProfile(rec, r)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
