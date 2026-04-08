package config

type Config struct {
	DB_URL                     string
	FOOTBALL_API_KEY           string
	RAPID_API_KEY              string
	GOOGLE_OAUTH_CLIENT_ID     string
	GOOGLE_OAUTH_CLIENT_SECRET string
	GOOGLE_OAUTH_REDIRECT_URIS string
	CLOUDINARY_CLOUD_NAME      string
	CLOUDINARY_API_KEY         string
	CLOUDINARY_API_SECRET      string
	CLOUDINARY_UPLOAD_PRESET   string
	REDIS_URL                  string
	OPENAI_API_KEY             string
	OPENAI_BASE_URL            string
	PORT                       string
	ENVIRONMENT                string
	JWT_SECRET                 string
	SYSTEM_USER_EMAIL          string // Email for Fucci system user (006 seeded comments); default contact@magistri.dev
}
