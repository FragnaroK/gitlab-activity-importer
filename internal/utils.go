package internal

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func CheckEnvVariables() error {
	if os.Getenv("ENV") == "DEVELOPMENT" {
		if err := godotenv.Load(); err != nil {
			return fmt.Errorf("error loading .env file: %v", err)
		}
	}

	requiredEnvVars := []string{
		"BASE_URL",
		"GITLAB_TOKEN",
		"COMMITER_NAME",
		"COMMITER_EMAIL",
		"ORIGIN_REPO_URL",
		"ORIGIN_TOKEN",
	}

	var missingVars []string
	for _, envVar := range requiredEnvVars {
		log.Printf("Checking for environment variable: %s", envVar)
		if os.Getenv(envVar) == "" {
			log.Printf("Missing environment variable: %s", envVar)
			missingVars = append(missingVars, envVar)
		}
	}

	if len(missingVars) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missingVars, ", "))
	}

	return nil
}

func GetHomeDirectory() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Unable to get the user home directory:", err)
	}

	log.Println("Home directory:", homeDir)
	return homeDir
}
