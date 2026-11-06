package repository

import (
	"database/sql"
	"strings"
	"time"
)

func strPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	v := ns.String
	return &v
}

func intPtr(ni sql.NullInt64) *int {
	if !ni.Valid {
		return nil
	}
	v := int(ni.Int64)
	return &v
}

func timePtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	v := nt.Time
	return &v
}

func nullStr(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func nullInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

// splitKeywords splits a chr(31)-joined keyword string.
func splitKeywords(joined string) []string {
	joined = strings.TrimSpace(joined)
	if joined == "" {
		return []string{}
	}
	return strings.Split(joined, "\x1f")
}

// joinKeywords joins keywords for string_to_array($n, chr(31)) writes.
func joinKeywords(keywords []string) string {
	clean := make([]string, 0, len(keywords))
	for _, k := range keywords {
		if strings.TrimSpace(k) != "" {
			clean = append(clean, k)
		}
	}
	return strings.Join(clean, "\x1f")
}

func clampLimit(limit, def, max int) int {
	if limit <= 0 {
		return def
	}
	if limit > max {
		return max
	}
	return limit
}
