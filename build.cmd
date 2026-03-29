@echo off
REM Fix: TDM-GCC 10.x + Go 1.21+ require -fasynchronous-unwind-tables
REM for proper Windows SEH unwind table generation in CGo binaries.
set CGO_CFLAGS=-fasynchronous-unwind-tables
go build -ldflags "-linkmode internal" -o havok-go.exe main.go
if %ERRORLEVEL% neq 0 (
    echo BUILD FAILED
    exit /b 1
)
echo BUILD OK