[CmdletBinding()]
param(
    [string]$Version = "latest",
    [string]$InstallDir = $(if ($env:TOKENSAVE_INSTALL_DIR) { $env:TOKENSAVE_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\TokenSave" }),
    [switch]$SkipCodexSkill,
    [switch]$WhatIf
)

$ErrorActionPreference = "Stop"
$Repository = "vexato/tokensave"
$Platform = "windows_amd64"

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

if ($WhatIf) {
    Write-Step "Would fetch" "$($release.tag_name) ($Platform)"
    Write-Step "Would install" (Join-Path $InstallDir "tokensave.exe")
    if (-not $SkipCodexSkill) { Write-Step "Would add" "Codex Skill: tokensave" }
    exit 0
}

$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("tokensave-" + [guid]::NewGuid())
$archive = Join-Path $tempDir $assetName
try {
    New-Item -ItemType Directory -Force -Path $tempDir | Out-Null
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Write-Step "Download" "$($release.tag_name) ($Platform)"
    Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $archive
    Write-Step "Extract" $assetName
    Expand-Archive -Path $archive -DestinationPath $tempDir -Force
    $binary = Get-ChildItem -Path $tempDir -Filter "tokensave.exe" -Recurse | Select-Object -First 1
    if (-not $binary) { throw "Downloaded archive does not contain tokensave.exe." }
    Copy-Item -LiteralPath $binary.FullName -Destination (Join-Path $InstallDir "tokensave.exe") -Force

    if (-not $SkipCodexSkill) {
        $codexHome = if ($env:CODEX_HOME) { $env:CODEX_HOME } else { Join-Path $HOME ".codex" }
        $skill = Get-ChildItem -Path $tempDir -Filter "SKILL.md" -Recurse | Where-Object { $_.FullName.Replace('/', '\') -like '*\skills\tokensave\SKILL.md' } | Select-Object -First 1
        if ($skill) {
            $skillDir = Join-Path $codexHome "skills\tokensave"
            New-Item -ItemType Directory -Force -Path $skillDir | Out-Null
            Copy-Item -LiteralPath $skill.FullName -Destination (Join-Path $skillDir "SKILL.md") -Force
            Write-Step "Codex Skill" $skillDir
        } else {
            $skillDir = Join-Path $codexHome "skills\tokensave"
            New-Item -ItemType Directory -Force -Path $skillDir | Out-Null
            $skillUrl = "https://raw.githubusercontent.com/$Repository/main/skills/tokensave/SKILL.md"
            Invoke-WebRequest -Uri $skillUrl -OutFile (Join-Path $skillDir "SKILL.md")
            Write-Step "Codex Skill" "$skillDir (from main)"
        }
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
