package config

type Config struct {
	DB_URL             string
	FOOTBALL_API_KEY   string
	RAPID_API_KEY      string
	REDIS_URL          string
	OPENAI_API_KEY     string
	OPENAI_BASE_URL    string
	PORT               string
	ENVIRONMENT        string
	JWT_SECRET         string
	SYSTEM_USER_EMAIL  string // Email for Fucci system user (006 seeded comments); default contact@magistri.dev
}
