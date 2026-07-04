package api

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"

	"github.com/ArronJLinton/fucci-api/internal/ai"
	"github.com/ArronJLinton/fucci-api/internal/database"
)

const (
	defaultSystemUserEmail  = "contact@magistri.dev"
	legacySystemUserEmail   = "fucci@system.local"
	seededCommentsPerDebate = 3
)

// getSystemUserID returns the system user (Fucci) ID for seeded comments.
// Tries Config.SystemUserEmail first, then known migration defaults.
// If none exist, provisions contact@magistri.dev (same as goose migration 20260215000003).
func (c *Config) getSystemUserID(ctx context.Context) (int32, error) {
	var candidates []string
	if email := strings.TrimSpace(c.SystemUserEmail); email != "" {
		candidates = append(candidates, email)
	}
	candidates = append(candidates, defaultSystemUserEmail, legacySystemUserEmail)

	seen := make(map[string]struct{}, len(candidates))
	var lastErr error
	for _, email := range candidates {
		if _, ok := seen[email]; ok {
			continue
		}
		seen[email] = struct{}{}
		user, err := c.DB.GetUserByEmail(ctx, email)
		if err == nil {
			return user.ID, nil
		}
		lastErr = err
	}
	if c.DB == nil {
		return 0, lastErr
	}
	user, err := c.DB.CreateUser(ctx, database.CreateUserParams{
		Firstname: "Fucci",
		Lastname:  "System",
		Email:     defaultSystemUserEmail,
		IsAdmin:   false,
	})
	if err != nil {
		if user, retryErr := c.DB.GetUserByEmail(ctx, defaultSystemUserEmail); retryErr == nil {
			return user.ID, nil
		}
		return 0, errors.Join(lastErr, err)
	}
	log.Printf("[debate] provisioned system user id=%d email=%s", user.ID, defaultSystemUserEmail)
	return user.ID, nil
}

func cardText(title, description string) string {
	if t := strings.TrimSpace(description); t != "" {
		return t
	}
	return strings.TrimSpace(title)
}

func seededCommentContentsFromPrompt(prompt *ai.DebatePrompt) []string {
	if prompt == nil {
		return nil
	}
	var contents []string
	for _, s := range prompt.Comments {
		if t := strings.TrimSpace(s); t != "" && len(contents) < seededCommentsPerDebate {
			contents = append(contents, t)
		}
	}
	for _, card := range prompt.Cards {
		if len(contents) >= seededCommentsPerDebate {
			break
		}
		if card.Stance != "agree" && card.Stance != "disagree" && card.Stance != "wildcard" {
			continue
		}
		if t := cardText(card.Title, card.Description); t != "" {
			contents = appendUniqueCommentContent(contents, t)
		}
	}
	return capCommentContents(contents)
}

func seededCommentContentsFromDebateCards(cards []database.DebateCards) []string {
	var contents []string
	for _, card := range cards {
		if card.Stance == "wildcard" {
			if t := cardText(card.Title, card.Description.String); t != "" {
				contents = appendUniqueCommentContent(contents, t)
			}
		}
	}
	for _, card := range cards {
		if len(contents) >= seededCommentsPerDebate {
			break
		}
		if card.Stance != "agree" && card.Stance != "disagree" {
			continue
		}
		if t := cardText(card.Title, card.Description.String); t != "" {
			contents = appendUniqueCommentContent(contents, t)
		}
	}
	return capCommentContents(contents)
}

func appendUniqueCommentContent(contents []string, text string) []string {
	for _, existing := range contents {
		if existing == text {
			return contents
		}
	}
	return append(contents, text)
}

func capCommentContents(contents []string) []string {
	if len(contents) > seededCommentsPerDebate {
		return contents[:seededCommentsPerDebate]
	}
	return contents
}

func existingSeededContents(rows []database.GetCommentsRow) map[string]struct{} {
	out := make(map[string]struct{})
	for _, row := range rows {
		if !row.Seeded || row.ParentCommentID.Valid {
			continue
		}
		if t := strings.TrimSpace(row.Content); t != "" {
			out[t] = struct{}{}
		}
	}
	return out
}

// ensureSeededComments inserts up to three top-level seeded comments when a debate has fewer than three.
// When prompt is nil (e.g. legacy debates), card titles/descriptions from the DB are used as fallback content.
func (c *Config) ensureSeededComments(ctx context.Context, debateID int32, prompt *ai.DebatePrompt) {
	if c.DB == nil {
		return
	}
	unlock := c.lockSeededComments(ctx, debateID)
	defer unlock()

	count, err := c.DB.CountSeededComments(ctx, sql.NullInt32{Int32: debateID, Valid: true})
	if err != nil {
		log.Printf("[debate] CountSeededComments debate=%d: %v", debateID, err)
		return
	}
	if count >= seededCommentsPerDebate {
		return
	}

	systemUserID, err := c.getSystemUserID(ctx)
	if err != nil {
		log.Printf("[debate] seeded comments skipped: system user not found (set SYSTEM_USER_EMAIL to your system user email, e.g. %s): %v", defaultSystemUserEmail, err)
		return
	}

	contents := seededCommentContentsFromPrompt(prompt)
	if len(contents) < seededCommentsPerDebate {
		cards, cardErr := c.DB.GetDebateCards(ctx, sql.NullInt32{Int32: debateID, Valid: true})
		if cardErr != nil {
			log.Printf("[debate] GetDebateCards debate=%d: %v", debateID, cardErr)
		} else {
			for _, t := range seededCommentContentsFromDebateCards(cards) {
				if len(contents) >= seededCommentsPerDebate {
					break
				}
				contents = appendUniqueCommentContent(contents, t)
			}
		}
	}
	if len(contents) == 0 {
		return
	}

	have := map[string]struct{}{}
	rows, rowErr := c.DB.GetComments(ctx, sql.NullInt32{Int32: debateID, Valid: true})
	if rowErr != nil {
		log.Printf("[debate] GetComments debate=%d: %v", debateID, rowErr)
	} else {
		have = existingSeededContents(rows)
	}

	inserted := int64(0)
	for _, content := range contents {
		if count+inserted >= seededCommentsPerDebate {
			break
		}
		if _, exists := have[content]; exists {
			continue
		}
		_, err := c.DB.CreateComment(ctx, database.CreateCommentParams{
			DebateID:        sql.NullInt32{Int32: debateID, Valid: true},
			ParentCommentID: sql.NullInt32{Valid: false},
			UserID:          sql.NullInt32{Int32: systemUserID, Valid: true},
			Content:         content,
			Seeded:          true,
		})
		if err != nil {
			log.Printf("[debate] ensureSeededComments debate=%d: %v", debateID, err)
			continue
		}
		have[content] = struct{}{}
		inserted++
	}
}

func (c *Config) lockSeededComments(ctx context.Context, debateID int32) func() {
	if c.DBConn == nil {
		return func() {}
	}
	conn, err := c.DBConn.Conn(ctx)
	if err != nil {
		log.Printf("[debate] seeded comments lock conn debate=%d: %v", debateID, err)
		return func() {}
	}
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", int64(debateID)); err != nil {
		log.Printf("[debate] seeded comments lock acquire debate=%d: %v", debateID, err)
		_ = conn.Close()
		return func() {}
	}
	return func() {
		if _, err := conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", int64(debateID)); err != nil {
			log.Printf("[debate] seeded comments lock release debate=%d: %v", debateID, err)
		}
		_ = conn.Close()
	}
}

// insertSeededComments inserts seeded comments from an AI prompt after debate creation.
func (c *Config) insertSeededComments(ctx context.Context, debateID int32, prompt *ai.DebatePrompt) {
	c.ensureSeededComments(ctx, debateID, prompt)
}
