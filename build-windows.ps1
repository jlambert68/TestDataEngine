Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

param(
    [switch]$SkipUi,
    [switch]$SkipGo,
    [string]$OutputDir = "bin"
)

$repoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$uiDir = Join-Path $repoRoot "ui"
$binDir = Join-Path $repoRoot $OutputDir

function Require-Command {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Required command not found: $Name"
    }
}

Write-Host "Repo root: $repoRoot"

if (-not $SkipUi) {
    Require-Command -Name "npm"

    Write-Host "Building UI..."
    Push-Location $uiDir
    try {
        if (Test-Path (Join-Path $uiDir "package-lock.json")) {
            npm ci
        }
        else {
            npm install
        }
        npm run build
    }
    finally {
        Pop-Location
    }
}

if (-not $SkipGo) {
    Require-Command -Name "go"

    if (-not (Test-Path $binDir)) {
        New-Item -ItemType Directory -Path $binDir | Out-Null
    }

    Write-Host "Building Go binaries..."
    Push-Location $repoRoot
    try {
        go build -o (Join-Path $binDir "testdataengine.exe") .\cmd\testdataengine
        go build -o (Join-Path $binDir "testdataengine-web.exe") .\cmd\testdataengine-web
        go build -o (Join-Path $binDir "csv2sqlite.exe") .\cmd\csv2sqlite
    }
    finally {
        Pop-Location
    }
}

Write-Host ""
Write-Host "Build complete."
if (-not $SkipUi) {
    Write-Host "UI output: $uiDir\dist"
}
if (-not $SkipGo) {
    Write-Host "Binaries:"
    Write-Host "  $binDir\testdataengine.exe"
    Write-Host "  $binDir\testdataengine-web.exe"
    Write-Host "  $binDir\csv2sqlite.exe"
}
