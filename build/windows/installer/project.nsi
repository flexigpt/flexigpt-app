Unicode true

!cd "${__FILEDIR__}"
!include "wails_tools.nsh"

; The default NSIS zlib compressor is materially larger for the embedded
; Go executable and frontend asset payload.
SetCompressor /SOLID lzma

VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName" "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion" "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion" "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright" "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName" "${INFO_PRODUCTNAME}"

ManifestDPIAware true

!include "MUI.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_ABORTWARNING

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

; NSIS creates the uninstaller during compilation.
; Sign that executable before NSIS embeds it in the installer.
!ifdef ARG_FLEXIGPT_SIGN_UNINSTALLER
  !ifndef ARG_FLEXIGPT_POWERSHELL
    !error "ARG_FLEXIGPT_POWERSHELL is required when signing the uninstaller"
  !endif
  !ifndef ARG_FLEXIGPT_SIGN_SCRIPT
    !error "ARG_FLEXIGPT_SIGN_SCRIPT is required when signing the uninstaller"
  !endif
  !ifndef ARG_FLEXIGPT_SIGN_LOG
    !error "ARG_FLEXIGPT_SIGN_LOG is required when signing the uninstaller"
  !endif
  !uninstfinalize '"${ARG_FLEXIGPT_POWERSHELL}" -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "${ARG_FLEXIGPT_SIGN_SCRIPT}" -FilePath "%1" -LogPath "${ARG_FLEXIGPT_SIGN_LOG}"' = 0
!endif

Name "${INFO_PRODUCTNAME}"
OutFile "${ARG_FLEXIGPT_INSTALLER}"
InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
ShowInstDetails show

Function .onInit
  !insertmacro wails.checkArchitecture
FunctionEnd

Section
  !insertmacro wails.setShellContext
  !insertmacro wails.webview2runtime

  SetOutPath "$INSTDIR"
  !insertmacro wails.files

  SetOutPath "$INSTDIR\licenses"
  File "..\..\licenses\PROJECT_LICENSE.txt"
  File "..\..\licenses\THIRD_PARTY_NOTICES.txt"
  File "..\..\licenses\go-dependency-licenses.txt"
  File "..\..\licenses\js-dependency-licenses.txt"
  SetOutPath "$INSTDIR"

  CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
  CreateShortcut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

  !insertmacro wails.associateFiles
  !insertmacro wails.associateCustomProtocols
  !insertmacro wails.writeUninstaller
SectionEnd

Section "uninstall"
  !insertmacro wails.setShellContext

  Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
  Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

  !insertmacro wails.unassociateFiles
  !insertmacro wails.unassociateCustomProtocols
  !insertmacro wails.deleteUninstaller

  ; Deliberately do not delete application data from AppData.
  RMDir /r "$INSTDIR"
SectionEnd
