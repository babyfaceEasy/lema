package seeder

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const (
	seedDir                  = "db/seeders"
	ErrSQlDuplicateEntryCode = "23505"
)

type seed struct {
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
	Values  [][]any  `json:"values"`
}

func Seed(db *sql.DB) error {
	sqlxDB := sqlx.NewDb(db, "postgres")

	files, err := os.ReadDir(seedDir)
	if err != nil {
		return fmt.Errorf("error in reading seeder directory, err: %w", err)
	}

	for _, file := range files {
		f := strings.Split(file.Name(), ".")
		if file.IsDir() || f[len(f)-1] != "json" {
			continue
		}

		content, err := os.ReadFile(filepath.Join(seedDir, file.Name()))
		if err != nil {
			return fmt.Errorf("error reading file, err: %w", err)
		}

		var data seed

		if err = sonic.Unmarshal(content, &data); err != nil {
			return fmt.Errorf("error during un-marsahling file content, err: %w", err)
		}

		if err := execQuery(data, sqlxDB, file.Name()); err != nil {
			log.Println(err)
		}
	}

	return nil
}

func execQuery(data seed, db *sqlx.DB, fileName string) error {
	query := fmt.Sprintf(
		`INSERT INTO %s (%s) VALUES (%s)`,
		data.Table,
		strings.Join(data.Columns, ","),
		preparedInsertQuery(data.Columns),
	)

	for _, value := range data.Values {
		if _, err := db.Exec(sqlx.Rebind(sqlx.DOLLAR, query), value...); err != nil {
			if !IsDuplicateEntry(err) {
				log.Printf(
					"error in running seeder file %s. err: %s",
					fileName,
					err,
				)
			}
		}
	}

	return nil
}

func preparedInsertQuery(columns []string) string {
	var query string

	for i := 0; i < len(columns); i++ {
		if i != len(columns)-1 {
			query += "?,"
			continue
		}

		query += "?"
	}

	return query
}

func IsDuplicateEntry(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == ErrSQlDuplicateEntryCode
	}

	return false
}
