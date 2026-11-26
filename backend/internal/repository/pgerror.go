package repository

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// isUniqueViolation reports whether err is a Postgres unique violation
// (23505): a concurrent create raced past the existence pre-check.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// isLicenseLimitExceeded reports whether err is the store license-cap
// trigger (000028): a concurrent device create raced past the service
// pre-check. Maps to 403, not 500.
func isLicenseLimitExceeded(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "P0001" &&
		strings.Contains(pgErr.Message, "store_license_limit_exceeded")
}
