@echo off
setlocal
cd /d "%~dp0.."
where npm >nul 2>nul
if errorlevel 1 (
    echo npm is required to build the frontend bundle.
    exit /b 1
)
where go >nul 2>nul
if errorlevel 1 (
    echo Go is required to build the JobCard binary.
    exit /b 1
)
if not exist node_modules call npm install
if errorlevel 1 exit /b 1
call npm run build:desktop
if errorlevel 1 exit /b 1
if not exist .\backend\cmd\jobcard\dist\.gitkeep type nul > .\backend\cmd\jobcard\dist\.gitkeep
if errorlevel 1 exit /b 1
cd /d backend
call go build -ldflags "-H=windowsgui" -o jobcard.exe .\cmd\jobcard
if errorlevel 1 exit /b 1
echo Build complete: backend\jobcard.exe
