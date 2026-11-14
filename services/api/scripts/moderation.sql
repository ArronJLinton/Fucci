-- Helper queries for Guideline 1.2 moderation within 24 hours.
-- Soft-deactivate offending users; keep the row so re-auth shows ACCOUNT_INACTIVE.
-- AI soft-flags also create pending content_reports (reporter = system user; description starts with "AI soft-flag").
-- Content remains published until a human resolves the report.

-- 1) Pending reports (oldest first) — includes user reports and AI soft-flags
-- SELECT id, reporter_id, reportable_type, reportable_id, reported_user_id, reason, description, status, created_at
-- FROM content_reports
-- WHERE status = 'pending'
-- ORDER BY created_at ASC
-- LIMIT 50;

-- 2) Mark a report resolved after action
-- UPDATE content_reports SET status = 'resolved' WHERE id = '<report-uuid>';

-- 3) Soft-deactivate the offending user (keeps email/google/apple identity)
-- UPDATE users SET is_active = FALSE, updated_at = CURRENT_TIMESTAMP WHERE id = <reported_user_id>;

-- 4) Hide a reported fan story
-- UPDATE match_stories SET is_active = FALSE WHERE id = '<story-uuid>';

-- 5) Remove a reported debate comment
-- DELETE FROM comments WHERE id = <comment_id>;

-- 6) Clear a reported avatar
-- UPDATE users SET avatar_url = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = <user_id>;

-- Example one-shot resolve (replace placeholders):
-- BEGIN;
-- UPDATE match_stories SET is_active = FALSE WHERE id = '<story-uuid>';
-- UPDATE users SET is_active = FALSE, updated_at = CURRENT_TIMESTAMP WHERE id = <reported_user_id>;
-- UPDATE content_reports SET status = 'resolved' WHERE id = '<report-uuid>';
-- COMMIT;
