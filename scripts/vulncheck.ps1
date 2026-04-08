$ErrorActionPreference = "Stop"

Push-Location "$PSScriptRoot\..\api"

try {
    $goExe = (Get-Command go -ErrorAction Stop).Source
    $goVersion = & $goExe version 2>&1
    Write-Host "Go: $goVersion" -ForegroundColor DarkGray

    $gopath = (& $goExe env GOPATH).Trim()
    $vulncheck = Join-Path $gopath "bin\govulncheck.exe"

    if (-not (Test-Path $vulncheck)) {
        Write-Host "Installing govulncheck..." -ForegroundColor Yellow
        & $goExe install golang.org/x/vuln/cmd/govulncheck@latest
        if ($LASTEXITCODE -ne 0) { throw "Failed to install govulncheck" }
    }

    $vulnVersion = & $vulncheck -version 2>&1
    Write-Host "Vulncheck: $vulnVersion" -ForegroundColor DarkGray
    Write-Host ""

    # Build fresh binary for scanning
    $bin = Join-Path $env:TEMP "runfive-vulncheck.exe"
    Write-Host "Building binary..." -ForegroundColor Cyan
    & $goExe build -o $bin .
    if ($LASTEXITCODE -ne 0) { throw "go build failed" }

    # Scan compiled binary — bypasses source-parsing Go version issues
    Write-Host "Running govulncheck on binary..." -ForegroundColor Cyan
    & $vulncheck -mode=binary $bin
    $exitCode = $LASTEXITCODE

    Remove-Item $bin -ErrorAction SilentlyContinue

    if ($exitCode -ne 0) { throw "govulncheck found vulnerabilities" }
    Write-Host "govulncheck: OK" -ForegroundColor Green
}
catch {
    Write-Host "`nFailed: $_" -ForegroundColor Red
    exit 1
}
finally {
    Pop-Location
}
