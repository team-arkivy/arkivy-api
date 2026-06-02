# start-project.ps1

param(
    [string]$option = "docker"  # "localhost" or "docker"
)

function Write-Done { param([string]$msg) Write-Host $msg -ForegroundColor Yellow }
function Write-Ok   { param([string]$msg) Write-Host $msg -ForegroundColor Green }
function Write-Fail { param([string]$msg) Write-Host $msg -ForegroundColor Red }

function Invoke-Step {
    param([string]$Label, [scriptblock]$Command)
    Write-Host "$Label..." -ForegroundColor Cyan
    $output = & $Command 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Fail "[FAILED] $Label"
        Write-Host $output -ForegroundColor DarkRed
        exit $LASTEXITCODE
    }
    Write-Ok "[OK] $Label"
}

function Check-Docker {
    docker info 2>&1 | Out-Null
    if ($LASTEXITCODE -ne 0) {
        Write-Fail "> Docker is not running. Please start Docker and try again."
        exit 1
    }
    Write-Ok "> Docker is running [DONE]"
}

function Check-Port {
    param([string]$portType, [int]$portNumber)
    $inUse = Get-NetTCPConnection -LocalPort $portNumber -State Listen -ErrorAction SilentlyContinue
    if ($inUse) {
        Write-Fail "> Port $portType $portNumber is already in use [FAILED]"
        $inUse | Format-Table LocalAddress, LocalPort, State -AutoSize
        exit 1
    }
    Write-Ok "> Port $portType $portNumber is available [DONE]"
}

function Check-Go {
    $goVersion = go version 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Fail "> Go is not installed. Please install Go 1.21 or higher."
        exit 1
    }
    if ($goVersion -match 'go(\d+)\.(\d+)') {
        $major = [int]$Matches[1]
        $minor = [int]$Matches[2]
        if ($major -lt 1 -or ($major -eq 1 -and $minor -lt 21)) {
            Write-Fail "> Go version is less than 1.21. Please update Go."
            exit 1
        }
        Write-Ok "> Go $major.$minor detected [DONE]"
    }
}

function Import-EnvFile {
    param([string]$envFile)
    if (-not (Test-Path $envFile)) { return }
    Get-Content $envFile | ForEach-Object {
        if ($_ -match '^\s*([^#\s][^=]*)=(.*)$') {
            $key   = $Matches[1].Trim()
            $value = $Matches[2].Trim()
            [System.Environment]::SetEnvironmentVariable($key, $value, 'Process')
        }
    }
    Write-Ok "> Loaded env: $envFile [DONE]"
}

# ─── Main ────────────────────────────────────────────────────────────────────

Check-Docker
Check-Port "http"  9090
Check-Port "mongo" 27017

if ($option -eq "localhost") {
    Check-Go

    Invoke-Step "Generating swagger" { swag init -g cmd/main.go --output docs/ }

    Write-Host "> Starting MongoDB container..." -ForegroundColor Cyan
    docker compose up mongo -d
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    Write-Ok "> MongoDB started [DONE]"

    Import-EnvFile ".env"

    Write-Host ""
    Write-Host "Starting API locally..." -ForegroundColor Green
    Write-Host "   Swagger: http://localhost:9090/swagger/index.html" -ForegroundColor White
    Write-Host ""

    try {
        go run ./cmd/main.go
    } finally {
        Write-Host ""
        Write-Host "Stopping MongoDB container..." -ForegroundColor Yellow
        docker compose stop mongo
        Write-Host "Done." -ForegroundColor Green
    }

} else {
    try {
        Invoke-Step "Generating swagger" { swag init -g cmd/main.go --output docs/ }
        Invoke-Step "Building docker image" { docker compose build }
        Invoke-Step "Starting containers"  { docker compose up -d }

        Write-Host ""
        Write-Host "Project started!" -ForegroundColor Green
        Write-Host "   Swagger: http://localhost:9090/swagger/index.html" -ForegroundColor White
        Write-Host ""
        Write-Host "Streaming container logs (Ctrl+C to stop)..." -ForegroundColor DarkGray
        Write-Host ""

        docker compose logs -f

    } finally {
        Write-Host ""
        Write-Host "Stopping containers..." -ForegroundColor Yellow
        docker compose stop
        Write-Host "Done." -ForegroundColor Green
    }
}