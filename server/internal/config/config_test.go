package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	path := filepath.Join("..", "..", "config", "config.yaml")
	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{"development without a database", map[string]string{"APP_ENV": "development"}, false},
		{"production without a database", map[string]string{"APP_ENV": "production"}, true},
		{"production with a database", map[string]string{"APP_ENV": "production", "DATABASE_URL": "postgres://x"}, false},
		{"bad port", map[string]string{"PORT": "eighty"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{"APP_ENV", "DATABASE_URL", "PORT", "WEBSITE_URL", "EXTENSION_ORIGINS"} {
				t.Setenv(key, tt.env[key])
			}
			cfg, err := Load(path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && (cfg.Trial.Voice != 5 || cfg.Trial.Checks != 300 || cfg.Licence.MaxInstalls != 5) {
				t.Fatalf("unexpected defaults: %+v %+v", cfg.Trial, cfg.Licence)
			}
		})
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("PORT", "9000")
	t.Setenv("WEBSITE_URL", "https://fuckalgo.app/")
	t.Setenv("EXTENSION_ORIGINS", " chrome-extension://abc/ , ,chrome-extension://def")
	cfg, err := Load(filepath.Join("..", "..", "config", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 9000 || cfg.WebsiteURL != "https://fuckalgo.app" || len(cfg.ExtensionOrigins) != 2 || cfg.ExtensionOrigins[0] != "chrome-extension://abc" {
		t.Fatalf("overrides not applied: %d %q %q", cfg.Server.Port, cfg.WebsiteURL, cfg.ExtensionOrigins)
	}
}

func TestLoadDotEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := "# comment\nFA_TEST_A=one\nexport FA_TEST_B=\"two words\"\nFA_TEST_C=already\nnot a pair\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FA_TEST_A", "")
	os.Unsetenv("FA_TEST_A")
	t.Setenv("FA_TEST_B", "")
	os.Unsetenv("FA_TEST_B")
	t.Setenv("FA_TEST_C", "kept")
	if !LoadDotEnv(path) || LoadDotEnv(path+".missing") {
		t.Fatal("LoadDotEnv reported the wrong files")
	}
	if os.Getenv("FA_TEST_A") != "one" || os.Getenv("FA_TEST_B") != "two words" || os.Getenv("FA_TEST_C") != "kept" {
		t.Fatalf("got %q %q %q", os.Getenv("FA_TEST_A"), os.Getenv("FA_TEST_B"), os.Getenv("FA_TEST_C"))
	}
}
