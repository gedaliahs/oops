package cmd

import (
	"strings"
	"testing"
)

func TestLinuxServiceUnitQuotesExecutablePath(t *testing.T) {
	unit := linuxServiceUnit(`/tmp/oops dir/oops"; touch /tmp/pwn #`)
	if strings.Contains(unit, `touch /tmp/pwn`) && !strings.Contains(unit, `\"; touch /tmp/pwn #`) {
		t.Fatalf("expected shell metacharacters to be quoted, got:\n%s", unit)
	}
	if !strings.Contains(unit, `ExecStart="/tmp/oops dir/oops\"; touch /tmp/pwn #" clean`) {
		t.Fatalf("unexpected ExecStart quoting:\n%s", unit)
	}
}

func TestMacLaunchAgentPlistEscapesExecutablePath(t *testing.T) {
	plist := macLaunchAgentPlist(`/tmp/oops & dir/<oops>"`)
	if strings.Contains(plist, `/tmp/oops & dir/<oops>"`) {
		t.Fatalf("expected XML metacharacters to be escaped, got:\n%s", plist)
	}
	for _, want := range []string{`/tmp/oops &amp; dir/`, `&lt;oops&gt;&#34;`} {
		if !strings.Contains(plist, want) {
			t.Fatalf("expected %q in plist:\n%s", want, plist)
		}
	}
}
