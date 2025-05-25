@echo off
reg delete "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v WeighbridgeWS /f
echo 已取消开机启动。
pause