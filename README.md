# wtf

A CLI tool that explains your last failed terminal command using an LLM. Run `wtf` after a command fails and get a concise explanation of what went wrong, how to fix it, and why.

## How it works

1. A shell hook captures every failed command and its stderr output to temp files
2. `wtf` reads those files, gathers system context (OS, shell, project type), and sends it to an LLM
3. The explanation streams to your terminal in real time

## Setup

### 1. Build the binary

```bash
go build -o wtf .
```

### 2. Install the shell hook

```bash
bash setup.sh
```

This adds a hook to your `.zshrc` or `.bashrc` that captures failed commands automatically. Restart your terminal or `source` your shell config to activate.

### 3. Configure your API key

**Option 1: Config file (recommended — persists across restarts)**

```bash
mkdir -p ~/.config/wtf
echo 'your-api-key' > ~/.config/wtf/api_key
```

**Option 2: Environment variable (add to your shell config to persist)**

```bash
export OPENAI_API_KEY=your-key-here
# or
export ANTHROPIC_API_KEY=your-key-here
```

## Usage

```bash
# Run a command that fails
$ gcc foo.c
foo.c:3:10: fatal error: 'bar.h' file not found

# Ask wtf happened
$ wtf
```

## Requirements

- Go 1.21+
- bash or zsh
- An OpenAI or Anthropic API key
