[CmdletBinding()]
param(
    [string]$Version = "latest",
    [string]$InstallDir = $(if ($env:TOKENSAVE_INSTALL_DIR) { $env:TOKENSAVE_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\TokenSave" }),
    [switch]$WhatIf
)

$ErrorActionPreference = "Stop"
$Repository = "vexato/tokensave"
$Platform = "windows_amd64"

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
    Write-Host "Would download $($asset.browser_download_url) to $InstallDir\tokensave.exe"
    exit 0
}

$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("tokensave-" + [guid]::NewGuid())
$archive = Join-Path $tempDir $assetName
try {
    New-Item -ItemType Directory -Force -Path $tempDir | Out-Null
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $archive
    Expand-Archive -Path $archive -DestinationPath $tempDir -Force
    $binary = Get-ChildItem -Path $tempDir -Filter "tokensave.exe" -Recurse | Select-Object -First 1
    if (-not $binary) { throw "Downloaded archive does not contain tokensave.exe." }
    Copy-Item -LiteralPath $binary.FullName -Destination (Join-Path $InstallDir "tokensave.exe") -Force
} finally {
    Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Host "TokenSave $($release.tag_name) installed in $InstallDir"
if ((($env:Path -split ';') -notcontains $InstallDir)) {
    Write-Host "Add this directory to PATH, then open a new terminal: $InstallDir"
}
