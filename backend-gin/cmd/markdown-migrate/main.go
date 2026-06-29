package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/markdownmigration"
	"yuem-go/backend-gin/internal/storage"
)

func main() {
	apply := flag.Bool("apply", false, "apply updates to the database")
	dryRun := flag.Bool("dry-run", false, "preview changes without updating the database")
	backup := flag.Bool("backup", false, "create backup rows before applying updates")
	backupTable := flag.String("backup-table", "markdown_migration_backups", "backup table name")
	tables := flag.String("tables", "", "comma-separated table or table.column targets")
	batchSize := flag.Int("batch-size", 200, "rows to scan per batch")
	limit := flag.Int("limit", 0, "maximum rows to scan per target; 0 means unlimited")
	sample := flag.Int("sample", 5, "number of changed samples to print per target")
	jsonOutput := flag.Bool("json", false, "print JSON output")
	flag.Parse()

	if *dryRun {
		*apply = false
	}
	cfg := config.Load()
	cfg.Database.AutoMigrate = false
	db, err := storage.OpenDatabase(cfg.Database)
	if err != nil {
		exitErr(err)
	}
	if db == nil {
		exitErr(fmt.Errorf("database is not configured"))
	}
	sqlDB, err := db.DB()
	if err != nil {
		exitErr(err)
	}
	defer sqlDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	result, err := markdownmigration.Run(ctx, db, markdownmigration.Options{
		Apply:       *apply,
		Backup:      *backup,
		BackupTable: *backupTable,
		BatchSize:   *batchSize,
		Limit:       *limit,
		Sample:      *sample,
		Tables:      splitCSV(*tables),
	})
	if err != nil {
		exitErr(err)
	}
	if *jsonOutput {
		data, err := markdownmigration.PrintJSON(result)
		if err != nil {
			exitErr(err)
		}
		fmt.Println(string(data))
		return
	}
	printText(result)
}

func splitCSV(value string) []string {
	out := []string{}
	for item := range strings.SplitSeq(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func printText(result markdownmigration.Result) {
	mode := "dry-run"
	if result.Apply {
		mode = "apply"
	}
	fmt.Printf("markdown migration %s started=%s finished=%s\n", mode, result.StartedAt, result.FinishedAt)
	if result.Backup {
		fmt.Printf("backup table: %s\n", result.BackupTable)
	}
	for _, target := range result.Targets {
		fmt.Printf("- %s.%s scanned=%d changed=%d updated=%d backed_up=%d\n",
			target.Table,
			target.Column,
			target.Scanned,
			target.Changed,
			target.Updated,
			target.BackedUp,
		)
		if target.ErrorText != "" {
			fmt.Printf("  error: %s\n", target.ErrorText)
		}
		for _, sample := range target.Samples {
			fmt.Printf("  sample id=%v\n", sample.ID)
			fmt.Printf("    before: %s\n", oneLine(sample.Before))
			fmt.Printf("    after:  %s\n", oneLine(sample.After))
		}
	}
}

func oneLine(value string) string {
	value = strings.ReplaceAll(value, "\n", `\n`)
	if len([]rune(value)) <= 220 {
		return value
	}
	return string([]rune(value)[:220]) + "..."
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
