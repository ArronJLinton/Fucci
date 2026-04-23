package config

type Config struct {
	DB_URL           string
	FOOTBALL_API_KEY string
	RAPID_API_KEY    string
	// NEWS_API_KEY is the X-API-Key for the Open Web Ninja realtime news HTTP client. If empty, RAPID_API_KEY is used.
	NEWS_API_KEY string
	// NEWS_BASE_URL is the full base URL for GET /search-style news requests (query string appended). Empty uses the client default.
	NEWS_BASE_URL              string
	GOOGLE_OAUTH_CLIENT_ID     string
	GOOGLE_OAUTH_CLIENT_SECRET string
	GOOGLE_OAUTH_REDIRECT_URIS string
	// GOOGLE_OAUTH_CALLBACK_URL is the full backend URL registered with Google (e.g. https://api.example.com/v1/api/auth/google/callback).
	GOOGLE_OAUTH_CALLBACK_URL string
	// GOOGLE_OAUTH_ALLOW_DEV_RETURN_URLS allows exp:// and http(s):// localhost/private-LAN return URLs for /auth/google/start.
	// When unset, this is true for ENVIRONMENT development|dev|local and false otherwise (set ENVIRONMENT=production when deploying).
	GOOGLE_OAUTH_ALLOW_DEV_RETURN_URLS bool
	CLOUDINARY_CLOUD_NAME              string
	CLOUDINARY_API_KEY                 string
	CLOUDINARY_API_SECRET              string
	CLOUDINARY_UPLOAD_PRESET           string
	REDIS_URL                          string
	OPENAI_API_KEY                     string
	OPENAI_BASE_URL                    string
	PORT                               string
	ENVIRONMENT                        string
	JWT_SECRET                         string
	SYSTEM_USER_EMAIL                  string // Email for Fucci system user (006 seeded comments); default contact@magistri.dev
}
