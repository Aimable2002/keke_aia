@echo off
setlocal

set VERSION=2.0.0
set ISCC="C:\Program Files (x86)\Inno Setup 6\ISCC.exe"

echo ================================================
echo Building Keke CLI Windows Installer v%VERSION%
echo ================================================
echo.

:: Check if Inno Setup is installed
if not exist %ISCC% (
    echo Error: Inno Setup not found!
    echo Please install from: https://jrsoftware.org/isdl.php
    pause
    exit /b 1
)

:: Check if keke.exe exists
if not exist "..\..\dist\keke.exe" (
    echo Error: keke.exe not found in dist folder!
    echo Please build the binary first.
    pause
    exit /b 1
)

:: Compile installer
echo Compiling installer...
%ISCC% keke.iss

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ========================================
    echo Success! Installer created:
    echo ========================================
    echo Location: ..\..\dist\keke-installer-windows-%VERSION%.exe
    echo.
    pause
) else (
    echo.
    echo Error: Build failed!
    pause
    exit /b 1
)