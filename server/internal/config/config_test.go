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
		{"production without clerk", map[string]string{"APP_ENV": "production", "DATABASE_URL": "postgres://x"}, true},
		{"production without authorised parties", map[string]string{"APP_ENV": "production", "DATABASE_URL": "postgres://x", "CLERK_SECRET_KEY": "sk_test_x"}, true},
		{"production complete", map[string]string{"APP_ENV": "production", "DATABASE_URL": "postgres://x", "CLERK_SECRET_KEY": "sk_test_x", "CLERK_AUTHORIZED_PARTIES": "https://fuckalgo.app"}, false},
		{"bad port", map[string]string{"PORT": "eighty"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{"APP_ENV", "DATABASE_URL", "PORT", "WEBSITE_URL", "EXTENSION_ORIGINS", "CLERK_SECRET_KEY", "CLERK_AUTHORIZED_PARTIES"} {
				t.Setenv(key, tt.env[key])
			}
			cfg, err := Load(path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && (cfg.Trial.Voice != 5 || cfg.Trial.Checks != 300) {
				t.Fatalf("unexpected defaults: %+v", cfg.Trial)
			}
		})
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("PORT", "9000")
	t.Setenv("WEBSITE_URL", "https://fuckalgo.app/")
	t.Setenv("EXTENSION_ORIGINS", " chrome-extension://abc/ , ,chrome-extension://def")
	t.Setenv("CLERK_AUTHORIZED_PARTIES", "https://fuckalgo.app, chrome-extension://abc")
	cfg, err := Load(filepath.Join("..", "..", "config", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 9000 || cfg.WebsiteURL != "https://fuckalgo.app" || len(cfg.ExtensionOrigins) != 2 || cfg.ExtensionOrigins[0] != "chrome-extension://abc" {
		t.Fatalf("overrides not applied: %d %q %q", cfg.Server.Port, cfg.WebsiteURL, cfg.ExtensionOrigins)
	}
	if len(cfg.ClerkAuthorizedParties) != 2 || cfg.ClerkAuthorizedParties[1] != "chrome-extension://abc" {
		t.Fatalf("authorised parties = %q", cfg.ClerkAuthorizedParties)
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
