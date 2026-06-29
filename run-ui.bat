@echo off
chcp 65001 >nul
cd /d "%~dp0ui"
set PORT=8090

echo === UI (Vite) :5173  ->  backend :%PORT% ===
echo (Backend'i once run-backend.bat ile baslat.)
echo Node gerekmiyorsa: tarayicidan dogrudan http://localhost:%PORT% ac.
echo.

where npm >nul 2>&1
if %errorlevel%==0 (
  if not exist node_modules npm install
  set BACKEND_URL=http://localhost:%PORT%
  start http://localhost:5173
  npm run dev
) else (
  echo npm Windows'ta yok, WSL deneniyor...
  wsl bash -lc "cd /mnt/c/Users/yavuz/OneDrive/Desktop/disci/brain/ui && ([ -d node_modules ] || npm install) && BACKEND_URL=http://localhost:%PORT% npm run dev"
)
pause
