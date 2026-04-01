#!/bin/bash

# setup.sh — Installs the `wtf` shell hook into your shell configuration.
#
# Uses a preexec/precmd pattern to capture stderr on the original run:
#   - preexec: runs BEFORE each command — redirects stderr to a temp file
#     via file descriptor duplication, while still displaying it to the terminal
#   - precmd: runs AFTER each command — if the command failed, saves the
#     command text and keeps the captured stderr; otherwise cleans up
#
# Key concepts you'll need:
#   - exec 3>&2        — duplicate fd 2 (stderr) to fd 3 (backup)
#   - exec 2> >(tee file >&3)  — redirect stderr through tee: writes to
#     both the file and the original stderr (fd 3), so the user still sees errors
#   - exec 2>&3 3>&-   — restore stderr from backup, close fd 3
#   - $? in precmd/PROMPT_COMMAND gives the exit code of the last command
#
# Supports: bash (~/.bashrc) and zsh (~/.zshrc)
# Idempotent: won't add the hook twice if already installed.

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
# Uses preexec/precmd to capture stderr on the original run — no re-execution.
# Temp files are namespaced by shell PID to avoid concurrent shell conflicts.

__wtf_tmp_dir="${TMPDIR:-/tmp}"
__wtf_last_cmd=""
__wtf_stderr_file="$__wtf_tmp_dir/wtf_last_stderr.$$"
__wtf_cmd_file="$__wtf_tmp_dir/wtf_last_command.$$"
# Symlinks point to the most recent session's files so `wtf` can find them.
__wtf_stderr_link="$__wtf_tmp_dir/wtf_last_stderr"
__wtf_cmd_link="$__wtf_tmp_dir/wtf_last_command"

# preexec runs BEFORE each command.
# $1 in zsh preexec gives the full command line (handles multi-line and pipes).
__wtf_preexec() {
    __wtf_last_cmd="$1"
    exec 3>&2
    exec 2> >(tee "$__wtf_stderr_file" >&3)
}

# precmd runs AFTER each command.
__wtf_postcmd() {
    local last_exit=$?
    exec 2>&3 3>&-

    if [[ $last_exit -ne 0 && -n "$__wtf_last_cmd" ]]; then
        echo "$__wtf_last_cmd" > "$__wtf_cmd_file"
        ln -sf "$__wtf_stderr_file" "$__wtf_stderr_link"
        ln -sf "$__wtf_cmd_file" "$__wtf_cmd_link"
    else
        rm -f "$__wtf_stderr_file"
    fi
}

# Clean up PID-namespaced files on shell exit.
__wtf_cleanup() {
    rm -f "$__wtf_stderr_file" "$__wtf_cmd_file"
    # Only remove symlinks if they point to our files.
    [[ "$(readlink "$__wtf_stderr_link" 2>/dev/null)" == "$__wtf_stderr_file" ]] && rm -f "$__wtf_stderr_link"
    [[ "$(readlink "$__wtf_cmd_link" 2>/dev/null)" == "$__wtf_cmd_file" ]] && rm -f "$__wtf_cmd_link"
}
trap __wtf_cleanup EXIT

preexec_functions+=(__wtf_preexec)
precmd_functions+=(__wtf_postcmd)
# <<< wtf shell hook <<<
HOOK

elif [[ "$CURRENT_SHELL" == "bash" ]]; then
    cat >> "$SHELL_RC" << 'HOOK'

# >>> wtf shell hook >>>
# Captures failed commands and their stderr for the `wtf` CLI tool.
# Uses DEBUG trap (preexec) + PROMPT_COMMAND (precmd) — no re-execution.
# Temp files are namespaced by shell PID to avoid concurrent shell conflicts.

__wtf_tmp_dir="${TMPDIR:-/tmp}"
__wtf_last_cmd=""
__wtf_active=0
__wtf_captured=0
__wtf_stderr_file="$__wtf_tmp_dir/wtf_last_stderr.$$"
__wtf_cmd_file="$__wtf_tmp_dir/wtf_last_command.$$"
__wtf_stderr_link="$__wtf_tmp_dir/wtf_last_stderr"
__wtf_cmd_link="$__wtf_tmp_dir/wtf_last_command"

# DEBUG trap runs BEFORE each command (bash's preexec equivalent).
# Guards:
#   - __wtf_active: prevents firing during PROMPT_COMMAND
#   - __wtf_captured: only captures the first simple command per prompt cycle,
#     so compound commands (if/for/pipes) don't overwrite __wtf_last_cmd
__wtf_preexec() {
    [[ $__wtf_active -eq 1 ]] && return
    [[ $__wtf_captured -eq 1 ]] && return
    __wtf_last_cmd="$BASH_COMMAND"
    __wtf_captured=1
    exec 3>&2
    exec 2> >(tee "$__wtf_stderr_file" >&3)
}

# PROMPT_COMMAND runs AFTER each command (bash's precmd equivalent).
__wtf_postcmd() {
    local last_exit=$?
    __wtf_active=1
    exec 2>&3 3>&- 2>/dev/null

    if [[ $last_exit -ne 0 && -n "$__wtf_last_cmd" ]]; then
        echo "$__wtf_last_cmd" > "$__wtf_cmd_file"
        ln -sf "$__wtf_stderr_file" "$__wtf_stderr_link"
        ln -sf "$__wtf_cmd_file" "$__wtf_cmd_link"
    else
        rm -f "$__wtf_stderr_file"
    fi
    __wtf_active=0
    __wtf_captured=0
}

# Clean up PID-namespaced files on shell exit.
__wtf_cleanup() {
    rm -f "$__wtf_stderr_file" "$__wtf_cmd_file"
    [[ "$(readlink "$__wtf_stderr_link" 2>/dev/null)" == "$__wtf_stderr_file" ]] && rm -f "$__wtf_stderr_link"
    [[ "$(readlink "$__wtf_cmd_link" 2>/dev/null)" == "$__wtf_cmd_file" ]] && rm -f "$__wtf_cmd_link"
}
trap __wtf_cleanup EXIT

trap '__wtf_preexec' DEBUG
PROMPT_COMMAND="${PROMPT_COMMAND:+$PROMPT_COMMAND;} __wtf_postcmd"
# <<< wtf shell hook <<<
HOOK

fi

echo ""
echo "wtf shell hook installed successfully!"
echo ""
echo "To activate, restart your terminal or run:"
echo "  source $SHELL_RC"

# --- Agent configuration ---
echo ""
echo "--- Coding Agent Setup ---"
echo ""
echo "Which coding agent should 'wtf fix' use?"
echo ""
echo "  1) claude  — Claude Code (default)"
echo "  2) codex   — OpenAI Codex CLI"
echo "  3) custom  — enter a custom command"
echo ""
read -rp "Choose [1/2/3] (default: 1): " agent_choice

agent="claude"
if [[ "$agent_choice" == "2" || "$agent_choice" == "codex" ]]; then
    agent="codex"
elif [[ "$agent_choice" == "3" || "$agent_choice" == "custom" ]]; then
    read -rp "Enter the command name: " agent
fi

mkdir -p "$HOME/.config/wtf"
echo "$agent" > "$HOME/.config/wtf/agent"
echo "Coding agent set to: $agent"
echo "You can change this later by editing ~/.config/wtf/agent"

# --- Fix mode configuration ---
echo ""
echo "--- Fix Mode Setup ---"
echo ""
echo "When you run 'wtf fix', how should the agent run?"
echo ""
echo "  1) oneshot     — agent fixes the issue and exits (default)"
echo "  2) interactive — agent opens a live session with the error context"
echo ""
read -rp "Choose [1/2] (default: 1): " fix_choice

fix_mode="oneshot"
if [[ "$fix_choice" == "2" || "$fix_choice" == "interactive" ]]; then
    fix_mode="interactive"
fi

echo "$fix_mode" > "$HOME/.config/wtf/fix_mode"
echo "Fix mode set to: $fix_mode"
echo "You can change this later by editing ~/.config/wtf/fix_mode"

# --- API key configuration ---
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
