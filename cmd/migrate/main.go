package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	postgres "github.com/alibaba0010/postgres-api/internal/database"
	"github.com/alibaba0010/postgres-api/internal/logger"
	"github.com/alibaba0010/postgres-api/internal/migration"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"
	"github.com/urfave/cli/v2"
)

func main() {
	// initialize project's logger (used by internal packages)
	logger.InitLogger()

	// connect to the database using the project's database package
	db := postgres.ConnectDB()

	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithEnabled(true),
		bundebug.FromEnv(),
	))
	l := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true}))

	app := &cli.App{
		Name: "migrate",
		Commands: []*cli.Command{
			newMigrationCmd(
				migrate.NewMigrator(db, migration.New(), migrate.WithMarkAppliedOnSuccess(true)),
				l,
			),
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

//nolint:errcheck // internal use only
func newMigrationCmd(m *migrate.Migrator, l *slog.Logger) *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "database migrations",
		Subcommands: []*cli.Command{
			{
				Name:  "init",
				Usage: "create migrations table",
				Action: func(ctx *cli.Context) error {
					return m.Init(ctx.Context)
				},
			},
			{
				Name:  "up",
				Usage: "run up migration",
				Action: func(ctx *cli.Context) error {
					if err := m.Lock(ctx.Context); err != nil {
						return fmt.Errorf("lock: %w", err)
					}
					defer m.Unlock(ctx.Context)

					group, err := m.Migrate(ctx.Context)
					if err != nil {
						return fmt.Errorf("migrate: %w", err)
					}
					if group.IsZero() {
						l.Info("there are no new migrations to run (database is up to date)")
						return nil
					}
					// l.Info("migrated to ", slog.Any("grous", group))
					l.Info("Migrated successfully", 
						slog.Int64("group_id", group.ID),
						slog.Int("migrations_count", len(group.Migrations)),
					)
					return nil
				},
			},
			{
				Name:  "down",
				Usage: "run down migration",
				Action: func(ctx *cli.Context) error {
					if err := m.Lock(ctx.Context); err != nil {
						return fmt.Errorf("lock migration: %w", err)
					}
					defer m.Unlock(ctx.Context)

					group, err := m.Rollback(ctx.Context)
					if err != nil {
						return fmt.Errorf("rollback: %w", err)
					}
					if group.IsZero() {
						l.Info("there are no groups to rollback")
						return nil
					}
					// l.Info("rolled back to ", slog.Any("grous", group))
						l.Info("Rolled back successfully", 
						slog.Int64("group_id", group.ID),
						slog.Int("migrations_count", len(group.Migrations)),
					)
					return nil
				},
			},
			{
				Name:  "create",
				Usage: "create up and down sql migrations",
				Action: func(ctx *cli.Context) error {
					name := strings.Join(ctx.Args().Slice(), "_")
					files, err := m.CreateTxSQLMigrations(ctx.Context, name)
					if err != nil {
						return fmt.Errorf("create migration: %w", err)
					}
					for _, f := range files {
						l.Info("created migration %s (%s)", f.Name, f.Path)
					}
					return nil
				},
			},
			{
				Name:  "status",
				Usage: "print migration status",
				Action: func(ctx *cli.Context) error {
					ms, err := m.MigrationsWithStatus(ctx.Context)
					if err != nil {
						return fmt.Errorf("migration status: %w", err)
					}
					var buf strings.Builder
					buf.WriteString(fmt.Sprintf("migrations: %s - ", ms))
					buf.WriteString(fmt.Sprintf("unapplied migrations: %s - ", ms.Unapplied()))
					buf.WriteString(fmt.Sprintf("last migration group: %s", ms.LastGroup()))
					l.Info(buf.String())
					return nil
				},
			},
		},
	}
}