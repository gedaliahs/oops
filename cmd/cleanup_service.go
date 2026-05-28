package cmd

import (
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/gedaliah/oops/internal/style"
	"github.com/spf13/cobra"
)

const cleanupServiceName = "com.gedaliah.oops.cleanup"

var cleanupServiceCmd = &cobra.Command{
	Use:   "cleanup-service <install|uninstall|status>",
	Short: "Manage hourly background cleanup",
	Args:  cobra.ExactArgs(1),
	RunE:  runCleanupService,
}

func init() {
	rootCmd.AddCommand(cleanupServiceCmd)
}

func runCleanupService(cmd *cobra.Command, args []string) error {
	switch runtime.GOOS {
	case "darwin":
		return runMacCleanupService(args[0])
	case "linux":
		return runLinuxCleanupService(args[0])
	default:
		return fmt.Errorf("cleanup-service is not supported on %s", runtime.GOOS)
	}
}

func cleanupServiceStatus() (installed bool, loaded bool, label string, err error) {
	switch runtime.GOOS {
	case "darwin":
		path, err := macLaunchAgentPath()
		if err != nil {
			return false, false, "", err
		}
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return false, false, "cleanup launch agent", nil
			}
			return false, false, "", err
		}
		_, err = launchctl("print", macLaunchDomain()+"/"+cleanupServiceName)
		return true, err == nil, "cleanup launch agent", nil
	case "linux":
		_, timerPath, err := linuxSystemdPaths()
		if err != nil {
			return false, false, "", err
		}
		if _, err := os.Stat(timerPath); err != nil {
			if os.IsNotExist(err) {
				return false, false, "cleanup systemd timer", nil
			}
			return false, false, "", err
		}
		_, err = systemctl("--user", "is-active", "--quiet", "oops-cleanup.timer")
		return true, err == nil, "cleanup systemd timer", nil
	default:
		return false, false, "cleanup service", fmt.Errorf("cleanup-service is not supported on %s", runtime.GOOS)
	}
}

func runMacCleanupService(action string) error {
	path, err := macLaunchAgentPath()
	if err != nil {
		return err
	}
	switch action {
	case "install":
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(macLaunchAgentPlist(exe)), 0o644); err != nil {
			return err
		}
		launchctl("bootout", macLaunchDomain(), path)
		if out, err := launchctl("bootstrap", macLaunchDomain(), path); err != nil {
			fmt.Fprintln(os.Stderr, style.Warning("Installed launch agent, but launchctl did not load it: "+string(out)))
		}
		fmt.Println(style.Success("Installed hourly cleanup launch agent"))
	case "uninstall":
		launchctl("bootout", macLaunchDomain(), path)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		fmt.Println(style.Success("Removed cleanup launch agent"))
	case "status":
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				fmt.Println(style.Dim.Render("Cleanup launch agent is not installed."))
				return nil
			}
			return err
		}
		if _, err := launchctl("print", macLaunchDomain()+"/"+cleanupServiceName); err == nil {
			fmt.Println(style.Success("Cleanup launch agent is installed and loaded"))
		} else {
			fmt.Println(style.Warning("Cleanup launch agent is installed but not loaded"))
		}
	default:
		return fmt.Errorf("unknown cleanup-service action: %s", action)
	}
	return nil
}

func macLaunchAgentPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", cleanupServiceName+".plist"), nil
}

func macLaunchDomain() string {
	return fmt.Sprintf("gui/%d", os.Getuid())
}

func macLaunchAgentPlist(exe string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>%s</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
    <string>clean</string>
  </array>
  <key>StartInterval</key>
  <integer>3600</integer>
  <key>RunAtLoad</key>
  <true/>
</dict>
</plist>
`, cleanupServiceName, html.EscapeString(exe))
}

func launchctl(args ...string) ([]byte, error) {
	path, err := exec.LookPath("launchctl")
	if err != nil {
		return nil, err
	}
	return exec.Command(path, args...).CombinedOutput()
}

func runLinuxCleanupService(action string) error {
	servicePath, timerPath, err := linuxSystemdPaths()
	if err != nil {
		return err
	}
	switch action {
	case "install":
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(servicePath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(servicePath, []byte(linuxServiceUnit(exe)), 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(timerPath, []byte(linuxTimerUnit()), 0o644); err != nil {
			return err
		}
		systemctl("--user", "daemon-reload")
		if out, err := systemctl("--user", "enable", "--now", "oops-cleanup.timer"); err != nil {
			fmt.Fprintln(os.Stderr, style.Warning("Installed systemd timer, but systemctl did not enable it: "+string(out)))
		}
		fmt.Println(style.Success("Installed hourly cleanup systemd timer"))
	case "uninstall":
		systemctl("--user", "disable", "--now", "oops-cleanup.timer")
		_ = os.Remove(timerPath)
		_ = os.Remove(servicePath)
		systemctl("--user", "daemon-reload")
		fmt.Println(style.Success("Removed cleanup systemd timer"))
	case "status":
		if _, err := os.Stat(timerPath); err != nil {
			if os.IsNotExist(err) {
				fmt.Println(style.Dim.Render("Cleanup systemd timer is not installed."))
				return nil
			}
			return err
		}
		if _, err := systemctl("--user", "is-active", "--quiet", "oops-cleanup.timer"); err == nil {
			fmt.Println(style.Success("Cleanup systemd timer is installed and active"))
		} else {
			fmt.Println(style.Warning("Cleanup systemd timer is installed but inactive"))
		}
	default:
		return fmt.Errorf("unknown cleanup-service action: %s", action)
	}
	return nil
}

func linuxSystemdPaths() (string, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	dir := filepath.Join(home, ".config", "systemd", "user")
	return filepath.Join(dir, "oops-cleanup.service"), filepath.Join(dir, "oops-cleanup.timer"), nil
}

func linuxServiceUnit(exe string) string {
	return fmt.Sprintf(`[Unit]
Description=oops backup cleanup

[Service]
Type=oneshot
ExecStart=%q clean
`, exe)
}

func linuxTimerUnit() string {
	return `[Unit]
Description=Run oops cleanup hourly

[Timer]
OnBootSec=5m
OnUnitActiveSec=1h
Unit=oops-cleanup.service

[Install]
WantedBy=timers.target
`
}

func systemctl(args ...string) ([]byte, error) {
	path, err := exec.LookPath("systemctl")
	if err != nil {
		return nil, err
	}
	return exec.Command(path, args...).CombinedOutput()
}
