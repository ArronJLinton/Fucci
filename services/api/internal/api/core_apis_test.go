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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockQueries is a mock implementation of the database.Queries interface
type MockQueries struct {
	mock.Mock
}

// Mock for User operations
func (m *MockQueries) GetUser(ctx context.Context, id int32) (database.Users, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(database.Users), args.Error(1)
}

func (m *MockQueries) CreateUser(ctx context.Context, arg database.CreateUserParams) (database.Users, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.Users), args.Error(1)
}

func (m *MockQueries) ListUsers(ctx context.Context) ([]database.Users, error) {
	args := m.Called(ctx)
	return args.Get(0).([]database.Users), args.Error(1)
}

// Mock for Player Profile operations
func (m *MockQueries) CreatePlayerProfile(ctx context.Context, arg database.CreatePlayerProfileParams) (database.PlayerProfiles, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.PlayerProfiles), args.Error(1)
}

func (m *MockQueries) GetPlayerProfile(ctx context.Context, id uuid.UUID) (database.PlayerProfiles, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(database.PlayerProfiles), args.Error(1)
}

func (m *MockQueries) UpdatePlayerProfile(ctx context.Context, arg database.UpdatePlayerProfileParams) (database.PlayerProfiles, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.PlayerProfiles), args.Error(1)
}

func (m *MockQueries) DeletePlayerProfile(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQueries) ListPlayerProfiles(ctx context.Context) ([]database.PlayerProfiles, error) {
	args := m.Called(ctx)
	return args.Get(0).([]database.PlayerProfiles), args.Error(1)
}

// Mock for League operations
func (m *MockQueries) CreateLeague(ctx context.Context, arg database.CreateLeagueParams) (database.Leagues, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.Leagues), args.Error(1)
}

func (m *MockQueries) GetLeague(ctx context.Context, id uuid.UUID) (database.Leagues, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(database.Leagues), args.Error(1)
}

func (m *MockQueries) ListLeagues(ctx context.Context) ([]database.Leagues, error) {
	args := m.Called(ctx)
	return args.Get(0).([]database.Leagues), args.Error(1)
}

func (m *MockQueries) UpdateLeague(ctx context.Context, arg database.UpdateLeagueParams) (database.Leagues, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.Leagues), args.Error(1)
}

func (m *MockQueries) DeleteLeague(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Mock for Team operations
func (m *MockQueries) CreateTeam(ctx context.Context, arg database.CreateTeamParams) (database.Teams, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.Teams), args.Error(1)
}

func (m *MockQueries) GetTeam(ctx context.Context, id uuid.UUID) (database.Teams, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(database.Teams), args.Error(1)
}

func (m *MockQueries) ListTeams(ctx context.Context) ([]database.Teams, error) {
	args := m.Called(ctx)
	return args.Get(0).([]database.Teams), args.Error(1)
}

func (m *MockQueries) UpdateTeam(ctx context.Context, arg database.UpdateTeamParams) (database.Teams, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.Teams), args.Error(1)
}

func (m *MockQueries) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Mock for Team Manager operations
func (m *MockQueries) CreateTeamManager(ctx context.Context, arg database.CreateTeamManagerParams) (database.TeamManagers, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.TeamManagers), args.Error(1)
}

func (m *MockQueries) GetTeamManager(ctx context.Context, id uuid.UUID) (database.TeamManagers, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(database.TeamManagers), args.Error(1)
}

func (m *MockQueries) GetTeamManagersByLeague(ctx context.Context, leagueID uuid.UUID) ([]database.TeamManagers, error) {
	args := m.Called(ctx, leagueID)
	return args.Get(0).([]database.TeamManagers), args.Error(1)
}

func (m *MockQueries) ListTeamManagers(ctx context.Context) ([]database.TeamManagers, error) {
	args := m.Called(ctx)
	return args.Get(0).([]database.TeamManagers), args.Error(1)
}

func (m *MockQueries) UpdateTeamManager(ctx context.Context, arg database.UpdateTeamManagerParams) (database.TeamManagers, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.TeamManagers), args.Error(1)
}

func (m *MockQueries) DeleteTeamManager(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Test-specific service structs that can accept the mock
type TestPlayerProfileService struct {
	DB *MockQueries
}

func (svc *TestPlayerProfileService) CreatePlayerProfile(w http.ResponseWriter, r *http.Request) {
	var req CreatePlayerProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Check if user exists first
	_, err := svc.DB.GetUser(r.Context(), req.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusBadRequest)
			return
		}
		http.Error(w, "Error checking user", http.StatusInternalServerError)
		return
	}

	// Convert to database params
	var teamID uuid.NullUUID
	if req.TeamID != nil {
		teamID = uuid.NullUUID{UUID: *req.TeamID, Valid: true}
	}

	dbParams := database.CreatePlayerProfileParams{
		UserID:    req.UserID,
		TeamID:    teamID,
		Position:  req.Position,
		Age:       req.Age,
		Country:   req.Country,
		HeightCm:  req.HeightCm,
		Pace:      req.Pace,
		Shooting:  req.Shooting,
		Passing:   req.Passing,
		Stamina:   req.Stamina,
		Dribbling: req.Dribbling,
		Defending: req.Defending,
		Physical:  req.Physical,
	}

	profile, err := svc.DB.CreatePlayerProfile(r.Context(), dbParams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(profile)
}

func (svc *TestPlayerProfileService) GetPlayerProfile(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	profile, err := svc.DB.GetPlayerProfile(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(profile)
}

// Test Player Profiles API
func TestPlayerProfileService_CreatePlayerProfile(t *testing.T) {
	mockDB := new(MockQueries)
	service := &TestPlayerProfileService{DB: mockDB}

	t.Run("successful creation", func(t *testing.T) {
		// Mock user exists
		mockUser := database.Users{
			ID:        1,
			Firstname: "John",
			Lastname:  "Doe",
			Email:     "john@example.com",
		}
		mockDB.On("GetUser", mock.Anything, int32(1)).Return(mockUser, nil)

		// Mock player profile creation
		expectedProfile := database.PlayerProfiles{
			ID:         uuid.New(),
			UserID:     1,
			Position:   "Forward",
			Age:        25,
			Country:    "USA",
			HeightCm:   180,
			Pace:       80,
			Shooting:   75,
			Passing:    70,
			Stamina:    85,
			Dribbling:  78,
			Defending:  60,
			Physical:   72,
			IsVerified: false,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		mockDB.On("CreatePlayerProfile", mock.Anything, mock.AnythingOfType("database.CreatePlayerProfileParams")).Return(expectedProfile, nil)

		// Create request
		reqBody := CreatePlayerProfileRequest{
			UserID:    1,
			Position:  "Forward",
			Age:       25,
			Country:   "USA",
			HeightCm:  180,
			Pace:      80,
			Shooting:  75,
			Passing:   70,
			Stamina:   85,
			Dribbling: 78,
			Defending: 60,
			Physical:  72,
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/player-profiles", bytes.NewBuffer(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		// Call the function
		service.CreatePlayerProfile(rec, req)

		// Assertions
		assert.Equal(t, http.StatusOK, rec.Code)

		var response database.PlayerProfiles
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, expectedProfile.ID, response.ID)
		assert.Equal(t, int32(1), response.UserID)
		assert.Equal(t, "Forward", response.Position)

		mockDB.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		mockDB.On("GetUser", mock.Anything, int32(999)).Return(database.Users{}, sql.ErrNoRows)

		reqBody := CreatePlayerProfileRequest{
			UserID:    999,
			Position:  "Forward",
			Age:       25,
			Country:   "USA",
			HeightCm:  180,
			Pace:      80,
			Shooting:  75,
			Passing:   70,
			Stamina:   85,
			Dribbling: 78,
			Defending: 60,
			Physical:  72,
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/player-profiles", bytes.NewBuffer(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		service.CreatePlayerProfile(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "User not found")

		mockDB.AssertExpectations(t)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/player-profiles", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		service.CreatePlayerProfile(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid request")
	})
}

func TestPlayerProfileService_GetPlayerProfile(t *testing.T) {
	mockDB := new(MockQueries)
	service := &TestPlayerProfileService{DB: mockDB}

	t.Run("successful retrieval", func(t *testing.T) {
		profileID := uuid.New()
		expectedProfile := database.PlayerProfiles{
			ID:         profileID,
			UserID:     1,
			Position:   "Forward",
			Age:        25,
			Country:    "USA",
			HeightCm:   180,
			Pace:       80,
			Shooting:   75,
			Passing:    70,
			Stamina:    85,
			Dribbling:  78,
			Defending:  60,
			Physical:   72,
			IsVerified: false,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		mockDB.On("GetPlayerProfile", mock.Anything, profileID).Return(expectedProfile, nil)

		req := httptest.NewRequest("GET", "/player-profiles?id="+profileID.String(), nil)
		rec := httptest.NewRecorder()

		service.GetPlayerProfile(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response database.PlayerProfiles
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, profileID, response.ID)

		mockDB.AssertExpectations(t)
	})

	t.Run("profile not found", func(t *testing.T) {
		profileID := uuid.New()
		mockDB.On("GetPlayerProfile", mock.Anything, profileID).Return(database.PlayerProfiles{}, sql.ErrNoRows)

		req := httptest.NewRequest("GET", "/player-profiles?id="+profileID.String(), nil)
		rec := httptest.NewRecorder()

		service.GetPlayerProfile(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "not found")

		mockDB.AssertExpectations(t)
	})

	t.Run("invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/player-profiles?id=invalid-uuid", nil)
		rec := httptest.NewRecorder()

		service.GetPlayerProfile(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid id")
	})
}

// Test CreatePlayerProfileRequest struct
func TestCreatePlayerProfileRequest(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := CreatePlayerProfileRequest{
			UserID:    1,
			Position:  "Forward",
			Age:       25,
			Country:   "USA",
			HeightCm:  180,
			Pace:      80,
			Shooting:  75,
			Passing:   70,
			Stamina:   85,
			Dribbling: 78,
			Defending: 60,
			Physical:  72,
		}

		assert.Equal(t, int32(1), req.UserID)
		assert.Equal(t, "Forward", req.Position)
		assert.Equal(t, int32(25), req.Age)
		assert.Equal(t, "USA", req.Country)
		assert.Equal(t, int32(180), req.HeightCm)
	})

	t.Run("JSON marshaling", func(t *testing.T) {
		req := CreatePlayerProfileRequest{
			UserID:    1,
			Position:  "Forward",
			Age:       25,
			Country:   "USA",
			HeightCm:  180,
			Pace:      80,
			Shooting:  75,
			Passing:   70,
			Stamina:   85,
			Dribbling: 78,
			Defending: 60,
			Physical:  72,
		}

		jsonData, err := json.Marshal(req)
		assert.NoError(t, err)

		var unmarshaled CreatePlayerProfileRequest
		err = json.Unmarshal(jsonData, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, req, unmarshaled)
	})
}

// Test error handling
func TestErrorHandling(t *testing.T) {
	mockDB := new(MockQueries)
	service := &TestPlayerProfileService{DB: mockDB}

	t.Run("database error during user lookup", func(t *testing.T) {
		mockDB.ExpectedCalls = nil // Clear previous expectations
		mockDB.On("GetUser", mock.Anything, int32(1)).Return(database.Users{}, assert.AnError)

		reqBody := CreatePlayerProfileRequest{
			UserID:    1,
			Position:  "Forward",
			Age:       25,
			Country:   "USA",
			HeightCm:  180,
			Pace:      80,
			Shooting:  75,
			Passing:   70,
			Stamina:   85,
			Dribbling: 78,
			Defending: 60,
			Physical:  72,
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/player-profiles", bytes.NewBuffer(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		service.CreatePlayerProfile(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Error checking user")

		mockDB.AssertExpectations(t)
	})

	t.Run("database error during profile creation", func(t *testing.T) {
		mockDB.ExpectedCalls = nil // Clear previous expectations
		mockUser := database.Users{
			ID:        1,
			Firstname: "John",
			Lastname:  "Doe",
			Email:     "john@example.com",
		}
		mockDB.On("GetUser", mock.Anything, int32(1)).Return(mockUser, nil)
		mockDB.On("CreatePlayerProfile", mock.Anything, mock.AnythingOfType("database.CreatePlayerProfileParams")).Return(database.PlayerProfiles{}, assert.AnError)

		reqBody := CreatePlayerProfileRequest{
			UserID:    1,
			Position:  "Forward",
			Age:       25,
			Country:   "USA",
			HeightCm:  180,
			Pace:      80,
			Shooting:  75,
			Passing:   70,
			Stamina:   85,
			Dribbling: 78,
			Defending: 60,
			Physical:  72,
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/player-profiles", bytes.NewBuffer(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		service.CreatePlayerProfile(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "assert.AnError")

		mockDB.AssertExpectations(t)
	})
}
