@echo off
REM Fix: TDM-GCC 10.x + Go 1.21+ CGo on Windows requires -linkmode internal
REM to prevent the external linker (ld) from injecting .CRT/.tls sections
REM that conflict with the Go runtime on Windows.
go build -ldflags "-linkmode internal" -o havok-go.exe .
if %ERRORLEVEL% neq 0 (
    echo BUILD FAILED
    exit /b 1
)
echo BUILD OK