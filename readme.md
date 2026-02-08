# Keke CLI - AI Developer in Your Terminal

## Installation

### Windows (Recommended - One-Click Installer)

Download and run the installer:
https://github.com/Aimable2002/keke_aia/releases/latest/download/keke-installer-windows-2.0.0.exe

**Or use PowerShell:**
```powershell
irm https://raw.githubusercontent.com/Aimable2002/keke_aia/main/install.ps1 | iex
```

### macOS / Linux
```bash
curl -fsSL https://raw.githubusercontent.com/Aimable2002/keke_aia/main/installe.sh | bash
```

## Quick Start

After installation, open a **new terminal** and run:
```bash
# Initialize Keke in your project
keke init

# Login to your account
keke login

# Check your credits
keke credits

# Ask Keke anything
keke ask "how do I create a REST API in Go?"

# Start a research task
keke research "machine learning optimization techniques"
```

## Usage
```
keke [command]

Commands:
  init      Initialize Keke in your project
  login     Authenticate with Keke
  credits   Check your credit balance
  ask       Ask Keke a question
  research  Start a deep research task
  upgrade   Upgrade Keke to latest version
  version   Show version information
  help      Show help
```

## Troubleshooting

**Command not found after installation?**
- Close and reopen your terminal
- Or run: `$env:Path = [System.Environment]::GetEnvironmentVariable('Path','User')`

**Need help?**
- Documentation: https://github.com/Aimable2002/keke_aia
- Issues: https://github.com/Aimable2002/keke_aia/issues