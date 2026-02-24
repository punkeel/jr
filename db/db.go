package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

type Job struct {
	ID             int64
	CreatedAtUTC   string
	Name           string
	Unit           string
	Cwd            string
	ArgvJSON       string
	EnvJSON        string
	PropertiesJSON string
	Host           sql.NullString
	User           sql.NullString
	Notes          sql.NullString
	LastKnownState sql.NullString
	LastStateAtUTC sql.NullString
}

type JobWithArgs struct {
	Job
	Argv []string
	Env  map[string]string
}

func InitDB() error {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dataDir = filepath.Join(home, ".local", "state")
	}

	jrDir := filepath.Join(dataDir, "jr")
	if err := os.MkdirAll(jrDir, 0755); err != nil {
		return err
	}

	dbPath := filepath.Join(jrDir, "jr.db")

	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	return createTables()
}

func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

func createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at_utc TEXT NOT NULL,
		name TEXT NOT NULL,
		unit TEXT UNIQUE NOT NULL,
		cwd TEXT NOT NULL,
		argv_json TEXT NOT NULL,
		env_json TEXT,
		properties_json TEXT,
		host TEXT,
		user TEXT,
		notes TEXT,
		last_known_state TEXT,
		last_state_at_utc TEXT
	);
	
	CREATE INDEX IF NOT EXISTS idx_jobs_created ON jobs(created_at_utc DESC);
	CREATE INDEX IF NOT EXISTS idx_jobs_unit ON jobs(unit);
	CREATE INDEX IF NOT EXISTS idx_jobs_name ON jobs(name);
	`

	_, err := DB.Exec(query)
	return err
}

func CreateJob(name, unit, cwd string, argv []string, env map[string]string, props map[string]string, host, user string) (int64, error) {
	argvJSON, err := json.Marshal(argv)
	if err != nil {
		return 0, err
	}

	var envJSON []byte
	if env != nil {
		envJSON, err = json.Marshal(env)
		if err != nil {
			return 0, err
		}
	}

	var propsJSON []byte
	if props != nil {
		propsJSON, err = json.Marshal(props)
		if err != nil {
			return 0, err
		}
	}

	query := `
		INSERT INTO jobs (created_at_utc, name, unit, cwd, argv_json, env_json, properties_json, host, user)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := DB.Exec(query,
		time.Now().UTC().Format(time.RFC3339),
		name,
		unit,
		cwd,
		string(argvJSON),
		string(envJSON),
		string(propsJSON),
		sql.NullString{String: host, Valid: host != ""},
		sql.NullString{String: user, Valid: user != ""},
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func GetJobByID(id int64) (*Job, error) {
	query := `SELECT * FROM jobs WHERE id = ?`
	row := DB.QueryRow(query, id)

	return scanJob(row)
}

func GetJobByUnit(unit string) (*Job, error) {
	query := `SELECT * FROM jobs WHERE unit = ?`
	row := DB.QueryRow(query, unit)

	return scanJob(row)
}

func FindJobByPartial(partial string) (*Job, error) {
	if id, err := parseInt(partial); err == nil {
		return GetJobByID(id)
	}

	return GetJobByUnit(partial)
}

func parseInt(s string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(s, "%d", &id)
	return id, err
}

func ListJobs(limit int, all bool) ([]*Job, error) {
	var query string
	if all {
		query = `SELECT * FROM jobs ORDER BY created_at_utc DESC`
	} else {
		query = `SELECT * FROM jobs ORDER BY created_at_utc DESC LIMIT ?`
	}

	var rows *sql.Rows
	var err error
	if all {
		rows, err = DB.Query(query)
	} else {
		rows, err = DB.Query(query, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job, err := scanJobRows(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

func ListJobsByName(name string, limit int) ([]*Job, error) {
	query := `SELECT * FROM jobs WHERE name LIKE ? ORDER BY created_at_utc DESC LIMIT ?`
	rows, err := DB.Query(query, name+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job, err := scanJobRows(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

func DeleteJob(id int64) error {
	query := `DELETE FROM jobs WHERE id = ?`
	_, err := DB.Exec(query, id)
	return err
}

func PruneJobs(keep int, olderThan time.Duration, failedOnly bool) error {
	var conditions []string
	var args []interface{}

	// Always keep the most recent N jobs
	if keep > 0 {
		conditions = append(conditions, "id NOT IN (SELECT id FROM jobs ORDER BY created_at_utc DESC LIMIT ?)")
		args = append(args, keep)
	}

	// Filter by age
	if olderThan > 0 {
		cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339)
		conditions = append(conditions, "created_at_utc < ?")
		args = append(args, cutoff)
	}

	// Filter by failed state
	if failedOnly {
		conditions = append(conditions, "last_known_state = 'failed'")
	}

	if len(conditions) == 0 {
		return nil
	}

	query := "DELETE FROM jobs WHERE " + conditions[0]
	for i := 1; i < len(conditions); i++ {
		query += " AND " + conditions[i]
	}

	_, err := DB.Exec(query, args...)
	return err
}

func UpdateJobState(id int64, state string) error {
	query := `UPDATE jobs SET last_known_state = ?, last_state_at_utc = ? WHERE id = ?`
	_, err := DB.Exec(query, state, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func scanJob(row *sql.Row) (*Job, error) {
	var j Job
	err := row.Scan(
		&j.ID,
		&j.CreatedAtUTC,
		&j.Name,
		&j.Unit,
		&j.Cwd,
		&j.ArgvJSON,
		&j.EnvJSON,
		&j.PropertiesJSON,
		&j.Host,
		&j.User,
		&j.Notes,
		&j.LastKnownState,
		&j.LastStateAtUTC,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &j, err
}

func scanJobRows(rows *sql.Rows) (*Job, error) {
	var j Job
	err := rows.Scan(
		&j.ID,
		&j.CreatedAtUTC,
		&j.Name,
		&j.Unit,
		&j.Cwd,
		&j.ArgvJSON,
		&j.EnvJSON,
		&j.PropertiesJSON,
		&j.Host,
		&j.User,
		&j.Notes,
		&j.LastKnownState,
		&j.LastStateAtUTC,
	)
	return &j, err
}
