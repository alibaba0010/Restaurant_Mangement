package database

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"go.uber.org/zap"

	"github.com/alibaba0010/postgres-api/internal/logger"
)

// PrettyQueryHook is a Bun query hook that formats SQL queries and logs them
// using the project's structured logger in a compact, readable way.
type PrettyQueryHook struct{}

// BeforeQuery is required by bun.QueryHook but we don't need to modify context.
func (h PrettyQueryHook) BeforeQuery(ctx context.Context, _ *bun.QueryEvent) context.Context {
    return ctx
}

// AfterQuery formats the query and logs it with duration, args and error (if any).
func (h PrettyQueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
    // normalize whitespace and newlines
    q := strings.TrimSpace(event.Query)
    q = strings.ReplaceAll(q, "\n", " ")
    // collapse multiple spaces
    re := regexp.MustCompile(`\s+`)
    q = re.ReplaceAllString(q, " ")

    // shorten long queries for readability
    formatted := q
    if len(formatted) > 1000 {
        formatted = formatted[:1000] + "..."
    }

    // Prepare fields
    dur := time.Since(event.StartTime)
    args := event.QueryArgs
    err := event.Err

    // Log using the project's logger at Debug level for SQL statements
    fields := []zap.Field{
        zap.String("sql", formatted),
        zap.Duration("duration", dur),
    }
    if args != nil {
        fields = append(fields, zap.Any("args", args))
    }
    if err != nil {
        fields = append(fields, zap.Error(err))
        logger.Log.Warn("sql query", fields...)
        return
    }
    logger.Log.Debug("sql query", fields...)
}
