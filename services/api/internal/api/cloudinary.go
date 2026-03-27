package api

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const cloudinaryMaxUploadBytes = 5 * 1024 * 1024 // 5 MB

type cloudinarySignatureRequest struct {
	Context string `json:"context"`
}

type cloudinarySignatureResponse struct {
	CloudName      string `json:"cloud_name"`
	APIKey         string `json:"api_key"`
	Timestamp      int64  `json:"timestamp"`
	Signature      string `json:"signature"`
	Folder         string `json:"folder"`
	PublicID       string `json:"public_id,omitempty"`
	UploadPreset   string `json:"upload_preset,omitempty"`
	MaxUploadBytes int64  `json:"max_upload_bytes"`
}

func cloudinaryFolderForContext(context string) (string, bool) {
	switch context {
	case "avatar":
		return "fucci/avatars", true
	case "player_profile":
		return "fucci/player-profiles", true
	default:
		return "", false
	}
}

func cloudinarySign(params map[string]string, apiSecret string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := params[k]
		if strings.TrimSpace(v) == "" {
			continue
		}
		parts = append(parts, k+"="+v)
	}
	signingPayload := strings.Join(parts, "&") + apiSecret
	sum := sha1.Sum([]byte(signingPayload))
	return hex.EncodeToString(sum[:])
}

func (c *Config) validateCloudinaryMediaURLForContext(rawURL string, context string) error {
	if strings.TrimSpace(rawURL) == "" {
		return fmt.Errorf("url is required")
	}
	folder, ok := cloudinaryFolderForContext(context)
	if !ok {
		return fmt.Errorf("invalid context")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url")
	}
	if u.Scheme != "https" {
		return fmt.Errorf("url must use https")
	}
	if !strings.EqualFold(u.Hostname(), "res.cloudinary.com") {
		return fmt.Errorf("url host must be res.cloudinary.com")
	}
	trimmedPath := strings.TrimPrefix(strings.TrimSpace(u.Path), "/")
	if c.CloudinaryCloudName != "" && !strings.HasPrefix(trimmedPath, c.CloudinaryCloudName+"/") {
		return fmt.Errorf("url must match configured cloud")
	}
	if !strings.Contains(trimmedPath, "/"+folder+"/") && !strings.HasSuffix(trimmedPath, "/"+folder) {
		return fmt.Errorf("url must be within allowed folder")
	}
	return nil
}

func (c *Config) postCloudinarySignature(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(c.CloudinaryCloudName) == "" || strings.TrimSpace(c.CloudinaryAPIKey) == "" {
		respondWithError(w, http.StatusInternalServerError, "Cloudinary is not configured")
		return
	}
	var req cloudinarySignatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	folder, ok := cloudinaryFolderForContext(req.Context)
	if !ok {
		respondWithError(w, http.StatusBadRequest, "context must be avatar or player_profile")
		return
	}
	userID, ok := r.Context().Value("user_id").(int32)
	if !ok || userID == 0 {
		respondWithError(w, http.StatusUnauthorized, "Authentication required")
		return
	}
	timestamp := time.Now().Unix()
	publicID := fmt.Sprintf("%s-%d-%d", req.Context, userID, timestamp)

	resp := cloudinarySignatureResponse{
		CloudName:      c.CloudinaryCloudName,
		APIKey:         c.CloudinaryAPIKey,
		Timestamp:      timestamp,
		Folder:         folder,
		PublicID:       publicID,
		MaxUploadBytes: cloudinaryMaxUploadBytes,
	}

	// Optional fallback path if team uses unsigned presets.
	if strings.TrimSpace(c.CloudinaryUploadPreset) != "" {
		resp.UploadPreset = c.CloudinaryUploadPreset
	}
	if strings.TrimSpace(c.CloudinaryAPISecret) != "" {
		resp.Signature = cloudinarySign(map[string]string{
			"folder":    folder,
			"public_id": publicID,
			"timestamp": strconv.FormatInt(timestamp, 10),
		}, c.CloudinaryAPISecret)
	}

	respondWithJSON(w, http.StatusOK, resp)
}
