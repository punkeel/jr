package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) func() {
	// Create a temporary directory for the test database
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "jr")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("Failed to create test data dir: %v", err)
	}

	// Set XDG_DATA_HOME to use the temp directory
	oldDataHome := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", dataDir)

	// Initialize the database
	if err := InitDB(); err != nil {
		t.Fatalf("Failed to init test DB: %v", err)
	}

	// Return cleanup function
	return func() {
		Close()
		os.Setenv("XDG_DATA_HOME", oldDataHome)
	}
}

func TestInitDB(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	if DB == nil {
		t.Fatal("DB should not be nil after InitDB")
	}

	// Verify tables were created by trying to insert
	err := DB.Ping()
	if err != nil {
		t.Fatalf("Failed to ping DB: %v", err)
	}
}

func TestCreateJob(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	argv := []string{"python", "train.py", "--epochs", "10"}
	env := map[string]string{"CUDA_VISIBLE_DEVICES": "0"}
	props := map[string]string{"MemoryMax": "4G"}

	id, err := CreateJob("test-job", "jr-test-20240101-120000-12345678.service", "/home/user", argv, env, props, "testhost", "testuser")
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	if id == 0 {
		t.Fatal("Expected non-zero job ID")
	}

	// Verify job was created
	job, err := GetJobByID(id)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if job == nil {
		t.Fatal("Job should not be nil")
	}

	if job.Name != "test-job" {
		t.Errorf("Expected Name='test-job', got %q", job.Name)
	}

	if job.Unit != "jr-test-20240101-120000-12345678.service" {
		t.Errorf("Expected Unit='jr-test-20240101-120000-12345678.service', got %q", job.Unit)
	}

	if job.Cwd != "/home/user" {
		t.Errorf("Expected Cwd='/home/user', got %q", job.Cwd)
	}
}

func TestGetJobByUnit(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	unit := "jr-test-unit.service"
	_, err := CreateJob("test", unit, "/tmp", []string{"echo", "hello"}, nil, nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	job, err := GetJobByUnit(unit)
	if err != nil {
		t.Fatalf("Failed to get job by unit: %v", err)
	}

	if job == nil {
		t.Fatal("Job should not be nil")
	}

	if job.Unit != unit {
		t.Errorf("Expected Unit=%q, got %q", unit, job.Unit)
	}
}

func TestGetJobByIDNotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	job, err := GetJobByID(99999)
	if err != nil {
		t.Fatalf("Should not error for non-existent job: %v", err)
	}

	if job != nil {
		t.Error("Expected nil for non-existent job")
	}
}

func TestListJobs(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create multiple jobs
	names := []string{"job-A", "job-B", "job-C", "job-D", "job-E"}
	for i, name := range names {
		_, err := CreateJob(
			name,
			"jr-job-"+string(rune('A'+i))+".service",
			"/tmp",
			[]string{"echo", string(rune('A' + i))},
			nil, nil, "", "",
		)
		if err != nil {
			t.Fatalf("Failed to create job %d: %v", i, err)
		}
		time.Sleep(1100 * time.Millisecond) // Ensure different timestamps (need at least 1 second for RFC3339)
	}

	// Test listing with limit
	jobs, err := ListJobs(3, false)
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}

	if len(jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(jobs))
	}

	// Test listing all
	jobs, err = ListJobs(0, true)
	if err != nil {
		t.Fatalf("Failed to list all jobs: %v", err)
	}

	if len(jobs) != 5 {
		t.Errorf("Expected 5 jobs, got %d", len(jobs))
	}

	// Verify order (most recent first)
	if jobs[0].Name != "job-E" {
		t.Errorf("Expected most recent job to be 'job-E', got %q", jobs[0].Name)
	}
}

func TestListJobsByName(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create jobs with different prefixes
	CreateJob("alpha-one", "jr-alpha-one.service", "/tmp", []string{"echo"}, nil, nil, "", "")
	CreateJob("alpha-two", "jr-alpha-two.service", "/tmp", []string{"echo"}, nil, nil, "", "")
	CreateJob("beta-one", "jr-beta-one.service", "/tmp", []string{"echo"}, nil, nil, "", "")

	jobs, err := ListJobsByName("alpha", 10)
	if err != nil {
		t.Fatalf("Failed to list jobs by name: %v", err)
	}

	if len(jobs) != 2 {
		t.Errorf("Expected 2 alpha jobs, got %d", len(jobs))
	}
}

func TestDeleteJob(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	id, err := CreateJob("delete-me", "jr-delete.service", "/tmp", []string{"echo"}, nil, nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	err = DeleteJob(id)
	if err != nil {
		t.Fatalf("Failed to delete job: %v", err)
	}

	job, err := GetJobByID(id)
	if err != nil {
		t.Fatalf("Should not error after delete: %v", err)
	}

	if job != nil {
		t.Error("Job should be nil after deletion")
	}
}

func TestUpdateJobState(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	id, err := CreateJob("state-test", "jr-state.service", "/tmp", []string{"echo"}, nil, nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	err = UpdateJobState(id, "running")
	if err != nil {
		t.Fatalf("Failed to update job state: %v", err)
	}

	job, err := GetJobByID(id)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if job.LastKnownState.String != "running" {
		t.Errorf("Expected state='running', got %q", job.LastKnownState.String)
	}

	if job.LastStateAtUTC.String == "" {
		t.Error("Expected LastStateAtUTC to be set")
	}
}

func TestPruneJobs(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create 5 jobs
	for i := 0; i < 5; i++ {
		_, err := CreateJob("prune-job", "jr-prune-"+string(rune('0'+i))+".service", "/tmp", []string{"echo"}, nil, nil, "", "")
		if err != nil {
			t.Fatalf("Failed to create job %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Prune keeping only 2
	err := PruneJobs(2, 0, false)
	if err != nil {
		t.Fatalf("Failed to prune jobs: %v", err)
	}

	jobs, err := ListJobs(0, true)
	if err != nil {
		t.Fatalf("Failed to list jobs after prune: %v", err)
	}

	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs after pruning, got %d", len(jobs))
	}
}

func TestFindJobByPartial(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	id, err := CreateJob("find-test", "jr-find.service", "/tmp", []string{"echo"}, nil, nil, "", "")
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Find by ID
	job, err := FindJobByPartial("1")
	if err != nil {
		t.Fatalf("Failed to find job by ID: %v", err)
	}
	if job == nil || job.ID != id {
		t.Error("Failed to find job by ID")
	}

	// Find by unit
	job, err = FindJobByPartial("jr-find.service")
	if err != nil {
		t.Fatalf("Failed to find job by unit: %v", err)
	}
	if job == nil || job.Unit != "jr-find.service" {
		t.Error("Failed to find job by unit")
	}

	// Find non-existent
	job, err = FindJobByPartial("nonexistent")
	if err != nil {
		t.Fatalf("Should not error for non-existent: %v", err)
	}
	if job != nil {
		t.Error("Expected nil for non-existent job")
	}
}
