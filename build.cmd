@echo off
REM Windows build helper.
REM TDM-GCC 10.x + Go 1.21+ with CGo may require -linkmode internal
REM to avoid linker-injected .CRT/.tls sections conflicting with Go runtime.
go build -ldflags "-linkmode internal" -o havok-go.exe .
if %ERRORLEVEL% neq 0 (
    echo BUILD FAILED
    exit /b 1
)
echo BUILD OK