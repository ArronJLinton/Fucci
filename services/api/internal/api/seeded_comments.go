package api

import (
	"context"
	"database/sql"
	"log"
	"strings"

	"github.com/ArronJLinton/fucci-api/internal/ai"
	"github.com/ArronJLinton/fucci-api/internal/database"
)

const (
	defaultSystemUserEmail = "contact@magistri.dev"
	legacySystemUserEmail  = "fucci@system.local"
	seededCommentsPerDebate = 3
)

// getSystemUserID returns the system user (Fucci) ID for seeded comments.
// Tries Config.SystemUserEmail first, then known migration defaults.
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
	return 0, lastErr
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

	var have map[string]struct{}
	if count > 0 {
		rows, rowErr := c.DB.GetComments(ctx, sql.NullInt32{Int32: debateID, Valid: true})
		if rowErr != nil {
			log.Printf("[debate] GetComments debate=%d: %v", debateID, rowErr)
		} else {
			have = existingSeededContents(rows)
		}
	}

	inserted := int64(0)
	for _, content := range contents {
		if count+inserted >= seededCommentsPerDebate {
			break
		}
		if have != nil {
			if _, exists := have[content]; exists {
				continue
			}
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
		inserted++
	}
}

// insertSeededComments inserts seeded comments from an AI prompt after debate creation.
func (c *Config) insertSeededComments(ctx context.Context, debateID int32, prompt *ai.DebatePrompt) {
	c.ensureSeededComments(ctx, debateID, prompt)
}
