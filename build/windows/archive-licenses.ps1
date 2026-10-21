[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$LicenseDirectory,

    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$OutputPath,

    [Parameter(Mandatory = $false)]
    [string]$LogPath = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Write-BuildLog {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateSet("INFO", "DEBUG", "WARN", "ERROR")]
        [string]$Level,

        [Parameter(Mandatory = $true)]
        [string]$Message
    )

    $line = "{0} {1} {2}" -f (Get-Date).ToUniversalTime().ToString("o"), $Level, $Message
    Write-Host $line

    if ($LogPath) {
        $directory = Split-Path -Parent $LogPath
        if ($directory) {
            New-Item -ItemType Directory -Force -Path $directory | Out-Null
        }
        Add-Content -LiteralPath $LogPath -Value $line -Encoding UTF8
    }
}

try {
    $LicenseDirectory = [System.IO.Path]::GetFullPath($LicenseDirectory)
    $OutputPath = [System.IO.Path]::GetFullPath($OutputPath)

    $requiredNames = @(
        "PROJECT_LICENSE.txt",
        "THIRD_PARTY_NOTICES.txt",
        "go-dependency-licenses.txt",
        "js-dependency-licenses.txt"
    )

    $files = @(
        foreach ($name in $requiredNames) {
            $file = Join-Path $LicenseDirectory $name

            if (-not (Test-Path -LiteralPath $file -PathType Leaf)) {
                throw "Required license file is missing: $file"
            }

            if ((Get-Item -LiteralPath $file).Length -eq 0) {
                throw "Required license file is empty: $file"
            }

            $file
        }
    )

    $outputDirectory = Split-Path -Parent $OutputPath
    New-Item -ItemType Directory -Force -Path $outputDirectory | Out-Null

    if (Test-Path -LiteralPath $OutputPath) {
        Remove-Item -LiteralPath $OutputPath -Force
    }

    Write-BuildLog -Level INFO -Message "Creating license archive: $OutputPath"

    Compress-Archive `
        -LiteralPath $files `
        -DestinationPath $OutputPath `
        -CompressionLevel Optimal `
        -Force

    if (-not (Test-Path -LiteralPath $OutputPath -PathType Leaf)) {
        throw "License archive was not created: $OutputPath"
    }

    if ((Get-Item -LiteralPath $OutputPath).Length -eq 0) {
        throw "License archive is empty: $OutputPath"
    }

    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $OutputPath).Hash
    Write-BuildLog -Level INFO -Message "Created license archive SHA256=$hash"
}
catch {
    Write-BuildLog -Level ERROR -Message $_.Exception.Message
    throw
}
