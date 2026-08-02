Output docs/assets/demo.gif

Require go
Require awk

Set Shell "bash"
Set Width 1200
Set Height 720
Set FontSize 22
Set TypingSpeed 20ms
Set Framerate 20
Set CursorBlink false
Set Padding 24
Set Theme { "name": "TokenSave", "black": "#45475a", "red": "#f38ba8", "green": "#a6e3a1", "yellow": "#f9e2af", "blue": "#89b4fa", "magenta": "#cba6f7", "cyan": "#94e2d5", "white": "#bac2de", "brightBlack": "#585b70", "brightRed": "#f38ba8", "brightGreen": "#a6e3a1", "brightYellow": "#f9e2af", "brightBlue": "#89b4fa", "brightMagenta": "#cba6f7", "brightCyan": "#94e2d5", "brightWhite": "#cdd6f4", "background": "#11111b", "foreground": "#cdd6f4", "selection": "#313244", "cursor": "#f5e0dc" }

Env TOKENSAVE_HOME ".tokensave/demo"

Hide
Type "rm -rf .tokensave/demo && mkdir -p .tokensave/demo/runs"
Enter
Type "go run ./scripts/demo-noisy.go >/dev/null 2>&1 || true"
Enter
Wait@30s
Type "clear"
Enter
Show

Type "echo 'TokenSave: keep the signal, keep the full log.'"
Enter
Sleep 800ms

Type "go run ./scripts/demo-noisy.go"
Enter
Sleep 1.8s

Ctrl+L
Type "echo 'The same command, now with TokenSave:'"
Enter
Type "bin/tokensave go run ./scripts/demo-noisy.go"
Enter
Sleep 4s

Type "bin/tokensave list --failed --limit 1"
Enter
Sleep 2s

Hide
Type "run_id=$(bin/tokensave list --failed --limit 1 | awk 'NR == 1 { print $1 }')"
Enter
Type "clear"
Enter
Show

Type "echo 'Inspect one saved failure:'"
Enter
Type `bin/tokensave show "$run_id" --failure 1`
Enter
Sleep 3s
