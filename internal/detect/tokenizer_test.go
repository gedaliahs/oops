package detect

import (
	"reflect"
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"rm file.txt", []string{"rm", "file.txt"}},
		{"rm -rf dir/", []string{"rm", "-rf", "dir/"}},
		{`rm "file with spaces.txt"`, []string{"rm", "file with spaces.txt"}},
		{`rm 'file with spaces.txt'`, []string{"rm", "file with spaces.txt"}},
		{`rm file\ with\ spaces.txt`, []string{"rm", "file with spaces.txt"}},
		{"echo hi | rm file", []string{"echo", "hi", "|", "rm", "file"}},
		{"rm a && rm b", []string{"rm", "a", "&&", "rm", "b"}},
		{"rm a; rm b", []string{"rm", "a", ";", "rm", "b"}},
		{"echo hi > file.txt", []string{"echo", "hi", ">", "file.txt"}},
		{"echo hi >> file.txt", []string{"echo", "hi", ">>", "file.txt"}},
		{"cat file 2>/dev/null", []string{"cat", "file", "2>", "/dev/null"}},
		{`FOO=bar rm file`, []string{"FOO=bar", "rm", "file"}},
		{"", nil},
		{"   ", nil},
		{`git reset --hard HEAD~1`, []string{"git", "reset", "--hard", "HEAD~1"}},
		{`sed -i 's/old/new/g' file.txt`, []string{"sed", "-i", "s/old/new/g", "file.txt"}},
	}

	for _, tt := range tests {
		got := Tokenize(tt.input)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Tokenize(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestSplitPipeline(t *testing.T) {
	tokens := []string{"echo", "hi", "|", "rm", "file"}
	got := SplitPipeline(tokens)
	want := [][]string{{"echo", "hi"}, {"rm", "file"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SplitPipeline = %v, want %v", got, want)
	}
}

func TestSplitCommands(t *testing.T) {
	tokens := []string{"rm", "a", "&&", "rm", "b", ";", "echo", "done"}
	got := SplitCommands(tokens)
	want := [][]string{{"rm", "a"}, {"rm", "b"}, {"echo", "done"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SplitCommands = %v, want %v", got, want)
	}
}
