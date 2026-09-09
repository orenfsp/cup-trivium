package db

import (
	"strings"
	"testing"
)

func TestSplitStatements(t *testing.T) {
	got := splitStatements("  CREATE TABLE a (id INT);  \n\n INSERT INTO a VALUES (1); ;\n")
	if len(got) != 2 {
		t.Fatalf("splitStatements returned %d statements, want 2: %q", len(got), got)
	}
	if got[0] != "CREATE TABLE a (id INT)" {
		t.Errorf("got[0] = %q", got[0])
	}
	if got[1] != "INSERT INTO a VALUES (1)" {
		t.Errorf("got[1] = %q", got[1])
	}
}

func TestSplitStatementsEmpty(t *testing.T) {
	if got := splitStatements(" ; ; \n\t "); len(got) != 0 {
		t.Errorf("expected no statements, got %q", got)
	}
}

// Миграции зашиты в бинарник: они должны существовать, не дублироваться
// и быть непустыми — иначе чистый деплой не поднимет схему.
func TestEmbeddedMigrations(t *testing.T) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 3 {
		t.Fatalf("expected at least 3 migrations, got %d", len(entries))
	}
	seen := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() {
			t.Errorf("unexpected directory in migrations: %s", e.Name())
			continue
		}
		if !strings.HasSuffix(e.Name(), ".sql") {
			t.Errorf("unexpected non-sql file in migrations: %s", e.Name())
			continue
		}
		if seen[e.Name()] {
			t.Errorf("duplicate migration name: %s", e.Name())
		}
		seen[e.Name()] = true
		raw, err := migrationsFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			t.Fatal(err)
		}
		if len(raw) == 0 {
			t.Errorf("migration %s is empty", e.Name())
		}
	}
}
