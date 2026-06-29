@echo off
chcp 65001 >nul
cd /d "%~dp0"
set PORT=8090

REM Eski/takili process'i oldur, sonra backend'i baslat.
for /f "tokens=5" %%p in ('netstat -ano ^| findstr ":%PORT% " ^| findstr LISTENING') do taskkill /F /PID %%p >nul 2>&1

echo === BACKEND :%PORT% ===
echo Gomulu arayuz: http://localhost:%PORT%   (Node gerekmez)
echo API: /v1/whatsapp /v1/sla /v1/arms /v1/budget/plan
echo.

where go >nul 2>&1
if %errorlevel%==0 (
  set BRAIN_ADDR=:%PORT%
  set BRAIN_SNAPSHOT=brain-data\snapshot.json
  go run ./cmd/brain serve
) else (
  echo Go Windows'ta yok, WSL deneniyor...
  wsl bash -lc "cd /mnt/c/Users/yavuz/OneDrive/Desktop/disci/brain && fuser -k %PORT%/tcp 2>/dev/null; BRAIN_ADDR=:%PORT% go run ./cmd/brain serve"
)
pause
