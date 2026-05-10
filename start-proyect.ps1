# start-project.ps1

function Invoke-Step {
    param([string]$Label, [scriptblock]$Command)

    Write-Host "$Label..." -ForegroundColor Cyan
    $output = & $Command 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[FAILED] $Label" -ForegroundColor Red
        Write-Host $output -ForegroundColor DarkRed
        exit $LASTEXITCODE
    }
    Write-Host "[OK] $Label" -ForegroundColor Green
}

try {
    Invoke-Step "Generating swagger" { swag init -g cmd/main.go --output docs/ }
    Invoke-Step "Building docker image" { docker compose build }
    Invoke-Step "Starting containers" { docker compose up -d }

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
