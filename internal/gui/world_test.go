package gui

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/danielriddell21/vivarium/internal/sim"
)

// captureStdout runs fn with os.Stdout redirected, and returns what it wrote.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	saved := os.Stdout
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	fn()
	os.Stdout = saved
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe: %v", err)
	}
	return <-done
}

func TestBuildWorldPrintConfig(t *testing.T) {
	var (
		world *sim.World
		err   error
	)
	out := captureStdout(t, func() { world, err = buildWorld(Config{PrintConfig: true}) })

	if err != nil {
		t.Fatalf("buildWorld: %v", err)
	}
	if world != nil {
		t.Error("printing the config built a world; there is nothing left to run")
	}
	if !json.Valid([]byte(out)) {
		t.Errorf("stdout is not valid JSON:\n%s", out)
	}
	if !strings.Contains(out, `"width"`) {
		t.Errorf("printed config is missing the world size:\n%s", out)
	}
}

func TestRunPrintConfigSucceedsWithoutAWindow(t *testing.T) {
	// --print-config asks for output, not a window, so the GUI-less build must
	// print it and exit cleanly rather than complaining about the missing GUI.
	var err error
	captureStdout(t, func() { err = Run(Config{PrintConfig: true}) })
	if err != nil {
		t.Errorf("Run(--print-config) = %v, want nil", err)
	}
}

func TestBuildWorldWithoutAChangedHook(t *testing.T) {
	// The demo generator sets no flags, so Config.Changed is nil. Defaults apply.
	w, err := buildWorld(Config{Seed: 5})
	if err != nil {
		t.Fatalf("buildWorld: %v", err)
	}
	if w == nil {
		t.Fatal("buildWorld returned no world")
	}
	if w.W <= 0 || w.H <= 0 {
		t.Errorf("world is %vx%v, want the default size", w.W, w.H)
	}
}
