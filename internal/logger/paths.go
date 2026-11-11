package logger

import (
	"os"
	"path/filepath"
	"runtime"
)

// DefaultFilePath returns the per-user log file path for the named binary:
//
//	macOS:   ~/Library/Logs/Orbit/<name>.log
//	Windows: %LocalAppData%\Orbit\logs\<name>.log
//	Linux:   $XDG_STATE_HOME/orbit/<name>.log (default ~/.local/state/orbit/<name>.log)
//
// A daemon running as a system service will need a system path instead
// (e.g. /var/log/orbit); that is not handled yet.
func DefaultFilePath(name string) (string, error) {
	file := name + ".log"

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Logs", "Orbit", file), nil
	case "windows":
		dir, err := os.UserCacheDir() // %LocalAppData%
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, "Orbit", "logs", file), nil
	default:
		if dir := os.Getenv("XDG_STATE_HOME"); filepath.IsAbs(dir) {
			return filepath.Join(dir, "orbit", file), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "state", "orbit", file), nil
	}
}
