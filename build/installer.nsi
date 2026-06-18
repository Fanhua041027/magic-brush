; Magic Brush (Shenbi Maliang) Installer
; NSIS 3.12+  (UTF-8 with BOM encoding)

Unicode true
ManifestDPIAware true

!define PRODUCT_NAME "Magic Brush"
!define PRODUCT_VERSION "1.0.0.0"
!define PRODUCT_PUBLISHER "Magic Brush contributors"
!define PRODUCT_WEB_SITE "https://github.com/Fanhua041027/magic-brush"
!define PRODUCT_DIR_REGKEY "Software\Microsoft\Windows\CurrentVersion\App Paths\ShenbiMaliang.exe"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"
!define PRODUCT_UNINST_ROOT_KEY "HKLM"

; ---------- Modern UI ----------
!include "MUI2.nsh"

; MUI Settings
!define MUI_ABORTWARNING
!define MUI_ICON "${NSISDIR}\Contrib\Graphics\Icons\modern-install.ico"
!define MUI_UNICON "${NSISDIR}\Contrib\Graphics\Icons\modern-uninstall.ico"

; Welcome page
!insertmacro MUI_PAGE_WELCOME

; Directory selection
!insertmacro MUI_PAGE_DIRECTORY

; Instfiles page
!insertmacro MUI_PAGE_INSTFILES

; Finish page
!define MUI_FINISHPAGE_RUN "$INSTDIR\ShenbiMaliang.exe"
!define MUI_FINISHPAGE_RUN_TEXT "$(launchNow)"
!insertmacro MUI_PAGE_FINISH

; Uninstall pages
!insertmacro MUI_UNPAGE_INSTFILES

; ---------- Languages (AFTER pages) ----------
!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "English"

; ---------- Custom LangStrings ----------
LangString launchNow ${LANG_SIMPCHINESE} "LiJi YunXing Magic Brush"
LangString launchNow ${LANG_ENGLISH} "Run Magic Brush now"

LangString productDesc ${LANG_SIMPCHINESE} "AI Zhi Neng Mian Shi Zhu Shou"
LangString productDesc ${LANG_ENGLISH} "AI Interview Assistant"

LangString desktopShortcut ${LANG_SIMPCHINESE} "Chuang Jian Zhuo Miao Kuai Jie Fang Shi"
LangString desktopShortcut ${LANG_ENGLISH} "Create Desktop Shortcut"

LangString deleteConfig ${LANG_SIMPCHINESE} "Shi Fou Tong Shi Shan Chu Ge Ren Pei Zhi Shu Ju?"
LangString deleteConfig ${LANG_ENGLISH} "Remove personal configuration data as well?"

; ---------- Installer Attributes ----------
Name "${PRODUCT_NAME} ${PRODUCT_VERSION}"
OutFile "MagicBrush_Setup_${PRODUCT_VERSION}.exe"
InstallDir "$PROGRAMFILES64\Magic Brush"
InstallDirRegKey HKLM "${PRODUCT_DIR_REGKEY}" ""
ShowInstDetails show
ShowUnInstDetails show
RequestExecutionLevel admin

Section "MainSection" SEC01
  SetOutPath "$INSTDIR"
  SetOverwrite ifnewer

  ; Main executable
  File "bin\ShenbiMaliang.exe"

  ; Icon resource
  File /oname=icon.ico "windows\icon.ico"

  ; Config example
  File /oname=config.example.json "..\config\config.example.json"

  ; README
  File /oname=README.txt "..\README.md"

  ; Start menu directory
  CreateDirectory "$SMPROGRAMS\Magic Brush"

  ; Main shortcut
  CreateShortCut "$SMPROGRAMS\Magic Brush\Magic Brush.lnk" "$INSTDIR\ShenbiMaliang.exe" "" "$INSTDIR\icon.ico" 0 SW_SHOWNORMAL ALTCONTROL|ALT|F7 "" "$(productDesc)"

  ; Uninstall shortcut
  CreateShortCut "$SMPROGRAMS\Magic Brush\Uninstall.lnk" "$INSTDIR\uninstall.exe" "" "$INSTDIR\icon.ico" 0

  ; Desktop shortcut
  CreateShortCut "$DESKTOP\Magic Brush.lnk" "$INSTDIR\ShenbiMaliang.exe" "" "$INSTDIR\icon.ico" 0

  ; Write uninstaller
  WriteUninstaller "$INSTDIR\uninstall.exe"

  ; Register in Windows Apps & Features
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayName" "${PRODUCT_NAME}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "URLInfoAbout" "${PRODUCT_WEB_SITE}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayIcon" "$INSTDIR\icon.ico"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "UninstallString" "$INSTDIR\uninstall.exe"
  WriteRegDWORD HKLM "${PRODUCT_UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKLM "${PRODUCT_UNINST_KEY}" "NoRepair" 1
  WriteRegStr HKLM "${PRODUCT_DIR_REGKEY}" "" "$INSTDIR\ShenbiMaliang.exe"
SectionEnd

; ---------- Uninstaller ----------
Section Uninstall
  ; Delete program files
  Delete "$INSTDIR\ShenbiMaliang.exe"
  Delete "$INSTDIR\icon.ico"
  Delete "$INSTDIR\config.example.json"
  Delete "$INSTDIR\README.txt"
  Delete "$INSTDIR\uninstall.exe"

  ; Delete shortcuts
  Delete "$SMPROGRAMS\Magic Brush\Magic Brush.lnk"
  Delete "$SMPROGRAMS\Magic Brush\Uninstall.lnk"
  Delete "$DESKTOP\Magic Brush.lnk"
  RMDir "$SMPROGRAMS\Magic Brush"

  ; Ask about config data
  MessageBox MB_YESNO|MB_ICONQUESTION "$(deleteConfig)" IDNO skipConfig
    RMDir /r "$APPDATA\Magic Brush"
  skipConfig:

  ; Remove install dir if empty
  RMDir "$INSTDIR"

  ; Clean registry
  DeleteRegKey HKLM "${PRODUCT_UNINST_KEY}"
  DeleteRegKey HKLM "${PRODUCT_DIR_REGKEY}"
SectionEnd
