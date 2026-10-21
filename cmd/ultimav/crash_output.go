package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

const (
	crashLogDirectory         = ".ultima_v_redux"
	crashLogFilename          = "crash.log"
	maxCrashLogBytes          = int64(4 << 20)
	diagnosticsLogFilename    = "redux.log"
	maxDiagnosticsLogBytes    = int64(1 << 20)
	crashDirectoryPermissions = os.FileMode(0o700)
	crashFilePermissions      = os.FileMode(0o600)
	buildSettingTrue          = "true"
)

func configureCrashOutput() {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to determine crash log directory: %v\n", err)

		return
	}

	path := filepath.Join(homeDirectory, crashLogDirectory, crashLogFilename)
	if err := configureCrashOutputAt(path); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to configure crash log %q: %v\n", path, err)
	}
	path = filepath.Join(homeDirectory, crashLogDirectory, diagnosticsLogFilename)
	if err := configureDiagnosticsLogAt(path); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to configure diagnostics log %q: %v\n", path, err)
	}
}

// configureDiagnosticsLogAt keeps the file open for the process so every
// warning and error reaches it.
func configureDiagnosticsLogAt(path string) error {
	file, err := openDiagnosticFile(path, maxDiagnosticsLogBytes)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprint(file, crashLaunchHeader()); err != nil {
		return closeCrashLog(file, fmt.Errorf("writing diagnostics log header: %w", err))
	}
	logging.SetDiagnosticsWriter(file)

	return nil
}

func openDiagnosticFile(path string, maxBytes int64) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), crashDirectoryPermissions); err != nil {
		return nil, fmt.Errorf("creating diagnostics directory: %w", err)
	}
	if err := rotateCrashLog(path, maxBytes); err != nil {
		return nil, err
	}

	file, err := os.OpenFile( //nolint:gosec // Path is the application-owned diagnostics path or a test temp path.
		path,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		crashFilePermissions,
	)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", filepath.Base(path), err)
	}
	if err := file.Chmod(crashFilePermissions); err != nil {
		return nil, closeCrashLog(file, fmt.Errorf("securing %s: %w", filepath.Base(path), err))
	}

	return file, nil
}

func configureCrashOutputAt(path string) error {
	file, err := openDiagnosticFile(path, maxCrashLogBytes)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprint(file, crashLaunchHeader()); err != nil {
		return closeCrashLog(file, fmt.Errorf("writing crash log header: %w", err))
	}
	if err := debug.SetCrashOutput(file, debug.CrashOptions{}); err != nil {
		return closeCrashLog(file, fmt.Errorf("registering crash output: %w", err))
	}

	return closeCrashLog(file, nil)
}

func closeCrashLog(file *os.File, operationErr error) error {
	closeErr := file.Close()
	if operationErr != nil {
		return operationErr
	}
	if closeErr != nil {
		return fmt.Errorf("closing crash log: %w", closeErr)
	}

	return nil
}

func rotateCrashLog(path string, maxBytes int64) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("checking crash log size: %w", err)
	}
	if info.Size() < maxBytes {
		return nil
	}

	oldPath := path + ".old"
	if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing previous crash log: %w", err)
	}
	if err := os.Rename(path, oldPath); err != nil {
		return fmt.Errorf("rotating crash log: %w", err)
	}

	return nil
}

func crashLaunchHeader() string {
	revision := ""
	dirty := false
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range buildInfo.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value
			case "vcs.modified":
				dirty = setting.Value == buildSettingTrue
			}
		}
	}

	return fmt.Sprintf(
		"\n=== Ultima V Redux launch %s pid=%d os=%s arch=%s version=%s revision=%s dirty=%t ===\n",
		time.Now().Format(time.RFC3339),
		os.Getpid(),
		runtime.GOOS,
		runtime.GOARCH,
		runtimeGameVersion(),
		revision,
		dirty,
	)
}
