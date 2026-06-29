@echo off
chcp 65001 >nul
REM Disci Brain'i durdur: 8090 portunu bosalt + brain/go/npm pencerelerini kapat.
set PORT=8090

for /f "tokens=5" %%p in ('netstat -ano ^| findstr ":%PORT% " ^| findstr LISTENING') do (
  echo :%PORT% kapatiliyor (PID %%p)
  taskkill /F /PID %%p >nul 2>&1
)
REM WSL tarafindaki backend de varsa:
wsl bash -lc "fuser -k %PORT%/tcp 2>/dev/null; pkill -9 -f cmd/brain 2>/dev/null; true" 2>nul

echo Durduruldu. (UI penceresini elle kapatabilirsin.)
