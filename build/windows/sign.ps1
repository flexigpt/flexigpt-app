[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateNotNullOrEmpty()]
    [string]$FilePath,

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

function Get-SignToolPath {
    $programFilesX86 = [Environment]::GetEnvironmentVariable("ProgramFiles(x86)")
    if (-not $programFilesX86) {
        throw "ProgramFiles(x86) is not available"
    }

    $kitsRoot = Join-Path $programFilesX86 "Windows Kits\10\bin"
    if (-not (Test-Path -LiteralPath $kitsRoot -PathType Container)) {
        throw "Windows SDK directory not found: $kitsRoot"
    }

    $candidates = @(
        Get-ChildItem -LiteralPath $kitsRoot -Filter "signtool.exe" -Recurse -ErrorAction SilentlyContinue |
            Where-Object { $_.Directory.Name -eq "x64" } |
            Sort-Object FullName -Descending
    )

    if ($candidates.Count -eq 0) {
        throw "signtool.exe was not found below $kitsRoot"
    }

    return $candidates[0].FullName
}

$store = $null
$collection = $null
$preview = $null
$importedCertificate = $false
$thumbprint = ""

try {
    if (-not (Test-Path -LiteralPath $FilePath -PathType Leaf)) {
        throw "Signing target does not exist: $FilePath"
    }

    $FilePath = [System.IO.Path]::GetFullPath($FilePath)

    if (-not $env:SIGN_WINDOWS_CERT) {
        throw "SIGN_WINDOWS_CERT is not set"
    }

    if (-not $env:SIGN_WINDOWS_CERT_PASSWORD) {
        throw "SIGN_WINDOWS_CERT_PASSWORD is not set"
    }

    Write-BuildLog -Level INFO -Message "Preparing Windows signing identity for $FilePath"

    try {
        $certificateBytes = [System.Convert]::FromBase64String($env:SIGN_WINDOWS_CERT)
    }
    catch {
        throw "SIGN_WINDOWS_CERT is not valid base64 PFX data"
    }

    $flags = [System.Security.Cryptography.X509Certificates.X509KeyStorageFlags]

    $preview = [System.Security.Cryptography.X509Certificates.X509Certificate2]::new(
        $certificateBytes,
        $env:SIGN_WINDOWS_CERT_PASSWORD,
        $flags::EphemeralKeySet
    )

    $thumbprint = $preview.Thumbprint
    if (-not $thumbprint) {
        throw "The signing certificate has no thumbprint"
    }

    $store = [System.Security.Cryptography.X509Certificates.X509Store]::new(
        "My",
        "CurrentUser"
    )
    $store.Open([System.Security.Cryptography.X509Certificates.OpenFlags]::ReadWrite)

    $certificate = @(
        $store.Certificates |
            Where-Object { $_.Thumbprint -eq $thumbprint } |
            Select-Object -First 1
    )

    if ($certificate.Count -eq 1) {
        if (-not $certificate[0].HasPrivateKey) {
            throw "Existing certificate $thumbprint has no private key"
        }

        Write-BuildLog -Level INFO -Message "Using existing certificate $thumbprint from CurrentUser\My"
    }
    else {
        $collection = [System.Security.Cryptography.X509Certificates.X509Certificate2Collection]::new()
        $collection.Import(
            $certificateBytes,
            $env:SIGN_WINDOWS_CERT_PASSWORD,
            $flags::PersistKeySet
        )

        $privateKeyCertificates = @(
            $collection | Where-Object { $_.HasPrivateKey }
        )

        if ($privateKeyCertificates.Count -ne 1) {
            throw "The PFX must contain exactly one certificate with a private key"
        }

        $certificate = $privateKeyCertificates
        $store.Add($certificate[0])
        $importedCertificate = $true

        Write-BuildLog -Level INFO -Message "Imported temporary certificate $thumbprint into CurrentUser\My"
    }

    $timestampURL = $env:WINDOWS_TIMESTAMP_URL
    if (-not $timestampURL) {
        $timestampURL = "https://timestamp.digicert.com"
    }

    $timestampURI = [System.Uri]$timestampURL
    if ($timestampURI.Scheme -ne "https") {
        throw "WINDOWS_TIMESTAMP_URL must use HTTPS"
    }

    $signTool = Get-SignToolPath
    Write-BuildLog -Level INFO -Message "Signing with $signTool"

    & $signTool sign `
        /fd SHA256 `
        /tr $timestampURL `
        /td SHA256 `
        /s My `
        /sha1 $thumbprint `
        $FilePath

    if ($LASTEXITCODE -ne 0) {
        throw "signtool sign failed with exit code $LASTEXITCODE"
    }

    Write-BuildLog -Level INFO -Message "Verifying Authenticode signature"

    & $signTool verify /pa /all $FilePath

    if ($LASTEXITCODE -ne 0) {
        throw "signtool verify failed with exit code $LASTEXITCODE"
    }

    Write-BuildLog -Level INFO -Message "Successfully signed and verified $FilePath"
}
catch {
    Write-BuildLog -Level ERROR -Message $_.Exception.Message
    throw
}
finally {
    if ($store) {
        $store.Close()
        $store.Dispose()
    }

    if ($importedCertificate -and $thumbprint) {
        try {
            $certificatePath = "Cert:\CurrentUser\My\$thumbprint"

            if (Test-Path -LiteralPath $certificatePath) {
                Remove-Item -LiteralPath $certificatePath -DeleteKey -Force
                Write-BuildLog -Level INFO -Message "Removed temporary certificate $thumbprint"
            }
        }
        catch {
            Write-BuildLog -Level WARN -Message "Could not remove temporary certificate $thumbprint: $($_.Exception.Message)"
        }
    }

    if ($preview) {
        $preview.Dispose()
    }

    if ($collection) {
        foreach ($item in $collection) {
            $item.Dispose()
        }
    }
}
