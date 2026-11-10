$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
        throw "npm is required to build the frontend bundle."
    }
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        throw "Go is required to build the JobCard binary."
    }

    if (-not (Test-Path (Join-Path $root "node_modules"))) {
        npm install
        if ($LASTEXITCODE -ne 0) {
            throw "npm install failed with exit code $LASTEXITCODE."
        }
    }
    npm run build:desktop
    if ($LASTEXITCODE -ne 0) {
        throw "Frontend build failed with exit code $LASTEXITCODE."
    }
    $embedMarker = Join-Path $root "backend\cmd\jobcard\dist\.gitkeep"
    if (-not (Test-Path $embedMarker)) {
        New-Item -ItemType File -Path $embedMarker | Out-Null
    }

    Push-Location (Join-Path $root "backend")
    try {
        go build -ldflags "-H=windowsgui" -o jobcard.exe ./cmd/jobcard
        if ($LASTEXITCODE -ne 0) {
            throw "Go build failed with exit code $LASTEXITCODE."
        }
    }
    finally {
        Pop-Location
    }
}
finally {
    Pop-Location
}

Write-Host "Build complete: backend\jobcard.exe"
