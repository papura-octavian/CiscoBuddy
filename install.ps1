# CiscoBuddy installer for Windows (PowerShell).
# Works in two modes:
#   1) Local: cd into the cloned repo and run .\install.ps1
#   2) Remote: iwr -useb <url-to-this-script> | iex
#      (clones the repo into a temp dir, builds, installs)

$ErrorActionPreference = "Stop"

$RepoUrl    = "https://github.com/papura-octavian/CiscoBuddy.git"
$BinName    = "ciscobuddy.exe"
$InstallDir = Join-Path $env:USERPROFILE "bin"

# If running locally (script file exists on disk), cd into its directory
if ($PSScriptRoot) {
    Set-Location -Path $PSScriptRoot
}

# If source files aren't here, clone the repo into a temp dir
if (-not (Test-Path "main.go") -or -not (Test-Path "go.mod")) {
    if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
        Write-Error "'git' nu este instalat. Get it: https://git-scm.com/download/win"
        exit 1
    }
    $tmp = Join-Path $env:TEMP "CiscoBuddy-$(Get-Random)"
    Write-Host ">> Clonez $RepoUrl ..."
    & git clone --depth 1 $RepoUrl $tmp
    if ($LASTEXITCODE -ne 0) {
        Write-Error "git clone a esuat."
        exit 1
    }
    Set-Location $tmp
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "'go' nu este instalat sau nu e in PATH. Instaleaza Go de la https://go.dev/dl/"
    exit 1
}

Write-Host ">> Build $BinName ..."
& go build -o $BinName .
if ($LASTEXITCODE -ne 0) {
    Write-Error "go build a esuat."
    exit 1
}

if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir | Out-Null
}

Copy-Item -Path $BinName -Destination (Join-Path $InstallDir $BinName) -Force
Write-Host ">> Instalat in: $InstallDir\$BinName"

# Add InstallDir to user PATH if not already there
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($null -eq $userPath) { $userPath = "" }

$pathParts = $userPath.Split(';') | Where-Object { $_ -ne "" }
if ($pathParts -notcontains $InstallDir) {
    $newPath = if ($userPath -eq "") { $InstallDir } else { "$userPath;$InstallDir" }
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Write-Host ">> $InstallDir a fost adaugat in PATH (user-level)."
    Write-Host ">> ATENTIE: deschide un terminal NOU ca modificarea sa fie vizibila."
} else {
    Write-Host ">> $InstallDir este deja in PATH."
}

Write-Host ""
Write-Host "Gata. Foloseste: ciscobuddy -ip ..."
