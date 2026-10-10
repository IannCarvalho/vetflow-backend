# Point git at the tracked .githooks directory so the shared pre-commit hook runs.
# Run once per clone, from the repo root:  powershell -ExecutionPolicy Bypass -File scripts\install-hooks.ps1
$ErrorActionPreference = 'Stop'
Set-Location (git rev-parse --show-toplevel)
git config core.hooksPath .githooks
Write-Output 'Hooks installed: core.hooksPath = .githooks'
Write-Output 'The pre-commit hook now rejects commits containing merge conflict markers.'
Write-Output 'Note: Git for Windows runs the bash pre-commit via its bundled sh, so no extra setup is needed.'
