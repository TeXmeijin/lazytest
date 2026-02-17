package runner

import (
	"testing"

	"github.com/meijin/lazytest/internal/config"
)

func TestBuildCommandFiles(t *testing.T) {
	e := NewExecutor(config.Config{
		Targets: []config.Target{
			{Name: "phpunit", Command: "phpunit --teamcity {files}"},
		},
	})

	cmd := e.BuildCommand("phpunit", []string{"tests/FooTest.php", "tests/BarTest.php"})
	expected := "phpunit --teamcity tests/FooTest.php tests/BarTest.php"
	if cmd != expected {
		t.Errorf("got %q, want %q", cmd, expected)
	}
}

func TestBuildCommandFile(t *testing.T) {
	e := NewExecutor(config.Config{
		Targets: []config.Target{
			{Name: "phpunit", Command: "phpunit --teamcity {file}"},
		},
	})

	cmd := e.BuildCommand("phpunit", []string{"tests/FooTest.php"})
	expected := "phpunit --teamcity tests/FooTest.php"
	if cmd != expected {
		t.Errorf("got %q, want %q", cmd, expected)
	}
}

func TestBuildCommandStripPrefix(t *testing.T) {
	e := NewExecutor(config.Config{
		Targets: []config.Target{
			{
				Name:            "phpunit",
				Command:         "docker exec php phpunit --teamcity {files}",
				PathStripPrefix: "src/",
			},
		},
	})

	cmd := e.BuildCommand("phpunit", []string{"src/tests/FooTest.php", "src/tests/BarTest.php"})
	expected := "docker exec php phpunit --teamcity tests/FooTest.php tests/BarTest.php"
	if cmd != expected {
		t.Errorf("got %q, want %q", cmd, expected)
	}
}

func TestBuildCommandNoPrefix(t *testing.T) {
	e := NewExecutor(config.Config{
		Targets: []config.Target{
			{Name: "phpunit", Command: "phpunit {files}"},
		},
	})

	cmd := e.BuildCommand("phpunit", []string{"tests/FooTest.php"})
	expected := "phpunit tests/FooTest.php"
	if cmd != expected {
		t.Errorf("got %q, want %q", cmd, expected)
	}
}

func TestBuildCommandEmptyFiles(t *testing.T) {
	e := NewExecutor(config.Config{
		Targets: []config.Target{
			{Name: "phpunit", Command: "phpunit --teamcity {files}"},
		},
	})

	cmd := e.BuildCommand("phpunit", nil)
	expected := "phpunit --teamcity "
	if cmd != expected {
		t.Errorf("got %q, want %q", cmd, expected)
	}
}

func TestBuildCommandVitest(t *testing.T) {
	e := NewExecutor(config.Config{
		Targets: []config.Target{
			{Name: "vitest", Command: "npx vitest run --reporter=teamcity {files}"},
		},
	})

	cmd := e.BuildCommand("vitest", []string{"src/App.test.ts", "src/Page.test.tsx"})
	expected := "npx vitest run --reporter=teamcity src/App.test.ts src/Page.test.tsx"
	if cmd != expected {
		t.Errorf("got %q, want %q", cmd, expected)
	}
}

func TestBuildCommandUnknownTarget(t *testing.T) {
	e := NewExecutor(config.Config{
		Targets: []config.Target{
			{Name: "phpunit", Command: "phpunit {files}"},
		},
	})

	cmd := e.BuildCommand("unknown", []string{"test.php"})
	if cmd != "" {
		t.Errorf("got %q, want empty string for unknown target", cmd)
	}
}

func TestBuildCommandMultipleTargets(t *testing.T) {
	e := NewExecutor(config.Config{
		Targets: []config.Target{
			{Name: "phpunit", Command: "phpunit --teamcity {files}"},
			{Name: "vitest", Command: "npx vitest run --reporter=teamcity {files}"},
		},
	})

	phpCmd := e.BuildCommand("phpunit", []string{"tests/FooTest.php"})
	vtCmd := e.BuildCommand("vitest", []string{"src/App.test.ts"})

	if phpCmd != "phpunit --teamcity tests/FooTest.php" {
		t.Errorf("phpunit cmd = %q", phpCmd)
	}
	if vtCmd != "npx vitest run --reporter=teamcity src/App.test.ts" {
		t.Errorf("vitest cmd = %q", vtCmd)
	}
}
