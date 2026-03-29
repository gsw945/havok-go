@echo off
REM convert.cmd - regenerate bindings under havok/generated from HavokPhysics.d.ts.
REM
REM Usage:
REM   convert.cmd                                   use default d.ts path (../BabylonJS-havok/...)
REM   convert.cmd --input <path\to\HavokPhysics.d.ts>   specify d.ts path manually
REM   convert.cmd --wasm  <path\to\HavokPhysics.wasm>   specify wasm path manually
REM
REM Default d.ts path (relative to this script's parent directory):
REM   ..\BabylonJS-havok\packages\havok\HavokPhysics.d.ts

setlocal

REM Default input path (can be overridden by --input)
set "DEFAULT_DTS=%~dp0..\BabylonJS-havok\packages\havok\HavokPhysics.d.ts"

REM If no arguments are provided, use the default path.
if "%~1"=="" (
    set "EXTRA_ARGS=--input %DEFAULT_DTS%"
) else (
    REM If --input is not present, prepend the default d.ts path.
    echo %* | findstr /i "\-\-input" >nul
    if errorlevel 1 (
        set "EXTRA_ARGS=--input %DEFAULT_DTS% %*"
    ) else (
        set "EXTRA_ARGS=%*"
    )
)

REM Reset errorlevel in case findstr left a non-zero code behind.
call :reset_errorlevel

go run -tags gen_only . convert --skip-types %EXTRA_ARGS%
if %ERRORLEVEL% neq 0 (
    echo CONVERT FAILED
    exit /b 1
)
echo CONVERT OK
endlocal
exit /b 0

:reset_errorlevel
exit /b 0
