@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion
cd /d "%~dp0"

REM ==========================================================================
REM  Disci Brain - tek tikla baslat (Windows)
REM  Backend :8090 + UI :5173.  8080'i KULLANMIYORUZ (orada baska servis var).
REM  Gereksinim: Go ve Node/npm kurulu olmali (Windows PATH'inde).
REM ==========================================================================

set PORT=8090
set UIPORT=5173

echo === Disci Brain baslatiliyor (backend :%PORT%, UI :%UIPORT%) ===

REM --- 1) Backend portunu bosalt (eski/takili process'i oldur) ---
for /f "tokens=5" %%p in ('netstat -ano ^| findstr ":%PORT% " ^| findstr LISTENING') do (
  echo  - :%PORT% portundaki eski process oldruluyor (PID %%p)
  taskkill /F /PID %%p >nul 2>&1
)

REM --- 2) Go var mi? (yoksa WSL'e dus) ---
where go >nul 2>&1
if %errorlevel%==0 (
  echo  - Go: Windows
  start "Disci Brain - Backend" cmd /k "set BRAIN_ADDR=:%PORT%&& set BRAIN_SNAPSHOT=brain-data\snapshot.json&& go run ./cmd/brain serve"
  start "Disci Brain - UI" cmd /k "cd ui&& set BACKEND_URL=http://localhost:%PORT%&& if not exist node_modules (npm install)&& npm run dev"
) else (
  echo  - Go Windows'ta yok, WSL kullaniliyor
  set WSLDIR=/mnt/c/Users/yavuz/OneDrive/Desktop/disci/brain
  wsl bash -lc "fuser -k %PORT%/tcp 2>/dev/null; true"
  start "Disci Brain - Backend" wsl bash -lc "cd !WSLDIR! && BRAIN_ADDR=:%PORT% go run ./cmd/brain serve"
  start "Disci Brain - UI" wsl bash -lc "cd !WSLDIR!/ui && ([ -d node_modules ] || npm install) && BACKEND_URL=http://localhost:%PORT% npm run dev"
)

echo.
echo  Ilk acilis derleme + npm install yuzunden ~1 dk surebilir.
echo  Tarayici acilinca sayfa bosss/down gorunuyorsa birkac saniye sonra YENILE (Ctrl+Shift+R).
echo.
timeout /t 18 >nul
start http://localhost:%UIPORT%

echo  Backend: http://localhost:%PORT%    UI: http://localhost:%UIPORT%
echo  Durdurmak icin acilan iki pencereyi kapatin ya da stop.bat calistirin.
endlocal
