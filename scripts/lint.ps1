$ErrorActionPreference = "Stop"

Push-Location "$PSScriptRoot\..\api"

try {
    $goExe = (Get-Command go -ErrorAction Stop).Source
    $goVersion = & $goExe version 2>&1
    Write-Host "Go: $goVersion" -ForegroundColor DarkGray

    $linter = Join-Path (& $goExe env GOPATH) "bin\golangci-lint.exe"
    if (-not (Test-Path $linter)) {
        Write-Host "golangci-lint not found, installing..." -ForegroundColor Yellow
        & $goExe install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
        if ($LASTEXITCODE -ne 0) { throw "Failed to install golangci-lint" }
    }

    $linterVersion = & $linter version 2>&1
    Write-Host "Linter: $linterVersion" -ForegroundColor DarkGray
    Write-Host ""

    Write-Host "Running go vet..." -ForegroundColor Cyan
    & $goExe vet ./...
    if ($LASTEXITCODE -ne 0) { throw "go vet failed" }
    Write-Host "go vet: OK" -ForegroundColor Green
    Write-Host ""

    Write-Host "Running golangci-lint..." -ForegroundColor Cyan
    & $linter run ./...
    if ($LASTEXITCODE -ne 0) { throw "golangci-lint found issues" }
    Write-Host "golangci-lint: OK" -ForegroundColor Green
}
catch {
    Write-Host "`nFailed: $_" -ForegroundColor Red
    exit 1
}
finally {
    Pop-Location
}
