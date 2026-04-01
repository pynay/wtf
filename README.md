# wtf

A CLI tool that explains your last failed terminal command using an LLM. Run `wtf` after a command fails and get a concise explanation of what went wrong, how to fix it, and why. Optionally hand off to an AI coding agent to fix the issue automatically.

## How it works

1. A shell hook captures every failed command and its stderr output to temp files (no re-execution — stderr is captured on the original run via fd redirection)
2. `wtf` reads those files, gathers system context (OS, shell, project type), redacts secrets, and sends it to an LLM
3. The explanation streams to your terminal in real time
4. With `wtf fix`, the explanation is piped to a coding agent (e.g., Claude Code) to apply the fix

## Setup

### 1. Build and install the binary

```bash
go build -o wtf .
```

To run `wtf` from anywhere, move it to a directory on your PATH:

```bash
mv wtf ~/.local/bin/
```

If `~/.local/bin` doesn't exist or isn't in your PATH:

```bash
mkdir -p ~/.local/bin
mv wtf ~/.local/bin/
# Add to your shell config (e.g., ~/.zshrc or ~/.bashrc):
export PATH="$HOME/.local/bin:$PATH"
```

### 2. Install the shell hook

```bash
bash setup.sh
```

This does three things:
- Adds a preexec/precmd hook to your `.zshrc` or `.bashrc` that captures failed commands automatically
- Prompts you to choose a fix mode (oneshot or interactive) for `wtf fix`
- Shows you how to configure your API key

Restart your terminal or `source` your shell config to activate the hook.

### 3. Configure your API key

**Option 1: Config file (recommended — persists across restarts)**

```bash
mkdir -p ~/.config/wtf
echo 'your-api-key' > ~/.config/wtf/api_key
```

**Option 2: Environment variable (add to your shell config to persist)**

```bash
export OPENAI_API_KEY=your-key-here
```

### 4. Configure fix mode

During setup you'll be prompted to choose, but you can change it anytime:

```bash
# oneshot — Claude fixes the issue and exits (default)
echo 'oneshot' > ~/.config/wtf/fix_mode

# interactive — Claude opens a live session with the error context
echo 'interactive' > ~/.config/wtf/fix_mode
```

## Usage

```bash
# Run a command that fails
$ gcc foo.c
foo.c:3:10: fatal error: 'bar.h' file not found

# Ask wtf happened
$ wtf

# Ask wtf happened AND have an AI agent fix it
$ wtf fix
```

### Fix modes

- **oneshot** (default) — runs `claude -p` with the error context, prints the fix, exits
- **interactive** — runs `claude -p --continue` so you get a live session to discuss and iterate on the fix

## Config files

All configuration lives in `~/.config/wtf/`:

| File | Purpose |
|------|---------|
| `api_key` | Your OpenAI API key |
| `fix_mode` | `oneshot` or `interactive` |
| `agent` | Coding agent CLI command (default: `claude`) |

## Using a different coding agent

The agent is configurable during setup, or anytime by editing `~/.config/wtf/agent`:

```bash
# Use Claude Code (default)
echo 'claude' > ~/.config/wtf/agent

# Use OpenAI Codex CLI
echo 'codex' > ~/.config/wtf/agent

# Use any custom agent
echo 'my-agent' > ~/.config/wtf/agent
```

The agent must have a CLI that accepts:
- `-p "prompt"` for oneshot mode (processes and exits)
- `-p "prompt" --continue` for interactive mode (stays open for follow-up)

## Requirements

- Go 1.21+
- bash or zsh
- An OpenAI API key
- A coding agent CLI like [Claude Code](https://docs.anthropic.com/en/docs/claude-code) or Codex (for `wtf fix`)
