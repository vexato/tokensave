[CmdletBinding()]
param(
    [string]$Version = "latest",
    [string]$InstallDir = $(if ($env:TOKENSAVE_INSTALL_DIR) { $env:TOKENSAVE_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\TokenSave" }),
    [Alias("SkipCodexSkill")]
    [switch]$SkipSkill,
    [switch]$WhatIf
)

$ErrorActionPreference = "Stop"
$Repository = "vexato/tokensave"
$Platform = "windows_amd64"
$SkillDir = if ($env:TOKENSAVE_SKILL_DIR) { $env:TOKENSAVE_SKILL_DIR } else { Join-Path $HOME ".agents\skills\tokensave" }
$LegacySkillDir = Join-Path $HOME ".codex\skills\tokensave"

function Write-Step([string]$Label, [string]$Message) {
    Write-Host ("  {0,-10}" -f $Label) -NoNewline -ForegroundColor DarkCyan
    Write-Host $Message -ForegroundColor Gray
}

function Add-ToUserPath([string]$Directory) {
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $entries = @($userPath -split ';' | Where-Object { $_ })
    $containsDirectory = $entries | Where-Object { $_.TrimEnd('\') -ieq $Directory.TrimEnd('\') }
    if (-not $containsDirectory) {
        $newPath = if ([string]::IsNullOrWhiteSpace($userPath)) { $Directory } else { "$userPath;$Directory" }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    }

    $sessionEntries = @($env:Path -split ';' | Where-Object { $_ })
    if (-not ($sessionEntries | Where-Object { $_.TrimEnd('\') -ieq $Directory.TrimEnd('\') })) {
        $env:Path = if ($env:Path) { "$env:Path;$Directory" } else { $Directory }
    }
    return -not $containsDirectory
}

function Publish-EnvironmentChange {
    if (-not ("TokenSave.EnvironmentNotifier" -as [type])) {
        Add-Type @'
using System;
using System.Runtime.InteropServices;
namespace TokenSave {
    public static class EnvironmentNotifier {
        [DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
        public static extern IntPtr SendMessageTimeout(
            IntPtr hWnd, uint msg, IntPtr wParam, string lParam,
            uint flags, uint timeout, out IntPtr result);
    }
}
'@
    }
    $result = [IntPtr]::Zero
    [void][TokenSave.EnvironmentNotifier]::SendMessageTimeout(
        [IntPtr]0xffff, 0x001A, [IntPtr]::Zero, "Environment", 0x0002, 5000, [ref]$result
    )
}

Write-Host ""
Write-Host "  +-----------------------------+" -ForegroundColor Cyan
Write-Host "  |     TokenSave installer     |" -ForegroundColor Cyan
Write-Host "  +-----------------------------+" -ForegroundColor Cyan

if ($Version -eq "latest") {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repository/releases/latest" -Headers @{ "User-Agent" = "tokensave-installer" }
} else {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repository/releases/tags/$Version" -Headers @{ "User-Agent" = "tokensave-installer" }
}

$assetName = "tokensave_${Platform}.zip"
$asset = @($release.assets | Where-Object { $_.name -eq $assetName }) | Select-Object -First 1
if (-not $asset) {
    throw "Release $($release.tag_name) does not contain $assetName."
}
$checksumsAsset = @($release.assets | Where-Object { $_.name -eq "checksums.txt" }) | Select-Object -First 1
if (-not $checksumsAsset) {
    throw "Release $($release.tag_name) does not contain checksums.txt."
}

if ($WhatIf) {
    Write-Step "Would fetch" "$($release.tag_name) ($Platform)"
    Write-Step "Would verify" "SHA-256 for $assetName"
    Write-Step "Would install" (Join-Path $InstallDir "tokensave.exe")
    if (-not $SkipSkill) { Write-Step "Would add" "Codex Skill: $SkillDir" }
    exit 0
}

$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("tokensave-" + [guid]::NewGuid())
$archive = Join-Path $tempDir $assetName
$checksumsPath = Join-Path $tempDir "checksums.txt"
try {
    New-Item -ItemType Directory -Force -Path $tempDir | Out-Null
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Write-Step "Download" "$($release.tag_name) ($Platform)"
    Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $archive
    Invoke-WebRequest -Uri $checksumsAsset.browser_download_url -OutFile $checksumsPath
    $escapedAssetName = [regex]::Escape($assetName)
    $checksumLine = @(Get-Content -LiteralPath $checksumsPath | Where-Object { $_ -match "^\s*([a-fA-F0-9]{64})\s+\*?$escapedAssetName\s*$" })
    if ($checksumLine.Count -ne 1) { throw "checksums.txt does not contain exactly one SHA-256 entry for $assetName." }
    $checksumMatch = [regex]::Match($checksumLine[0], "^\s*([a-fA-F0-9]{64})\s+\*?$escapedAssetName\s*$")
    $expectedHash = $checksumMatch.Groups[1].Value.ToLowerInvariant()
    $actualHash = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualHash -ne $expectedHash) { throw "SHA-256 verification failed for $assetName." }
    Write-Step "Verified" "SHA-256 for $assetName"
    Write-Step "Extract" $assetName
    Expand-Archive -Path $archive -DestinationPath $tempDir -Force
    $binary = Get-ChildItem -Path $tempDir -Filter "tokensave.exe" -Recurse | Select-Object -First 1
    if (-not $binary) { throw "Downloaded archive does not contain tokensave.exe." }
    Copy-Item -LiteralPath $binary.FullName -Destination (Join-Path $InstallDir "tokensave.exe") -Force

    if (-not $SkipSkill) {
        $skillSource = Join-Path $tempDir "skills\tokensave"
        if (-not (Test-Path -LiteralPath $skillSource -PathType Container)) { throw "Downloaded archive does not contain skills/tokensave/." }
        if ((Test-Path -LiteralPath $LegacySkillDir -PathType Container) -and ($LegacySkillDir -ne $SkillDir)) {
            Write-Warning "Legacy Codex Skill found at $LegacySkillDir. It was not removed; migrate or remove it manually after verifying $SkillDir."
        }
        New-Item -ItemType Directory -Force -Path $SkillDir | Out-Null
        Get-ChildItem -Force -LiteralPath $skillSource | Copy-Item -Destination $SkillDir -Recurse -Force
        Write-Step "Codex Skill" $SkillDir
    }
} finally {
    Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}

$pathWasAdded = Add-ToUserPath $InstallDir
if ($pathWasAdded) { Publish-EnvironmentChange }
Write-Step "PATH" $(if ($pathWasAdded) { "added permanently for this user" } else { "already configured" })
Write-Host ""
Write-Host "  [OK] TokenSave $($release.tag_name) is ready." -ForegroundColor Green
Write-Host "    Try: " -NoNewline -ForegroundColor Gray
Write-Host "tokensave git status" -ForegroundColor White
if (-not $SkipSkill) {
    Write-Host "    Verify skills: " -NoNewline -ForegroundColor Gray
    Write-Host "codex /skills" -ForegroundColor White
}
