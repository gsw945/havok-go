@echo off
REM example.cmd - run the Havok example simulation.
REM
REM Usage:
REM   example.cmd                          use embedded wasm (run convert.cmd first)
REM   example.cmd --wasm <path\to\*.wasm>  specify wasm path manually

go run . example %*
if %ERRORLEVEL% neq 0 (
    echo EXAMPLE FAILED
    exit /b 1
)
echo EXAMPLE OK
exit /b 0
