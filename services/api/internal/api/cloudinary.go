package api

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
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

// isCloudinaryVersionSegment reports whether s is a Cloudinary version path component (e.g. v1695123456).
func isCloudinaryVersionSegment(s string) bool {
	if len(s) < 2 {
		return false
	}
	if s[0] != 'v' && s[0] != 'V' {
		return false
	}
	for _, r := range s[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// cloudinaryPublicIDFromDeliveryPath extracts the public_id (including folder prefix) from a delivery URL path
// after /image/upload/, skipping optional transformation segments and an optional v{version} segment.
func cloudinaryPublicIDFromDeliveryPath(trimmedPath, cloudName string) (string, error) {
	var suffix string
	if strings.TrimSpace(cloudName) != "" {
		prefix := cloudName + "/"
		if !strings.HasPrefix(trimmedPath, prefix) {
			return "", fmt.Errorf("url must match configured cloud")
		}
		afterCloud := trimmedPath[len(prefix):]
		const uploadPrefix = "image/upload/"
		if len(afterCloud) < len(uploadPrefix) || !strings.EqualFold(afterCloud[:len(uploadPrefix)], uploadPrefix) {
			return "", fmt.Errorf("url must use Cloudinary image delivery path")
		}
		suffix = afterCloud[len(uploadPrefix):]
	} else {
		lower := strings.ToLower(trimmedPath)
		const marker = "image/upload/"
		idx := strings.Index(lower, marker)
		if idx < 0 {
			return "", fmt.Errorf("url must use Cloudinary image delivery path")
		}
		suffix = trimmedPath[idx+len(marker):]
	}

	var parts []string
	for _, s := range strings.Split(suffix, "/") {
		if s != "" {
			parts = append(parts, s)
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("url missing asset path after upload")
	}

	versionIdx := -1
	for i, seg := range parts {
		if isCloudinaryVersionSegment(seg) {
			versionIdx = i
			break
		}
	}
	var rest []string
	if versionIdx >= 0 {
		rest = parts[versionIdx+1:]
	} else {
		// No version segment: strip only leading comma-separated transformation chains, then treat the rest as public_id.
		i := 0
		for i < len(parts) && strings.Contains(parts[i], ",") {
			i++
		}
		rest = parts[i:]
	}
	if len(rest) == 0 {
		return "", fmt.Errorf("url missing public id under allowed folder")
	}
	return strings.Join(rest, "/"), nil
}

func publicIDHasAllowedFolderPrefix(publicID, folder string) bool {
	if publicID == folder {
		return true
	}
	return strings.HasPrefix(publicID, folder+"/")
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
	cleaned := path.Clean("/" + strings.TrimSpace(u.Path))
	if cleaned == "/" || cleaned == "." {
		return fmt.Errorf("invalid url")
	}
	trimmedPath := strings.TrimPrefix(cleaned, "/")

	publicID, err := cloudinaryPublicIDFromDeliveryPath(trimmedPath, c.CloudinaryCloudName)
	if err != nil {
		return err
	}
	if !publicIDHasAllowedFolderPrefix(publicID, folder) {
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

	if strings.TrimSpace(resp.Signature) == "" && strings.TrimSpace(resp.UploadPreset) == "" {
		respondWithError(w, http.StatusInternalServerError,
			"Cloudinary upload auth is not configured: set CLOUDINARY_API_SECRET for signed uploads or CLOUDINARY_UPLOAD_PRESET for an unsigned upload preset")
		return
	}

	respondWithJSON(w, http.StatusOK, resp)
}
