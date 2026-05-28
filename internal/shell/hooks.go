package shell

import "fmt"

// ZshHook returns the zsh preexec hook script.
func ZshHook(oopsBin string) string {
	return fmt.Sprintf(`# oops — terminal undo (zsh hook)
_oops_preexec() {
  local cmd="$1"
  local output
  case "$cmd" in
    rm\ *|rm|mv\ *|cp\ *|sed\ *|gsed\ *|perl\ *-[pi]*|chmod\ *|chown\ *|truncate\ *|gtruncate\ *|dd\ *of=*|find\ *-delete*|rsync\ *--delete*|git\ reset*|git\ checkout*|git\ restore*|git\ switch*|git\ clean*|git\ branch\ *-[dD]*)
      output=$(%s protect -- "$cmd" 2>&1 1>/dev/null)
      if echo "$output" | grep -q "^OOPS_CONFIRM:"; then
        local desc="${output#OOPS_CONFIRM:}"
        printf "\033[0;33m%%s\033[0m Proceed? [Y/n] " "$desc"
        read -r reply
        case "$reply" in
          [nN]*) return 1 ;;
        esac
      elif [ -n "$output" ]; then
        echo "$output" >&2
      fi
      ;;
    *\>*)
      %s protect-redirect -- "$cmd"
      ;;
  esac
}
export OOPS_HOOK=1
autoload -Uz add-zsh-hook
add-zsh-hook preexec _oops_preexec
`, oopsBin, oopsBin)
}

// BashHook returns the bash DEBUG trap hook script.
func BashHook(oopsBin string) string {
	return fmt.Sprintf(`# oops — terminal undo (bash hook)
_oops_preexec() {
  [ -n "$COMP_LINE" ] && return
  [ "$BASH_COMMAND" = "$PROMPT_COMMAND" ] && return

  local cmd="$BASH_COMMAND"
  local output
  case "$cmd" in
    rm\ *|rm|mv\ *|cp\ *|sed\ *|gsed\ *|perl\ *-[pi]*|chmod\ *|chown\ *|truncate\ *|gtruncate\ *|dd\ *of=*|find\ *-delete*|rsync\ *--delete*|git\ reset*|git\ checkout*|git\ restore*|git\ switch*|git\ clean*|git\ branch\ *-[dD]*)
      output=$(%s protect -- "$cmd" 2>&1 1>/dev/null)
      if echo "$output" | grep -q "^OOPS_CONFIRM:"; then
        local desc="${output#OOPS_CONFIRM:}"
        printf "\033[0;33m%%s\033[0m Proceed? [Y/n] " "$desc"
        read -r reply
        case "$reply" in
          [nN]*) return 1 ;;
        esac
      elif [ -n "$output" ]; then
        echo "$output" >&2
      fi
      ;;
    *\>*)
      %s protect-redirect -- "$cmd"
      ;;
  esac
}
export OOPS_HOOK=1
trap '_oops_preexec' DEBUG
`, oopsBin, oopsBin)
}

// FishHook returns the fish preexec hook script.
func FishHook(oopsBin string) string {
	return fmt.Sprintf(`# oops — terminal undo (fish hook)
function _oops_preexec --on-event fish_preexec
  set -l cmd $argv[1]
  set -l output
  switch $cmd
    case 'rm *' 'mv *' 'cp *' 'sed *' 'gsed *' 'perl *-[pi]*' 'chmod *' 'chown *' 'truncate *' 'gtruncate *' 'dd *of=*' 'find *-delete*' 'rsync *--delete*' 'git reset*' 'git checkout*' 'git restore*' 'git switch*' 'git clean*' 'git branch *-D*' 'git branch *-d*'
      set output (%s protect -- "$cmd" 2>&1 1>/dev/null)
      if string match -q "OOPS_CONFIRM:*" -- $output
        set -l desc (string replace "OOPS_CONFIRM:" "" -- $output)
        read -P (set_color yellow)"$desc"(set_color normal)" Proceed? [Y/n] " reply
        if string match -qi "n*" -- $reply
          return 1
        end
      else if test -n "$output"
        echo $output >&2
      end
    case '*>*'
      %s protect-redirect -- "$cmd"
  end
end
set -gx OOPS_HOOK 1
`, oopsBin, oopsBin)
}
