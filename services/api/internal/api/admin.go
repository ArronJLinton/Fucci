package api

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/ArronJLinton/fucci-api/internal/database"
)

const (
	errCodeAuthRequired    = "AUTH_REQUIRED"
	errCodeAdminDBMissing  = "ADMIN_DB_NOT_CONFIGURED"
	errCodeAdminVerifyFail = "ADMIN_VERIFY_FAILED"
	errCodeAdminForbidden  = "ADMIN_REQUIRED"
)

// requireAdmin ensures the request has an authenticated user who is an admin in the DB
// (is_admin or role=admin). Writes 401/403/500 and returns false on failure.
func (c *Config) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	return c.requireAdminWithCodes(w, r, errCodeAuthRequired, errCodeAdminDBMissing, errCodeAdminVerifyFail, errCodeAdminForbidden)
}

func (c *Config) requireAdminWithCodes(
	w http.ResponseWriter,
	r *http.Request,
	authRequiredCode, dbMissingCode, verifyFailCode, forbiddenCode string,
) bool {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == 0 {
		respondWithErrorCode(w, http.StatusUnauthorized, "Authentication required", authRequiredCode)
		return false
	}
	if c.DB == nil {
		logErrorAndRespond500(w, "verify admin", fmt.Errorf("database not configured"), dbMissingCode)
		return false
	}

	user, err := c.DB.GetUser(r.Context(), userID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithErrorCode(w, http.StatusUnauthorized, "Account not found. Please log in again.", authRequiredCode)
			return false
		}
		logErrorAndRespond500(w, "verify admin", err, verifyFailCode)
		return false
	}
	if user.IsAdmin || (user.Role.Valid && user.Role.UserRole == database.UserRoleAdmin) {
		return true
	}

	respondWithErrorCode(w, http.StatusForbidden, "Admin privileges required", forbiddenCode)
	return false
}

// requireAdminMiddleware rejects non-admin callers after RequireAuth has attached JWT claims.
func (c *Config) requireAdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !c.requireAdmin(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}
