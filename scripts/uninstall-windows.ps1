$ErrorActionPreference = "Stop"

$AppName = "api"
$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\api"
$Target = Join-Path $InstallDir "$AppName.exe"

if (Test-Path $Target) {
    Remove-Item $Target -Force
    Write-Host "Removed $Target"
} else {
    Write-Host "$AppName is not installed at $Target"
}

if (Test-Path $InstallDir) {
    $Remaining = Get-ChildItem $InstallDir -Force -ErrorAction SilentlyContinue
    if (-not $Remaining) {
        Remove-Item $InstallDir -Force
    }
}

$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath) {
    $PathParts = $UserPath -split ";" | Where-Object { $_ -and ($_ -ne $InstallDir) }
    $NewPath = $PathParts -join ";"
    if ($NewPath -ne $UserPath) {
        [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
        Write-Host "Removed $InstallDir from user PATH. Open a new terminal for PATH changes."
    }
}
