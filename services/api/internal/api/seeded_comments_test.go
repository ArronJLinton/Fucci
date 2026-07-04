package api

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/ai"
	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeededCommentContentsFromPrompt_PrefersCommentsThenCards(t *testing.T) {
	t.Parallel()
	prompt := &ai.DebatePrompt{
		Comments: []string{" agree take ", "disagree take", "wildcard take"},
		Cards: []ai.DebateCard{
			{Stance: "agree", Title: "Agree title"},
			{Stance: "disagree", Title: "Disagree title"},
		},
	}
	got := seededCommentContentsFromPrompt(prompt)
	require.Len(t, got, 3)
	assert.Equal(t, "agree take", got[0])
	assert.Equal(t, "disagree take", got[1])
	assert.Equal(t, "wildcard take", got[2])
}

func TestSeededCommentContentsFromPrompt_FallsBackToCards(t *testing.T) {
	t.Parallel()
	prompt := &ai.DebatePrompt{
		Cards: []ai.DebateCard{
			{Stance: "agree", Title: "Yes", Description: "Agree desc"},
			{Stance: "disagree", Title: "No", Description: "Disagree desc"},
			{Stance: "wildcard", Title: "Hot take"},
		},
	}
	got := seededCommentContentsFromPrompt(prompt)
	require.Len(t, got, 3)
	assert.Equal(t, "Agree desc", got[0])
	assert.Equal(t, "Disagree desc", got[1])
	assert.Equal(t, "Hot take", got[2])
}

func TestSeededCommentContentsFromDebateCards(t *testing.T) {
	t.Parallel()
	cards := []database.DebateCards{
		{Stance: "wildcard", Title: "Wildcard card"},
		{Stance: "agree", Title: "Agree card", Description: sql.NullString{String: "Pro side", Valid: true}},
		{Stance: "disagree", Title: "Disagree card", Description: sql.NullString{String: "Con side", Valid: true}},
	}
	got := seededCommentContentsFromDebateCards(cards)
	require.Len(t, got, 3)
	assert.Equal(t, "Wildcard card", got[0])
	assert.Equal(t, "Pro side", got[1])
	assert.Equal(t, "Con side", got[2])
}

func systemUserMockRows(userID int32, email string) *sqlmock.Rows {
	now := time.Unix(1700, 0).UTC()
	return sqlmock.NewRows([]string{
		"id", "firstname", "lastname", "email", "created_at", "updated_at", "is_admin",
		"display_name", "avatar_url", "google_id", "auth_provider", "locale", "last_login_at",
		"is_verified", "is_active", "role",
	}).AddRow(
		userID, "Fucci", "System", email, now, now, false,
		"Fucci", nil, nil, "local", nil, nil, true, true, "user",
	)
}
func TestGetSystemUserID_FallsBackToDefaultEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT .+ FROM users WHERE email`).
		WithArgs("wrong@example.com").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT .+ FROM users WHERE email`).
		WithArgs(defaultSystemUserEmail).
		WillReturnRows(systemUserMockRows(99, defaultSystemUserEmail))

	cfg := &Config{
		DB:              database.New(db),
		SystemUserEmail: "wrong@example.com",
	}
	id, err := cfg.getSystemUserID(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int32(99), id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSystemUserID_ProvisionsWhenMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT .+ FROM users WHERE email`).
		WithArgs(defaultSystemUserEmail).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT .+ FROM users WHERE email`).
		WithArgs(legacySystemUserEmail).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs("Fucci", "System", defaultSystemUserEmail, false).
		WillReturnRows(systemUserMockRows(42, defaultSystemUserEmail))

	cfg := &Config{DB: database.New(db)}
	id, err := cfg.getSystemUserID(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int32(42), id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEnsureSeededComments_InsertsWhenMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	debateID := int32(12)
	debateNull := sql.NullInt32{Int32: debateID, Valid: true}

	mock.ExpectQuery(`SELECT COUNT\(\*\)::bigint FROM comments`).
		WithArgs(debateNull).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	mock.ExpectQuery(`SELECT .+ FROM users WHERE email`).
		WithArgs(defaultSystemUserEmail).
		WillReturnRows(systemUserMockRows(1, defaultSystemUserEmail))

	for i := 1; i <= 3; i++ {
		mock.ExpectQuery(`INSERT INTO comments`).
			WithArgs(debateNull, sql.NullInt32{Valid: false}, sql.NullInt32{Int32: 1, Valid: true}, sqlmock.AnyArg(), true).
			WillReturnRows(sqlmock.NewRows([]string{"id", "debate_id", "parent_comment_id", "user_id", "content", "created_at", "updated_at", "seeded"}).
				AddRow(int32(i), debateID, nil, 1, "content", nil, nil, true))
	}

	cfg := &Config{
		DB:              database.New(db),
		SystemUserEmail: defaultSystemUserEmail,
	}
	cfg.ensureSeededComments(context.Background(), debateID, &ai.DebatePrompt{
		Comments: []string{"one", "two", "three"},
	})
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEnsureSeededComments_SkipsWhenAlreadyPresent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	debateID := int32(7)
	debateNull := sql.NullInt32{Int32: debateID, Valid: true}

	mock.ExpectQuery(`SELECT COUNT\(\*\)::bigint FROM comments`).
		WithArgs(debateNull).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(3)))

	cfg := &Config{DB: database.New(db)}
	cfg.ensureSeededComments(context.Background(), debateID, nil)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEnsureSeededComments_BackfillsFromCardsWhenPromptNil(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	debateID := int32(5)
	debateNull := sql.NullInt32{Int32: debateID, Valid: true}

	mock.ExpectQuery(`SELECT COUNT\(\*\)::bigint FROM comments`).
		WithArgs(debateNull).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	mock.ExpectQuery(`SELECT .+ FROM users WHERE email`).
		WithArgs(defaultSystemUserEmail).
		WillReturnRows(systemUserMockRows(1, defaultSystemUserEmail))
	mock.ExpectQuery(`SELECT .+ FROM debate_cards WHERE debate_id`).
		WithArgs(debateNull).
		WillReturnRows(sqlmock.NewRows([]string{"id", "debate_id", "stance", "title", "description", "ai_generated", "created_at", "updated_at"}).
			AddRow(1, debateID, "agree", "Agree", "Pro", true, nil, nil).
			AddRow(2, debateID, "disagree", "Disagree", "Con", true, nil, nil).
			AddRow(3, debateID, "wildcard", "Wildcard", "", true, nil, nil))

	for _, content := range []string{"Wildcard", "Pro", "Con"} {
		mock.ExpectQuery(`INSERT INTO comments`).
			WithArgs(debateNull, sql.NullInt32{Valid: false}, sql.NullInt32{Int32: 1, Valid: true}, content, true).
			WillReturnRows(sqlmock.NewRows([]string{"id", "debate_id", "parent_comment_id", "user_id", "content", "created_at", "updated_at", "seeded"}).
				AddRow(1, debateID, nil, 1, content, nil, nil, true))
	}

	cfg := &Config{
		DB:              database.New(db),
		SystemUserEmail: defaultSystemUserEmail,
	}
	cfg.ensureSeededComments(context.Background(), debateID, nil)
	require.NoError(t, mock.ExpectationsWereMet())
}
