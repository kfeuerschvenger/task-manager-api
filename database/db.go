package database

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// validSSLModes lists the sslmode values PostgreSQL accepts.
var validSSLModes = []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}

// SSLMode returns the PostgreSQL sslmode taken from DB_SSLMODE, defaulting to "disable".
// It does not validate the value; use ValidateSSLMode for that.
func SSLMode() string {
	if mode := os.Getenv("DB_SSLMODE"); mode != "" {
		return mode
	}
	return "disable"
}

// ValidateSSLMode returns an error when DB_SSLMODE is not one of the sslmode values PostgreSQL accepts.
func ValidateSSLMode() error {
	mode := SSLMode()
	for _, valid := range validSSLModes {
		if mode == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid DB_SSLMODE %q: must be one of %s", mode, strings.Join(validSSLModes, ", "))
}

// SSLRootCert returns the path of the CA certificate used by verify-ca and verify-full, taken from DB_SSLROOTCERT.
// An empty value means libpq falls back to its default location.
func SSLRootCert() string {
	return os.Getenv("DB_SSLROOTCERT")
}

// quoteDSNValue quotes a value for a libpq key/value connection string:
// it is wrapped in single quotes and every backslash and single quote is escaped with a backslash.
func quoteDSNValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `'`, `\'`)
	return "'" + v + "'"
}

// buildDSN builds the libpq key/value connection string used by GORM, quoting every value.
func buildDSN(host, port, user, password, dbname string) (string, error) {
	if err := ValidateSSLMode(); err != nil {
		return "", err
	}

	pairs := [][2]string{
		{"host", host},
		{"user", user},
		{"password", password},
		{"dbname", dbname},
		{"port", port},
		{"sslmode", SSLMode()},
	}
	if cert := SSLRootCert(); cert != "" {
		pairs = append(pairs, [2]string{"sslrootcert", cert})
	}

	parts := make([]string, 0, len(pairs)+1)
	for _, p := range pairs {
		parts = append(parts, p[0]+"="+quoteDSNValue(p[1]))
	}
	// TimeZone is a fixed literal; the pgx driver keeps quotes around runtime parameters, so it stays unquoted.
	parts = append(parts, "TimeZone=UTC")
	return strings.Join(parts, " "), nil
}

// buildURL builds the postgres:// URL used by the migrations, escaping every component.
func buildURL(host, port, user, password, dbname string) (string, error) {
	if err := ValidateSSLMode(); err != nil {
		return "", err
	}

	query := url.Values{}
	query.Set("sslmode", SSLMode())
	if cert := SSLRootCert(); cert != "" {
		query.Set("sslrootcert", cert)
	}

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, password),
		Host:     net.JoinHostPort(host, port),
		Path:     "/" + dbname,
		RawQuery: query.Encode(),
	}
	return u.String(), nil
}

// Connect initializes the database connection using GORM with PostgreSQL.
// It reads the database connection parameters from environment variables
func Connect() error {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	dsn, err := buildDSN(host, port, user, password, dbname)
	if err != nil {
		return err
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	DB = db
	return nil
}
