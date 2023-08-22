package env

import (
	"fmt"
	"os"
	"strings"
)

const (
	AuthUsername     = "AUTH_USERNAME"
	AuthPassword     = "AUTH_PASSWORD"
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
		AuthUsername,
		AuthPassword,
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

	if len(missingEnvVarNames) != 0 {
		err = fmt.Errorf("some environment variables are missing: %s", strings.Join(missingEnvVarNames, ", "))
	}

	return err
}

func Get(name string) string {
	return os.Getenv(name)
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
	return fmt.Sprintf("%s.sqlite", DatabaseName)
}
