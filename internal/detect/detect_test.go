package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyze_RM(t *testing.T) {
	// Create temp file
	tmp := t.TempDir()
	f := filepath.Join(tmp, "test.txt")
	os.WriteFile(f, []byte("hello"), 0o644)

	prots := Analyze("rm " + f)
	if len(prots) == 0 {
		t.Fatal("expected protection for rm")
	}
	if prots[0].Action != ActionRM {
		t.Errorf("expected ActionRM, got %s", prots[0].Action)
	}
	if len(prots[0].Files) != 1 || prots[0].Files[0] != f {
		t.Errorf("expected file %s, got %v", f, prots[0].Files)
	}
}

func TestAnalyze_RM_RF_HighRisk(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "mydir")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644)

	prots := Analyze("rm -rf " + dir)
	if len(prots) == 0 {
		t.Fatal("expected protection for rm -rf")
	}
	if prots[0].Risk != RiskHigh {
		t.Errorf("expected RiskHigh, got %s", prots[0].Risk)
	}
}

func TestAnalyze_MV_Overwrite(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "dst.txt")
	os.WriteFile(src, []byte("new"), 0o644)
	os.WriteFile(dst, []byte("old"), 0o644)

	prots := Analyze("mv " + src + " " + dst)
	if len(prots) == 0 {
		t.Fatal("expected protection for mv overwrite")
	}
	if prots[0].Action != ActionMV {
		t.Errorf("expected ActionMV, got %s", prots[0].Action)
	}
}

func TestAnalyze_MV_NoOverwrite(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "newname.txt")
	os.WriteFile(src, []byte("data"), 0o644)

	prots := Analyze("mv " + src + " " + dst)
	if len(prots) != 0 {
		t.Error("expected no protection when mv target doesn't exist")
	}
}

func TestAnalyze_CP_Overwrite(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "dst.txt")
	os.WriteFile(src, []byte("new"), 0o644)
	os.WriteFile(dst, []byte("old"), 0o644)

	prots := Analyze("cp " + src + " " + dst)
	if len(prots) == 0 {
		t.Fatal("expected protection for cp overwrite")
	}
	if prots[0].Action != ActionCP {
		t.Errorf("expected ActionCP, got %s", prots[0].Action)
	}
}

func TestAnalyze_SedInPlace(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "test.txt")
	os.WriteFile(f, []byte("hello world"), 0o644)

	prots := Analyze("sed -i 's/hello/bye/' " + f)
	if len(prots) == 0 {
		t.Fatal("expected protection for sed -i")
	}
	if prots[0].Action != ActionSed {
		t.Errorf("expected ActionSed, got %s", prots[0].Action)
	}
}

func TestAnalyze_Chmod(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "test.txt")
	os.WriteFile(f, []byte("data"), 0o644)

	prots := Analyze("chmod 777 " + f)
	if len(prots) == 0 {
		t.Fatal("expected protection for chmod")
	}
	if prots[0].Action != ActionChmod {
		t.Errorf("expected ActionChmod, got %s", prots[0].Action)
	}
}

func TestAnalyze_GitResetHard(t *testing.T) {
	prots := Analyze("git reset --hard")
	if len(prots) == 0 {
		t.Fatal("expected protection for git reset --hard")
	}
	if prots[0].GitAction != "stash" {
		t.Errorf("expected stash action, got %s", prots[0].GitAction)
	}
	if prots[0].Risk != RiskHigh {
		t.Errorf("expected RiskHigh, got %s", prots[0].Risk)
	}
}

func TestAnalyze_GitRestoreAndForceSwitch(t *testing.T) {
	cases := []string{
		"git restore .",
		"git checkout -f main",
		"git switch --discard-changes main",
	}
	for _, c := range cases {
		prots := Analyze(c)
		if len(prots) == 0 {
			t.Fatalf("expected protection for %q", c)
		}
		if prots[0].GitAction != "stash" {
			t.Errorf("expected stash action for %q, got %s", c, prots[0].GitAction)
		}
	}
}

func TestAnalyze_AdditionalDestructiveCommands(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "file.txt")
	dest := filepath.Join(tmp, "dest")
	os.WriteFile(file, []byte("data"), 0o644)
	os.MkdirAll(dest, 0o755)

	cases := map[string]ActionType{
		"dd if=/dev/zero of=" + file:             ActionDD,
		"find " + tmp + " -name '*.tmp' -delete": ActionFind,
		"rsync -a --delete source/ " + dest:      ActionRsync,
		"perl -pi -e 's/a/b/' " + file:           ActionPerl,
		"perl -i.bak -pe 's/a/b/' " + file:       ActionPerl,
	}
	for command, action := range cases {
		prots := Analyze(command)
		if len(prots) == 0 {
			t.Fatalf("expected protection for %q", command)
		}
		if prots[0].Action != action {
			t.Fatalf("expected %s for %q, got %s", action, command, prots[0].Action)
		}
	}
}

func TestAnalyze_BroadCleanupCommands(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	worktree := filepath.Join(tmp, "other-worktree")
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatal(err)
	}

	cases := map[string]ActionType{
		"find . -name '*.tmp' | xargs rm":         ActionXargs,
		"fd -e tmp -x rm -f":                      ActionFD,
		"fdfind . --exec-batch=rm":                ActionFD,
		"parallel rm -rf ::: build dist":          ActionParallel,
		"make clean":                              ActionMake,
		"make -C " + tmp + " distclean":           ActionMake,
		"npm run clean":                           ActionScript,
		"npm run-script clean:dist":               ActionScript,
		"yarn clean":                              ActionScript,
		"pnpm run clean":                          ActionScript,
		"git worktree remove " + worktree:         ActionGit,
		"git worktree remove --force " + worktree: ActionGit,
	}
	for command, action := range cases {
		prots := Analyze(command)
		if len(prots) == 0 {
			t.Fatalf("expected protection for %q", command)
		}
		if prots[0].Action != action {
			t.Fatalf("expected %s for %q, got %s", action, command, prots[0].Action)
		}
		if len(prots[0].Files) == 0 {
			t.Fatalf("expected files for %q", command)
		}
	}
}

func TestAnalyze_Redirect(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "out.txt")
	os.WriteFile(f, []byte("existing"), 0o644)

	prots := Analyze("echo hi > " + f)
	if len(prots) == 0 {
		t.Fatal("expected protection for redirect overwrite")
	}

	found := false
	for _, p := range prots {
		if p.Action == ActionRedirect {
			found = true
		}
	}
	if !found {
		t.Error("expected ActionRedirect in protections")
	}
}

func TestAnalyze_Safe(t *testing.T) {
	safe := []string{
		"ls -la",
		"cat file.txt",
		"echo hello",
		"cd /tmp",
		"grep -r foo .",
		"git status",
		"git log",
		"git diff",
	}
	for _, cmd := range safe {
		prots := Analyze(cmd)
		if len(prots) != 0 {
			t.Errorf("expected no protection for %q, got %v", cmd, prots)
		}
	}
}

func TestAnalyze_Sudo(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "test.txt")
	os.WriteFile(f, []byte("data"), 0o644)

	prots := Analyze("sudo rm " + f)
	if len(prots) == 0 {
		t.Fatal("expected protection for sudo rm")
	}
	if prots[0].Action != ActionRM {
		t.Errorf("expected ActionRM, got %s", prots[0].Action)
	}
}

func TestAnalyze_Pipeline(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "test.txt")
	os.WriteFile(f, []byte("data"), 0o644)

	// Direct rm in pipeline is detected
	prots := Analyze("echo foo | rm " + f)
	if len(prots) == 0 {
		t.Fatal("expected protection for rm in pipeline")
	}
}

func TestAnalyze_XargsPipeline(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "test.txt")
	os.WriteFile(f, []byte("data"), 0o644)

	prots := Analyze("find . -name '*.tmp' | xargs rm " + f)
	if len(prots) == 0 {
		t.Fatal("expected protection for xargs rm")
	}
	if prots[0].Action != ActionXargs {
		t.Fatalf("expected ActionXargs, got %s", prots[0].Action)
	}
}

func TestAnalyze_ChainedCommands(t *testing.T) {
	tmp := t.TempDir()
	f1 := filepath.Join(tmp, "a.txt")
	f2 := filepath.Join(tmp, "b.txt")
	os.WriteFile(f1, []byte("a"), 0o644)
	os.WriteFile(f2, []byte("b"), 0o644)

	prots := Analyze("rm " + f1 + " && rm " + f2)
	if len(prots) < 2 {
		t.Errorf("expected 2 protections, got %d", len(prots))
	}
}

func TestQuickMatch(t *testing.T) {
	matches := QuickMatch("rm -rf /tmp/foo")
	if len(matches) == 0 {
		t.Error("expected match for rm")
	}

	matches = QuickMatch("ls -la")
	if len(matches) != 0 {
		t.Error("expected no match for ls")
	}

	matches = QuickMatch("echo hi > file.txt")
	found := false
	for _, m := range matches {
		if m == ">" {
			found = true
		}
	}
	if !found {
		t.Error("expected redirect match")
	}
}
