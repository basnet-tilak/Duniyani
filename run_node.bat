@echo off
echo Building Duniyani Node...
go build -o duniyani-node.exe main.go
if not exist duniyani-node.exe (
    echo [!] Build failed or executable not found. Exiting...
    exit /b 1
)
echo Starting Duniyani Node Auto-Restarter...

:loop
.\duniyani-node.exe -node
echo.
echo [!] Duniyani node crashed or stopped. Respawning in 5 seconds...
ping 127.0.0.1 -n 6 > NUL
goto loop