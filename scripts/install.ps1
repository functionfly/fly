# FunctionFly CLI installer for Windows
# Usage: iwr -useb https://raw.githubusercontent.com/functionfly/fly/main/scripts/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "functionfly/fly"
$Binary = "fly"
$InstallDir = if ($env:FLY_INSTALL_DIR) { $env:FLY_INSTALL_DIR } else { "$env:LOCALAPPDATA\fly\bin" }

function Write-Info { param([string]$Msg) Write-Host "[fly] $Msg" -ForegroundColor Blue }
function Write-Ok   { param([string]$Msg) Write-Host "[fly] $Msg" -ForegroundColor Green }
function Write-Die  { param([string]$Msg) Write-Host "[fly] error: $Msg" -ForegroundColor Red; exit 1 }

function Get-LatestVersion {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers @{ "Accept" = "application/vnd.github+json" }
    return $release.tag_name
}

function Main {
    $arch = if ([Environment]::Is64BitOperatingSystem) {
        if ((Get-CimInstance Win32_OperatingSystem).SystemArchitecture -match "ARM") { "arm64" } else { "amd64" }
    } else {
        Write-Die "32-bit Windows is not supported"
    }

    $version = if ($env:FLY_VERSION) { $env:FLY_VERSION } else { Get-LatestVersion }
    $versionNum = $version -replace "^v", ""

    Write-Info "Installing fly $version for windows/$arch..."

    $archive = "${Binary}_${versionNum}_windows_${arch}.zip"
    $url = "https://github.com/$Repo/releases/download/$version/$archive"

    $tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
    New-Item -ItemType Directory -Path $tmpDir | Out-Null

    try {
        Write-Info "Downloading $url"
        $archivePath = Join-Path $tmpDir $archive
        Invoke-WebRequest -Uri $url -OutFile $archivePath -UseBasicParsing

        Write-Info "Extracting..."
        Expand-Archive -Path $archivePath -DestinationPath $tmpDir -Force

        $binaryPath = Get-ChildItem -Path $tmpDir -Name $Binary -Recurse -File | Select-Object -First 1
        if (-not $binaryPath) {
            Write-Die "Binary not found in archive"
        }
        $binaryPath = Join-Path $tmpDir $binaryPath

        if (-not (Test-Path $InstallDir)) {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }

        $destPath = Join-Path $InstallDir "${Binary}.exe"
        Copy-Item -Path $binaryPath -Destination $destPath -Force

        # Add to PATH if not already there
        $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($userPath -notlike "*$InstallDir*") {
            Write-Info "Adding $InstallDir to your PATH"
            [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir", "User")
            $env:Path = "$env:Path;$InstallDir"
        }

        Write-Ok "fly $version installed to $destPath"
        Write-Ok "Run 'fly --help' to get started."
    }
    finally {
        Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Main
