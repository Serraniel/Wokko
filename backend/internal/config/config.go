package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL        string
	JWTSecret          string
	UploadPath         string
	PublicURL          string
	Port               string
	OpenRegistration   bool
	AllowPublicRecipes bool
	MCPEnabled         bool
	RPDisplayName      string
	RPID               string
	RPOrigin           string
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	publicURL := os.Getenv("PUBLIC_URL")
	if publicURL == "" {
		publicURL = "http://localhost:3000"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	uploadPath := os.Getenv("UPLOAD_PATH")
	if uploadPath == "" {
		uploadPath = "/data/uploads"
	}
	openReg, _ := strconv.ParseBool(os.Getenv("OPEN_REGISTRATION"))
	allowPublic := true
	if v := os.Getenv("ALLOW_PUBLIC_RECIPES"); v != "" {
		allowPublic, _ = strconv.ParseBool(v)
	}
	mcpEnabled, _ := strconv.ParseBool(os.Getenv("MCP_ENABLED"))
	rpDisplayName := os.Getenv("RP_DISPLAY_NAME")
	if rpDisplayName == "" {
		rpDisplayName = "Wokko"
	}
	rpID := os.Getenv("RP_ID")
	if rpID == "" {
		rpID = "localhost"
	}
	rpOrigin := os.Getenv("RP_ORIGIN")
	if rpOrigin == "" {
		rpOrigin = "http://localhost:3000"
	}
	return &Config{
		DatabaseURL:        dbURL,
		JWTSecret:          jwtSecret,
		UploadPath:         uploadPath,
		PublicURL:          publicURL,
		Port:               port,
		OpenRegistration:   openReg,
		AllowPublicRecipes: allowPublic,
		MCPEnabled:         mcpEnabled,
		RPDisplayName:      rpDisplayName,
		RPID:               rpID,
		RPOrigin:           rpOrigin,
	}, nil
}
