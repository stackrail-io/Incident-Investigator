#Requires -Version 5.1
<#
.SYNOPSIS
  Install incident-investigator on Windows.

.DESCRIPTION
  Downloads the latest release (or a specific version) from GitHub and installs
  the binary to %LOCALAPPDATA%\Programs\incident-investigator.

.EXAMPLE
  irm https://raw.githubusercontent.com/stackrail-io/Incident-Investigator/main/scripts/install.ps1 | iex

.EXAMPLE
  $env:INCIDENT_INVESTIGATOR_VERSION = "0.1.0"; irm .../install.ps1 | iex
#>

$ErrorActionPreference = "Stop"

$Repo = if ($env:INCIDENT_INVESTIGATOR_REPO) { $env:INCIDENT_INVESTIGATOR_REPO } else { "stackrail-io/Incident-Investigator" }
$Binary = "incident-investigator"
$Version = if ($env:INCIDENT_INVESTIGATOR_VERSION) { $env:INCIDENT_INVESTIGATOR_VERSION } else { "latest" }
$InstallRoot = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\incident-investigator" }

function Resolve-Arch {
    if ([System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture -eq [System.Runtime.InteropServices.Architecture]::Arm64) {
        return "arm64"
    }
    return "amd64"
}

function Resolve-Version {
    param([string]$Requested)
    if ($Requested -ne "latest") {
        return $Requested.TrimStart("v")
    }
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers @{ "User-Agent" = "incident-investigator-installer" }
    return $release.tag_name.TrimStart("v")
}

function Ensure-Directory {
    param([string]$Path)
    if (-not (Test-Path $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

$Version = Resolve-Version -Requested $Version
$Arch = Resolve-Arch
$Archive = "${Binary}_${Version}_windows_${Arch}.zip"
$Url = "https://github.com/$Repo/releases/download/v${Version}/$Archive"
$TempDir = Join-Path $env:TEMP ("ii-install-" + [guid]::NewGuid().ToString("n"))
$ZipPath = Join-Path $TempDir $Archive

try {
    Write-Host "Installing $Binary v$Version for windows/$Arch..."
    Write-Host "Downloading $Url"
    Ensure-Directory -Path $TempDir
    Invoke-WebRequest -Uri $Url -OutFile $ZipPath -UseBasicParsing
    Expand-Archive -Path $ZipPath -DestinationPath $TempDir -Force

    $ExeSrc = Join-Path $TempDir "$Binary.exe"
    if (-not (Test-Path $ExeSrc)) {
        throw "archive did not contain $Binary.exe"
    }

    Ensure-Directory -Path $InstallRoot
    $ExeDest = Join-Path $InstallRoot "$Binary.exe"
    Copy-Item -Path $ExeSrc -Destination $ExeDest -Force

    # Add install dir to user PATH if missing.
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$InstallRoot*") {
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallRoot", "User")
        $env:Path = "$env:Path;$InstallRoot"
        Write-Host "Added $InstallRoot to user PATH (open a new terminal if the command is not found)."
    }

    Write-Host ""
    Write-Host "Installed $Binary v$Version to $ExeDest"
    & $ExeDest version
    Write-Host ""
    Write-Host "Add to your MCP client config (Cursor, Claude Code, etc.):"
    Write-Host ""
    $configPath = $ExeDest -replace '\\', '/'
    @"
{
  "mcpServers": {
    "incident-investigator": {
      "command": "$configPath"
    }
  }
}
"@ | Write-Host
}
finally {
    if (Test-Path $TempDir) {
        Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue
    }
}
