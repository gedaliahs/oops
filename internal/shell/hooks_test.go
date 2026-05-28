package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func fakeOops(t *testing.T, dir string) (string, string) {
	t.Helper()
	logPath := filepath.Join(dir, "oops.log")
	binPath := filepath.Join(dir, "oops")
	script := "#!/usr/bin/env bash\nprintf '%s\\n' \"$*\" >> " + shellQuote(logPath) + "\n"
	if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return binPath, logPath
}

func TestBashHookInterceptsExpandedCommandSet(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	tmp := t.TempDir()
	fake, logPath := fakeOops(t, tmp)
	hookPath := filepath.Join(tmp, "hook.sh")
	if err := os.WriteFile(hookPath, []byte(BashHook(fake)), 0o644); err != nil {
		t.Fatal(err)
	}

	script := "source " + shellQuote(hookPath) + "\n" +
		"cp(){ :; }\n" +
		"dd(){ :; }\n" +
		"rsync(){ :; }\n" +
		"cp old new\n" +
		"dd if=/dev/zero of=file.img\n" +
		"rsync -a --delete src/ dest/\n"
	cmd := exec.Command("bash", "--noprofile", "--norc", "-c", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bash hook failed: %v\n%s", err, out)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	log := string(data)
	for _, want := range []string{
		"protect -- cp old new",
		"protect -- dd if=/dev/zero of=file.img",
		"protect -- rsync -a --delete src/ dest/",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("missing %q in hook log:\n%s", want, log)
		}
	}
}

func TestZshHookCanInvokeProtect(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available")
	}
	tmp := t.TempDir()
	fake, logPath := fakeOops(t, tmp)
	hookPath := filepath.Join(tmp, "hook.zsh")
	if err := os.WriteFile(hookPath, []byte(ZshHook(fake)), 0o644); err != nil {
		t.Fatal(err)
	}

	script := "source " + shellQuote(hookPath) + "\n_oops_preexec 'git restore .'\n_oops_preexec 'find . -name \"*.tmp\" -delete'\n_oops_preexec 'xargs rm -rf'\n_oops_preexec 'npm run clean'\n_oops_preexec 'git worktree remove ../old'\n"
	cmd := exec.Command("zsh", "-f", "-c", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("zsh hook failed: %v\n%s", err, out)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	log := string(data)
	for _, want := range []string{
		"protect -- git restore .",
		"protect -- find . -name \"*.tmp\" -delete",
		"protect -- xargs rm -rf",
		"protect -- npm run clean",
		"protect -- git worktree remove ../old",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("missing %q in hook log:\n%s", want, log)
		}
	}
}

func TestFishHookParsesWhenFishIsAvailable(t *testing.T) {
	if _, err := exec.LookPath("fish"); err != nil {
		t.Skip("fish not available")
	}
	tmp := t.TempDir()
	fake, logPath := fakeOops(t, tmp)
	hookPath := filepath.Join(tmp, "hook.fish")
	if err := os.WriteFile(hookPath, []byte(FishHook(fake)), 0o644); err != nil {
		t.Fatal(err)
	}

	script := "source " + shellQuote(hookPath) + "\nemit fish_preexec 'perl -pi -e s/a/b/ file.txt'\nemit fish_preexec 'fd -x rm {}'\n"
	cmd := exec.Command("fish", "-c", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fish hook failed: %v\n%s", err, out)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	log := string(data)
	if !strings.Contains(log, "protect -- perl -pi -e s/a/b/ file.txt") {
		t.Fatalf("unexpected fish hook log:\n%s", data)
	}
	if !strings.Contains(log, "protect -- fd -x rm {}") {
		t.Fatalf("unexpected fish hook log:\n%s", data)
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
