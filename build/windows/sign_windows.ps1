param(
  [ValidateSet("app", "installer", "all")]
  [string]$Target = "all"
)

$ErrorActionPreference = "Stop"

# assumes CI exported required env vars (bash script sources build/buildvars.env with export)

if (-not $env:SIGN_WINDOWS_CERT) { throw "SIGN_WINDOWS_CERT not set" }
if (-not $env:SIGN_WINDOWS_CERT_PASSWORD) { throw "SIGN_WINDOWS_CERT_PASSWORD not set" }
if (-not $env:COMMON_BUILD_NAME) { throw "COMMON_BUILD_NAME not set" }

$certDir = Join-Path ([System.IO.Path]::GetTempPath()) ("flexigpt-sign-" + [guid]::NewGuid())
$pfxPath = Join-Path $certDir "certificate.pfx"

# Try to locate signtool.exe in typical Windows Kits locations
$signtoolCandidates = @(
  "C:\Program Files (x86)\Windows Kits\10\bin\x64\signtool.exe",
  "C:\Program Files (x86)\Windows Kits\10\bin\x86\signtool.exe"
)

$kitsRoot = "C:\Program Files (x86)\Windows Kits\10\bin"
if (Test-Path $kitsRoot) {
  # Also scan versioned subfolders, e.g. 10.0.22621.0\x64\signtool.exe
  Get-ChildItem -Path $kitsRoot -Directory -ErrorAction SilentlyContinue |
    Sort-Object Name -Descending |
    ForEach-Object {
      foreach ($arch in @("x64","x86")) {
        $candidate = Join-Path $_.FullName "$arch\signtool.exe"
        $signtoolCandidates += $candidate
      }
    }
}

$signtool = $signtoolCandidates | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $signtool) {
  throw "signtool.exe not found in Windows Kits paths"
}

$appExe = "build/bin/$($env:COMMON_BUILD_NAME).exe"
$installerExe = $env:WIN_INSTALLER_PATH
if (-not $installerExe) {
  $installerExe = "build/bin/$($env:COMMON_BUILD_NAME)-amd64-installer.exe"
}

$files = @()
if ($Target -eq "app" -or $Target -eq "all") {
  $files += $appExe
}
if ($Target -eq "installer" -or $Target -eq "all") {
  $files += $installerExe
}
foreach ($file in $files) {
  if (-not (Test-Path -LiteralPath $file)) {
    throw "Signing target not found: $file"
  }
}

try {
  New-Item -ItemType Directory -Force -Path $certDir | Out-Null
  [System.IO.File]::WriteAllBytes(
    $pfxPath,
    [System.Convert]::FromBase64String($env:SIGN_WINDOWS_CERT)
  )

  foreach ($file in $files) {
    Write-Host "Signing: $file"
    & $signtool sign /fd sha256 /tr http://ts.ssl.com /td sha256 /f $pfxPath /p $env:SIGN_WINDOWS_CERT_PASSWORD $file
    if ($LASTEXITCODE -ne 0) { throw "signtool sign failed: $file" }
    & $signtool verify /pa $file
    if ($LASTEXITCODE -ne 0) { throw "signtool verify failed: $file" }
  }
}
finally {
  Remove-Item -LiteralPath $certDir -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Host "Signing done."
