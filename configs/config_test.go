package configs

import (
	"os"
	"testing"
)

func cleanEnvironment(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	for _, key := range []string{"ENV", "DATABASE_URL", "PORT"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
}

func writeEnv(t *testing.T, content string) {
	t.Helper()
	if err := os.WriteFile(".env", []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadConfigLoadsDotEnvBeforeReadingValues(t *testing.T) {
	cleanEnvironment(t)
	writeEnv(t, "ENV=development\nDATABASE_URL=postgres://dotenv\nPORT=9000\n")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != DEVELOPMENT || cfg.DbURL != "postgres://dotenv" || cfg.Port != "9000" {
		t.Fatalf("unexpected configuration: %+v", cfg)
	}
}

func TestLoadConfigPreservesProcessEnvironment(t *testing.T) {
	cleanEnvironment(t)
	writeEnv(t, "DATABASE_URL=postgres://dotenv\nPORT=9000\n")
	t.Setenv("DATABASE_URL", "postgres://process")
	t.Setenv("PORT", "8001")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DbURL != "postgres://process" || cfg.Port != "8001" {
		t.Fatalf("process environment was overwritten: %+v", cfg)
	}
}

func TestLoadConfigProductionSkipsDotEnv(t *testing.T) {
	cleanEnvironment(t)
	writeEnv(t, "this is not a valid env file!")
	t.Setenv("ENV", PRODUCTION)
	t.Setenv("DATABASE_URL", "postgres://production")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != PRODUCTION || cfg.Port != "8000" {
		t.Fatalf("unexpected configuration: %+v", cfg)
	}
}

func TestLoadConfigDefaultsWithoutDotEnv(t *testing.T) {
	cleanEnvironment(t)
	t.Setenv("DATABASE_URL", "postgres://local")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != DEVELOPMENT || cfg.Port != "8000" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadConfigRejectsInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct{ name, key, value string }{
		{"missing database", "DATABASE_URL", ""},
		{"unknown environment", "ENV", "staging"},
		{"non-numeric port", "PORT", "http"},
		{"zero port", "PORT", "0"},
		{"negative port", "PORT", "-1"},
		{"out of range port", "PORT", "65536"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cleanEnvironment(t)
			t.Setenv("DATABASE_URL", "postgres://local")
			t.Setenv(tc.key, tc.value)
			if _, err := LoadConfig(); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

func TestLoadConfigRejectsMalformedDotEnv(t *testing.T) {
	cleanEnvironment(t)
	writeEnv(t, "INVALID KEY!=value\n")
	t.Setenv("DATABASE_URL", "postgres://local")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected .env parsing error")
	}
}
