package runner

import (
	"testing"

	"github.com/meijin/lazytest/internal/config"
)

func TestBuildCommandFiles(t *testing.T) {
	e := NewExecutor(config.Config{
		Command: "phpunit --teamcity {files}",
	})

	cmd := e.BuildCommand([]string{"tests/FooTest.php", "tests/BarTest.php"})
	expected := "phpunit --teamcity tests/FooTest.php tests/BarTest.php"
	if cmd != expected {
		t.Errorf("got %q, want %q", cmd, expected)
	}
}

func TestBuildCommandFile(t *testing.T) {
	e := NewExecutor(config.Config{
		Command: "phpunit --teamcity {file}",
	})

	cmd := e.BuildCommand([]string{"tests/FooTest.php"})
	expected := "phpunit --teamcity tests/FooTest.php"
	if cmd != expected {
		t.Errorf("got %q, want %q", cmd, expected)
	}
}

func TestBuildCommandStripPrefix(t *testing.T) {
	e := NewExecutor(config.Config{
		Command:         "docker exec php phpunit --teamcity {files}",
		PathStripPrefix: "src/",
	})

	cmd := e.BuildCommand([]string{"src/tests/FooTest.php", "src/tests/BarTest.php"})
	expected := "docker exec php phpunit --teamcity tests/FooTest.php tests/BarTest.php"
	if cmd != expected {
		t.Errorf("got %q, want %q", cmd, expected)
	}
}

func TestBuildCommandNoPrefix(t *testing.T) {
	e := NewExecutor(config.Config{
		Command: "phpunit {files}",
	})

	cmd := e.BuildCommand([]string{"tests/FooTest.php"})
	expected := "phpunit tests/FooTest.php"
	if cmd != expected {
		t.Errorf("got %q, want %q", cmd, expected)
	}
}

func TestBuildCommandEmptyFiles(t *testing.T) {
	e := NewExecutor(config.Config{
		Command: "phpunit --teamcity {files}",
	})

	cmd := e.BuildCommand(nil)
	expected := "phpunit --teamcity "
	if cmd != expected {
		t.Errorf("got %q, want %q", cmd, expected)
	}
}
