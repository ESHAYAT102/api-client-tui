$ErrorActionPreference = "Stop"

$AppName = "api"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Resolve-Path (Join-Path $ScriptDir "..")
$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\api"
$Target = Join-Path $InstallDir "$AppName.exe"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "go is required but was not found in PATH"
}

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Push-Location $RootDir
try {
    go build -buildvcs=false -o $Target .
} finally {
    Pop-Location
}

$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
$PathParts = @()
if ($UserPath) {
    $PathParts = $UserPath -split ";"
}

if ($PathParts -notcontains $InstallDir) {
    $NewPath = if ($UserPath) { "$UserPath;$InstallDir" } else { $InstallDir }
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
    Write-Host "Added $InstallDir to user PATH. Open a new terminal to use '$AppName' globally."
}

Write-Host "Installed $AppName to $Target"
