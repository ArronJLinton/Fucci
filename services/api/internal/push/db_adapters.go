package push

import (
	"context"
	"database/sql"

	"github.com/ArronJLinton/fucci-api/internal/database"
)

// DebateQuerier is the sqlc surface for debate push selection.
type DebateQuerier interface {
	GetTopUnvotedDebateForPush(ctx context.Context, userID sql.NullInt32) (database.GetTopUnvotedDebateForPushRow, error)
}

// DebateDB adapts *database.Queries for DebatePushStore.
type DebateDB struct {
	Q DebateQuerier
}

func (d DebateDB) GetTopUnvotedDebateForPush(ctx context.Context, userID int32) (database.GetTopUnvotedDebateForPushRow, error) {
	return d.Q.GetTopUnvotedDebateForPush(ctx, sql.NullInt32{Int32: userID, Valid: true})
}
