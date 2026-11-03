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

// validSSLModes lists the sslmode values supported by both drivers in use. The migration driver (lib/pq)
// rejects allow and prefer, so they are not accepted even though PostgreSQL itself defines them.
var validSSLModes = []string{"disable", "require", "verify-ca", "verify-full"}

// SSLMode returns the PostgreSQL sslmode taken from DB_SSLMODE, defaulting to "disable".
// It does not validate the value; use ValidateSSLMode for that.
func SSLMode() string {
	if mode := os.Getenv("DB_SSLMODE"); mode != "" {
		return mode
	}
	return "disable"
}

// isValidSSLMode reports whether mode is in the allowlist.
func isValidSSLMode(mode string) bool {
	for _, valid := range validSSLModes {
		if mode == valid {
			return true
		}
	}
	return false
}

// ValidateSSLMode returns an error when DB_SSLMODE is not one of the supported sslmode values.
func ValidateSSLMode() error {
	mode := SSLMode()
	if !isValidSSLMode(mode) {
		return fmt.Errorf("invalid DB_SSLMODE %q: must be one of %s", mode, strings.Join(validSSLModes, ", "))
	}
	return nil
}

// ValidateDatabaseURL returns an error when DATABASE_URL is set, parses, and carries an sslmode query
// parameter outside the allowlist. An unset DATABASE_URL or one without sslmode is accepted.
func ValidateDatabaseURL() error {
	raw := os.Getenv("DATABASE_URL")
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid DATABASE_URL: %w", err)
	}
	if mode := u.Query().Get("sslmode"); mode != "" && !isValidSSLMode(mode) {
		return fmt.Errorf("invalid sslmode %q in DATABASE_URL: must be one of %s", mode, strings.Join(validSSLModes, ", "))
	}
	return nil
}

// bareHost strips the brackets from an IPv6 literal such as "[::1]" so net.JoinHostPort can add them back.
func bareHost(host string) string {
	return strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
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
		{"host", bareHost(host)},
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
		Host:     net.JoinHostPort(bareHost(host), port),
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
