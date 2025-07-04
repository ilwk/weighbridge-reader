@echo off
set EXE_PATH=%~dp0reader.exe
reg add "HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run" /v WeighbridgeWS /d "%EXE_PATH%" /f
echo 开机启动项已添加: %EXE_PATH%
pause