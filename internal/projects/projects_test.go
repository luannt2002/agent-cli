package projects

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestRegisterAndList(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Override the projects file path for testing
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(tmpDir, "crush"))

	// Test registering a project
	err := Register("/home/user/project1", "/home/user/project1/.crush")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// List projects
	projects, err := List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(projects))
	}

	if projects[0].Path != "/home/user/project1" {
		t.Errorf("Expected path /home/user/project1, got %s", projects[0].Path)
	}

	if projects[0].DataDir != "/home/user/project1/.crush" {
		t.Errorf("Expected data_dir /home/user/project1/.crush, got %s", projects[0].DataDir)
	}

	// Register another project
	err = Register("/home/user/project2", "/home/user/project2/.crush")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	projects, err = List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(projects) != 2 {
		t.Fatalf("Expected 2 projects, got %d", len(projects))
	}

	// Most recent should be first
	if projects[0].Path != "/home/user/project2" {
		t.Errorf("Expected most recent project first, got %s", projects[0].Path)
	}
}

func TestRegisterUpdatesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(tmpDir, "crush"))

	// Register a project
	err := Register("/home/user/project1", "/home/user/project1/.crush")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	projects, _ := List()
	firstAccess := projects[0].LastAccessed

	// Wait a bit and re-register
	time.Sleep(10 * time.Millisecond)

	err = Register("/home/user/project1", "/home/user/project1/.crush-new")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	projects, _ = List()

	if len(projects) != 1 {
		t.Fatalf("Expected 1 project after update, got %d", len(projects))
	}

	if projects[0].DataDir != "/home/user/project1/.crush-new" {
		t.Errorf("Expected updated data_dir, got %s", projects[0].DataDir)
	}

	if !projects[0].LastAccessed.After(firstAccess) {
		t.Error("Expected LastAccessed to be updated")
	}
}

func TestLoadEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(tmpDir, "crush"))

	// List before any projects exist
	projects, err := List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(projects) != 0 {
		t.Errorf("Expected 0 projects, got %d", len(projects))
	}
}

func TestProjectsFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(tmpDir, "crush"))

	expected := filepath.Join(tmpDir, "crush", "projects.json")
	actual := projectsFilePath()

	if actual != expected {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}

func TestRegisterWithParentDataDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(tmpDir, "crush"))

	// Register a project where .crush is in a parent directory.
	// e.g., working in /home/user/monorepo/packages/app but .crush is at /home/user/monorepo/.crush
	err := Register("/home/user/monorepo/packages/app", "/home/user/monorepo/.crush")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	projects, err := List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(projects))
	}

	if projects[0].Path != "/home/user/monorepo/packages/app" {
		t.Errorf("Expected path /home/user/monorepo/packages/app, got %s", projects[0].Path)
	}

	if projects[0].DataDir != "/home/user/monorepo/.crush" {
		t.Errorf("Expected data_dir /home/user/monorepo/.crush, got %s", projects[0].DataDir)
	}
}

func TestRegisterWithExternalDataDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(tmpDir, "crush"))

	// Register a project where .crush is in a completely different location.
	// e.g., project at /home/user/project but data stored at /var/data/crush/myproject
	err := Register("/home/user/project", "/var/data/crush/myproject")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	projects, err := List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(projects))
	}

	if projects[0].Path != "/home/user/project" {
		t.Errorf("Expected path /home/user/project, got %s", projects[0].Path)
	}

	if projects[0].DataDir != "/var/data/crush/myproject" {
		t.Errorf("Expected data_dir /var/data/crush/myproject, got %s", projects[0].DataDir)
	}
}

func TestRegisterOrdersByRecency(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(tmpDir, "crush"))

	for _, path := range []string{"/p1", "/p2", "/p3", "/p1"} {
		if err := Register(path, path+"/.crush"); err != nil {
			t.Fatalf("Register failed: %v", err)
		}
	}

	projects, err := List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	var got []string
	for _, p := range projects {
		got = append(got, p.Path)
	}
	want := []string{"/p1", "/p3", "/p2"}
	if !slices.Equal(got, want) {
		t.Fatalf("Expected order %v, got %v", want, got)
	}
}

// Worktree của `dew routine run` nằm dưới <data dir>/routines: không được vào
// sổ dự án (đẩy dự án thật khỏi Home), và các mục đã lọt vào từ trước phải
// bị ẩn khi liệt kê.
func TestRoutineWorktreesAreNotProjects(t *testing.T) {
	tmpDir := realDir(t, t.TempDir()) // macOS: /var là symlink, sổ lưu dạng đã resolve
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(tmpDir, "crush"))

	wt := filepath.Join(tmpDir, "crush", "routines", "nightly", "wt", "20261021T010203")
	real := filepath.Join(tmpDir, "work", "app")
	// Thư mục anh em có tiền tố giống "routines" vẫn là dự án thường.
	sibling := filepath.Join(tmpDir, "crush", "routines-notes")
	for _, d := range []string{wt, real, sibling} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Register bỏ qua worktree routine.
	for _, d := range []string{wt, real, sibling} {
		if err := Register(d, filepath.Join(d, ".crush")); err != nil {
			t.Fatalf("Register(%s): %v", d, err)
		}
	}
	got, err := List()
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, p := range got {
		paths = append(paths, p.Path)
	}
	if want := []string{sibling, real}; !slices.Equal(paths, want) {
		t.Fatalf("after Register: got %v, want %v", paths, want)
	}

	// Register không ghi gì vào file (List một mình cũng ẩn được, nên nhìn thẳng file).
	if raw, err := Load(); err != nil || len(raw.Projects) != 2 {
		t.Fatalf("registry file after Register: %+v, err %v; want 2 entries", raw, err)
	}

	// Mục có sẵn trong projects.json (từ bản cũ) bị ẩn khi List.
	seeded := &ProjectList{Projects: []Project{
		{Path: wt, DataDir: wt + "/.crush", Host: currentHost()},
		{Path: real, DataDir: real + "/.crush", Host: currentHost()},
	}}
	if err := Save(seeded); err != nil {
		t.Fatal(err)
	}
	got, err = List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Path != real {
		t.Fatalf("after seeded file: got %+v, want only %s", got, real)
	}
}
