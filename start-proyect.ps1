# start-project.ps1

try {
    Write-Host "Generating swagger..." -ForegroundColor Cyan
    swag init -g cmd/main.go --output docs/

    Write-Host "Building docker image..." -ForegroundColor Cyan
    docker compose build

    Write-Host "Starting containers..." -ForegroundColor Cyan
    docker compose up -d

    Write-Host "Proyect started!" -ForegroundColor Green
    Write-Host "   Swagger: http://localhost:9090/swagger/index.html" -ForegroundColor White
    Write-Host ""
    Write-Host "Press Ctrl+C to stop all..." -ForegroundColor DarkGray

    # Mantain the script live until Ctrl+C
    while ($true) { Start-Sleep -Seconds 1 }

} finally {
    Write-Host ""
    Write-Host "Stopping containers..." -ForegroundColor Yellow
    docker compose stop
    Write-Host "Program finish." -ForegroundColor Green
}