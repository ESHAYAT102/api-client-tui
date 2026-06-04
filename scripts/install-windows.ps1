$ErrorActionPreference = "Stop"

$AppName = "api"
$RepoUrl = "https://github.com/ESHAYAT102/api-client-tui.git"
$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\api"
$Target = Join-Path $InstallDir "$AppName.exe"
$CloneDir = Join-Path ([System.IO.Path]::GetTempPath()) "$AppName.install.$([System.Guid]::NewGuid().ToString('N'))"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "go is required but was not found in PATH"
}

if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Error "git is required but was not found in PATH"
}

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
try {
    git clone --depth 1 $RepoUrl $CloneDir
    Push-Location $CloneDir
    go build -buildvcs=false -o $Target .
} finally {
    if ((Get-Location).Path -eq $CloneDir) {
        Pop-Location
    }
    if (Test-Path $CloneDir) {
        Remove-Item -Recurse -Force $CloneDir
    }
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
