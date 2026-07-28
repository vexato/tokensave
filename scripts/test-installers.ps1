$ErrorActionPreference = "Stop"
$installer = Join-Path $PSScriptRoot "install.ps1"
$content = Get-Content -Raw -LiteralPath $installer
foreach ($required in @(
    'TOKENSAVE_SKILL_DIR',
    '.agents\skills\tokensave',
    '.codex\skills\tokensave',
    'Get-FileHash -LiteralPath $archive -Algorithm SHA256',
    'Copy-Item -Destination $SkillDir -Recurse -Force'
)) {
    if (-not $content.Contains($required)) { throw "Missing installer behavior: $required" }
}
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("tokensave-installer-test-" + [guid]::NewGuid())
try {
    New-Item -ItemType Directory -Path $tempDir | Out-Null
    $asset = Join-Path $tempDir "tokensave_windows_amd64.zip"
    Set-Content -LiteralPath $asset -NoNewline -Value "fixture"
    $expected = (Get-FileHash -LiteralPath $asset -Algorithm SHA256).Hash
    $manifest = "$expected  tokensave_windows_amd64.zip"
    $match = [regex]::Match($manifest, '^\s*([a-fA-F0-9]{64})\s+\*?tokensave_windows_amd64\.zip\s*$')
    if (-not $match.Success -or $match.Groups[1].Value -ne $expected) { throw "Checksum parsing failed." }
} finally {
    Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}
Write-Output "PowerShell installer path and checksum checks passed."
