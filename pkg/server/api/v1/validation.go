package v1

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// ListScansQuery represents supported query params for GET /api/v1/scans
type ListScansQuery struct {
	Status string
	Limit  int
	Cursor string // Opaque cursor for pagination (empty for first page)
}

// ParseListScansQuery parses and validates query params.
// Returns validated query with sane defaults (Limit=50) when omitted.
func ParseListScansQuery(r *http.Request) (*ListScansQuery, error) {
	q := r.URL.Query()
	var res ListScansQuery

	if v := strings.TrimSpace(q.Get("status")); v != "" {
		// Validate only if provided
		if err := validate.Var(v, "oneof=pending running completed failed"); err != nil {
			return nil, &ValidationError{Field: "status", Reason: "must be one of: pending,running,completed,failed"}
		}
		res.Status = v
	}

	if v := strings.TrimSpace(q.Get("limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, &ValidationError{Field: "limit", Reason: "must be an integer"}
		}
		if err := validate.Var(n, "min=1,max=100"); err != nil {
			return nil, &ValidationError{Field: "limit", Reason: "must be between 1 and 100"}
		}
		res.Limit = n
	}

	// Cursor parameter (opaque string, no validation needed beyond trimming)
	if v := strings.TrimSpace(q.Get("cursor")); v != "" {
		res.Cursor = v
	}

	// Defaults
	if res.Limit == 0 {
		res.Limit = 50
	}

	return &res, nil
}

// ValidationError is a lightweight error used for 400 responses.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Field == "" {
		return "validation failed"
	}
	if e.Reason == "" {
		return e.Field + ": invalid"
	}
	return e.Field + ": " + e.Reason
}
