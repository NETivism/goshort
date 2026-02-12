package env

import (
	"fmt"
	"os"
	"strings"
)

const (
	AuthType         = "AUTH_TYPE"
	AuthUsername      = "AUTH_USERNAME"
	AuthPassword     = "AUTH_PASSWORD"
	AuthApikey       = "AUTH_APIKEY"
	Debug            = "DEBUG"
	ListenPort       = "LISTEN_PORT"
	DatabaseType     = "DATABASE_TYPE"
	DatabaseHost     = "DATABASE_HOST"
	DatabasePort     = "DATABASE_PORT"
	DatabaseName     = "DATABASE_NAME"
	DatabaseUser     = "DATABASE_USER"
	DatabasePassword = "DATABASE_PASSWORD"
	DatabaseMigrate  = "DATABASE_MIGRATE"
)

var (
	requiredEnvVarNames = []string{
		DatabaseType,
		ListenPort,
	}
)

func Check() error {
	var err error

	missingEnvVarNames := []string{}

	for _, envVarName := range requiredEnvVarNames {
		if Get(envVarName) == "" {
			missingEnvVarNames = append(missingEnvVarNames, envVarName)
		}
	}
	dbType := Get(DatabaseType)
	switch dbType {
	case "mysql":
	case "postgres":
		for _, envVarName := range []string{DatabaseHost, DatabaseUser, DatabasePassword, DatabasePort, DatabaseName} {
			if Get(envVarName) == "" {
				missingEnvVarNames = append(missingEnvVarNames, envVarName)
			}
		}
	case "sqlite":
		if Get(DatabaseName) == "" {
			missingEnvVarNames = append(missingEnvVarNames, DatabaseName)
		}
	}

	authType := Get(AuthType)
	switch authType {
	case "apikey":
		if Get(AuthApikey) == "" {
			missingEnvVarNames = append(missingEnvVarNames, AuthApikey)
		}
	default:
		// default to basic auth
		for _, envVarName := range []string{AuthUsername, AuthPassword} {
			if Get(envVarName) == "" {
				missingEnvVarNames = append(missingEnvVarNames, envVarName)
			}
		}
	}

	if len(missingEnvVarNames) != 0 {
		err = fmt.Errorf("some environment variables are missing: %s", strings.Join(missingEnvVarNames, ", "))
	}

	return err
}

func Get(name string) string {
	val := os.Getenv(name)
	return val
}

func GetPostgresDsn() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		Get(DatabaseUser),
		Get(DatabasePassword),
		Get(DatabaseHost),
		Get(DatabasePort),
		Get(DatabaseName),
	)
}

func GetMysqlDsn() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		Get(DatabaseUser),
		Get(DatabasePassword),
		Get(DatabaseHost),
		Get(DatabasePort),
		Get(DatabaseName),
	)
}

func GetSqliteDsn() string {
	return fmt.Sprintf("%s.sqlite", Get(DatabaseName))
}
