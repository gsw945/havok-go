@echo off
REM convert.cmd - regenerate havok/generated/bindings_gen.go from HavokPhysics.d.ts.
REM               The calling convention is read from the shipped HavokPhysics.wasm.
REM
REM Usage:
REM   convert.cmd                                        auto-detect the d.ts
REM   convert.cmd --input <path\to\HavokPhysics.d.ts>     specify the d.ts explicitly
REM   convert.cmd --wasm  <path\to\HavokPhysics.wasm>     specify the wasm explicitly
REM
REM Auto-detected locations (first match wins):
REM   ..\BabylonJS-havok\packages\havok\HavokPhysics.d.ts
REM   ..\refs\BabylonJS-havok\packages\havok\HavokPhysics.d.ts
REM   ..\..\refs\BabylonJS-havok\packages\havok\HavokPhysics.d.ts
REM
REM NOTE: the generator uses tree-sitter through CGo. If your C compiler is not on
REM PATH, set CC and PATH first, e.g.
REM   set "CC=D:\msys64\mingw64\bin\gcc.exe"
REM   set "PATH=D:\msys64\mingw64\bin;%PATH%"

setlocal

set "CAND1=%~dp0..\BabylonJS-havok\packages\havok\HavokPhysics.d.ts"
set "CAND2=%~dp0..\refs\BabylonJS-havok\packages\havok\HavokPhysics.d.ts"
set "CAND3=%~dp0..\..\refs\BabylonJS-havok\packages\havok\HavokPhysics.d.ts"

set "DEFAULT_DTS="
for %%C in ("%CAND1%" "%CAND2%" "%CAND3%") do (
    if not defined DEFAULT_DTS if exist "%%~C" set "DEFAULT_DTS=%%~C"
)

if not defined DEFAULT_DTS (
    echo ERROR: HavokPhysics.d.ts not found. Tried:
    echo   %CAND1%
    echo   %CAND2%
    echo   %CAND3%
    echo Pass --input ^<path^> to specify it.
    exit /b 1
)

echo Using d.ts: %DEFAULT_DTS%

REM If no arguments are provided, use the detected path.
if "%~1"=="" (
    set "EXTRA_ARGS=--input "%DEFAULT_DTS%""
) else (
    REM If --input is not present, prepend the detected d.ts path.
    echo %* | findstr /i "\-\-input" >nul
    if errorlevel 1 (
        set "EXTRA_ARGS=--input "%DEFAULT_DTS%" %*"
    ) else (
        set "EXTRA_ARGS=%*"
    )
)

go run -tags gen_only . convert --skip-types %EXTRA_ARGS%
if %ERRORLEVEL% neq 0 (
    echo CONVERT FAILED
    exit /b 1
)
echo CONVERT OK
endlocal
exit /b 0
