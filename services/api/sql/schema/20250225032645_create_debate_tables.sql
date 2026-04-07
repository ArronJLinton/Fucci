-- +goose Up
-- Create debates table
CREATE TABLE IF NOT EXISTS debates (
    id SERIAL PRIMARY KEY,
    match_id VARCHAR(50) NOT NULL,
    debate_type VARCHAR(20) NOT NULL CHECK (debate_type IN ('pre_match', 'post_match')),
    headline TEXT NOT NULL,
    description TEXT,
    ai_generated BOOLEAN DEFAULT true,
    deleted_at TIMESTAMP NULL, -- Soft delete timestamp
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create debate_cards table for different stances
CREATE TABLE IF NOT EXISTS debate_cards (
    id SERIAL PRIMARY KEY,
    debate_id INTEGER REFERENCES debates(id) ON DELETE CASCADE,
    stance VARCHAR(20) NOT NULL CHECK (stance IN ('agree', 'disagree', 'wildcard')),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    ai_generated BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create votes table for fan voting
CREATE TABLE IF NOT EXISTS votes (
    id SERIAL PRIMARY KEY,
    debate_card_id INTEGER REFERENCES debate_cards(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    vote_type VARCHAR(10) NOT NULL CHECK (vote_type IN ('upvote', 'downvote', 'emoji')),
    emoji VARCHAR(10), -- For emoji votes
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(debate_card_id, user_id, vote_type, emoji)
);

-- Create comments table for text-based discussions
CREATE TABLE IF NOT EXISTS comments (
    id SERIAL PRIMARY KEY,
    debate_id INTEGER REFERENCES debates(id) ON DELETE CASCADE,
    parent_comment_id INTEGER REFERENCES comments(id) ON DELETE SET NULL, -- For nested comments
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create debate_analytics table for tracking engagement
CREATE TABLE IF NOT EXISTS debate_analytics (
    id SERIAL PRIMARY KEY,
    debate_id INTEGER REFERENCES debates(id) ON DELETE CASCADE,
    total_votes INTEGER DEFAULT 0,
    total_comments INTEGER DEFAULT 0,
    engagement_score DECIMAL(5,2) DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_debates_match_id ON debates(match_id);
CREATE INDEX IF NOT EXISTS idx_debates_deleted_at ON debates(deleted_at);
CREATE INDEX IF NOT EXISTS idx_debates_type ON debates(debate_type);
CREATE INDEX IF NOT EXISTS idx_debate_cards_debate_id ON debate_cards(debate_id);
CREATE INDEX IF NOT EXISTS idx_votes_debate_card_id ON votes(debate_card_id);
CREATE INDEX IF NOT EXISTS idx_votes_user_id ON votes(user_id);
CREATE INDEX IF NOT EXISTS idx_comments_debate_id ON comments(debate_id);
CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id);
CREATE INDEX IF NOT EXISTS idx_comments_parent_id ON comments(parent_comment_id);
CREATE INDEX IF NOT EXISTS idx_debate_analytics_debate_id ON debate_analytics(debate_id);

-- +goose Down
DROP INDEX IF EXISTS idx_debate_analytics_debate_id;
DROP INDEX IF EXISTS idx_comments_parent_id;
DROP INDEX IF EXISTS idx_comments_user_id;
DROP INDEX IF EXISTS idx_comments_debate_id;
DROP INDEX IF EXISTS idx_votes_user_id;
DROP INDEX IF EXISTS idx_votes_debate_card_id;
DROP INDEX IF EXISTS idx_debate_cards_debate_id;
DROP INDEX IF EXISTS idx_debates_type;
DROP INDEX IF EXISTS idx_debates_deleted_at;
DROP INDEX IF EXISTS idx_debates_match_id;
DROP TABLE IF EXISTS debate_analytics;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS votes;
DROP TABLE IF EXISTS debate_cards;
DROP TABLE IF EXISTS debates; 