#!/bin/bash

# setup.sh — Installs the `wtf` shell hook into your shell configuration.
#
# The hook runs after every command and captures:
#   - The failed command text → ${TMPDIR:-/tmp}/wtf_last_command
#   - The stderr output → ${TMPDIR:-/tmp}/wtf_last_stderr
#
# Supports: bash (~/.bashrc) and zsh (~/.zshrc)
# Idempotent: won't add the hook twice if already installed.
#
# NOTE (v1 limitation): This approach re-runs the failed command to capture
# stderr. Side-effecting commands (rm, curl POST) will run twice.
# A v2 could use `script` or fd redirection to capture stderr on first run.

set -e

# Detect the current shell
CURRENT_SHELL=$(basename "$SHELL")

if [[ "$CURRENT_SHELL" != "bash" && "$CURRENT_SHELL" != "zsh" ]]; then
    echo "Error: Unsupported shell '$CURRENT_SHELL'."
    echo "wtf currently supports bash and zsh only."
    exit 1
fi

echo "Detected shell: $CURRENT_SHELL"

# Determine the shell config file
if [[ "$CURRENT_SHELL" == "zsh" ]]; then
    SHELL_RC="$HOME/.zshrc"
elif [[ "$CURRENT_SHELL" == "bash" ]]; then
    if [[ -f "$HOME/.bashrc" ]]; then
        SHELL_RC="$HOME/.bashrc"
    else
        SHELL_RC="$HOME/.bash_profile"
    fi
fi

echo "Config file: $SHELL_RC"

# Don't install twice
if grep -q "# >>> wtf shell hook >>>" "$SHELL_RC" 2>/dev/null; then
    echo "wtf shell hook is already installed in $SHELL_RC"
    exit 0
fi

# Backup the config file
if [[ -f "$SHELL_RC" ]]; then
    cp "$SHELL_RC" "${SHELL_RC}.wtf-backup"
    echo "Backup created: ${SHELL_RC}.wtf-backup"
fi

# Install the hook
if [[ "$CURRENT_SHELL" == "zsh" ]]; then
    cat >> "$SHELL_RC" << 'HOOK'

# >>> wtf shell hook >>>
# Captures failed commands and their stderr for the `wtf` CLI tool.
# NOTE: Re-runs the failed command to capture stderr (v1 limitation).
__wtf_capture() {
    local last_exit=$?
    local tmp_dir="${TMPDIR:-/tmp}"

    if [[ $last_exit -ne 0 ]]; then
        local last_cmd
        last_cmd=$(fc -ln -1 | sed 's/^[[:space:]]*//')
        echo "$last_cmd" > "${tmp_dir}/wtf_last_command"
        eval "$last_cmd" 2>&1 >/dev/null > "${tmp_dir}/wtf_last_stderr" 2>&1 || true
    fi
}
precmd_functions+=(__wtf_capture)
# <<< wtf shell hook <<<
HOOK

elif [[ "$CURRENT_SHELL" == "bash" ]]; then
    cat >> "$SHELL_RC" << 'HOOK'

# >>> wtf shell hook >>>
# Captures failed commands and their stderr for the `wtf` CLI tool.
# NOTE: Re-runs the failed command to capture stderr (v1 limitation).
__wtf_capture() {
    local last_exit=$?
    local tmp_dir="${TMPDIR:-/tmp}"

    if [[ $last_exit -ne 0 ]]; then
        local last_cmd
        last_cmd=$(history 1 | sed 's/^[[:space:]]*[0-9]*[[:space:]]*//')
        echo "$last_cmd" > "${tmp_dir}/wtf_last_command"
        eval "$last_cmd" 2>&1 >/dev/null > "${tmp_dir}/wtf_last_stderr" 2>&1 || true
    fi
}
PROMPT_COMMAND="${PROMPT_COMMAND:+$PROMPT_COMMAND;} __wtf_capture"
# <<< wtf shell hook <<<
HOOK

fi

echo ""
echo "wtf shell hook installed successfully!"
echo ""
echo "To activate, restart your terminal or run:"
echo "  source $SHELL_RC"
echo ""
echo "--- API Key Setup ---"
echo ""
echo "wtf needs an API key to work. Pick one of these options:"
echo ""
echo "  Option 1: Config file (recommended — persists across restarts):"
echo "    mkdir -p ~/.config/wtf"
echo "    echo 'your-api-key' > ~/.config/wtf/api_key"
echo ""
echo "  Option 2: Environment variable (add to $SHELL_RC to persist):"
echo "    export OPENAI_API_KEY=your-key-here"
echo "    export ANTHROPIC_API_KEY=your-key-here"
