package config

import (
	"fmt"
	"net/url"
	"strings"
)

func databaseURLFromEnv() string {
	if value := getEnv("DATABASE_URL", ""); value != "" {
		return value
	}
	if getEnv("DB_HOST", "") == "" && getEnv("DB_USER", "") == "" && getEnv("DB_NAME", "") == "" {
		return ""
	}
	host := getEnv("DB_HOST", "localhost")
	user := getEnv("DB_USER", "root")
	password := getEnv("DB_PASSWORD", "")
	name := getEnv("DB_NAME", "xiaoshiliu")
	port := intEnv("DB_PORT", 3306)
	driver := strings.ToLower(getEnv("DB_DRIVER", getEnv("DATABASE_DRIVER", "")))
	if driver == "postgres" || driver == "postgresql" || port == 5432 {
		return buildPostgresURL(user, password, host, port, name)
	}
	return buildMySQLURL(user, password, host, port, name)
}

func buildMySQLURL(user string, password string, host string, port int, name string) string {
	if port <= 0 {
		port = 3306
	}
	out := url.URL{
		Scheme: "mysql",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%d", host, port),
		Path:   "/" + name,
	}
	return out.String()
}

func buildPostgresURL(user string, password string, host string, port int, name string) string {
	if port <= 0 {
		port = 5432
	}
	out := url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%d", host, port),
		Path:   "/" + name,
	}
	return out.String()
}

func frontendBaseURL() string {
	for _, value := range []string{
		getEnv("FRONTEND_URL", ""),
		getEnv("DISCORD_SITE_URL", ""),
		getEnv("LOCAL_BASE_URL", ""),
		webBaseFromAPI(getEnv("API_BASE_URL", "")),
	} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimRight(value, "/")
		}
	}
	return "http://localhost:5173"
}

func webBaseFromAPI(value string) string {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	return strings.TrimSuffix(value, "/api")
}
