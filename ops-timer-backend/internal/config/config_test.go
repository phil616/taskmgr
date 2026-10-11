package config

import (
	"os"
	"testing"
)

func TestInitialAdminConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("TASK_MANAGER_INIT_ADMIN_USERNAME", "")
	t.Setenv("TASK_MANAGER_INIT_ADMIN_PASSWORD", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InitAdmin.Username != "admin" || cfg.InitAdmin.Password != "Admin@12345" {
		t.Fatal("incorrect admin defaults")
	}
	// Remove the empty entries to allow godotenv to load values from the file.
	os.Unsetenv("TASK_MANAGER_INIT_ADMIN_USERNAME")
	os.Unsetenv("TASK_MANAGER_INIT_ADMIN_PASSWORD")
	if err := os.WriteFile(".env", []byte("TASK_MANAGER_INIT_ADMIN_USERNAME=fileadmin\nTASK_MANAGER_INIT_ADMIN_PASSWORD=FilePassword123\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InitAdmin.Username != "fileadmin" || cfg.InitAdmin.Password != "FilePassword123" {
		t.Fatal("initial admin settings were not loaded from .env")
	}
	t.Setenv("TASK_MANAGER_INIT_ADMIN_USERNAME", "envadmin")
	t.Setenv("TASK_MANAGER_INIT_ADMIN_PASSWORD", "EnvPassword123")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InitAdmin.Username != "envadmin" || cfg.InitAdmin.Password != "EnvPassword123" {
		t.Fatal("environment must override .env")
	}
}
