; bat2exe Installer for Windows
; Inno Setup Script

#define MyAppName "bat2exe"
#define MyAppVersion "1.1.0"
#define MyAppPublisher "bat2exe"
#define MyAppURL "https://github.com/bat2exe"
#define MyAppExeName "bat2exe.exe"

[Setup]
AppId={{8A2E3B1C-5D4F-4A6B-9C8D-1E2F3A4B5C6D}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
LicenseFile=..\LICENSE
OutputDir=.\output
OutputBaseFilename=bat2exe-setup-{#MyAppVersion}
Compression=lzma
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=admin
ChangesEnvironment=yes
UninstallDisplayIcon={app}\{#MyAppExeName}

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "contextmenu"; Description: "Add &Convert to EXE with bat2exe to .bat file context menu"; GroupDescription: "Context menu integration:"

[Files]
Source: "..\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\test_simple.bat"; DestDir: "{app}\examples"; Flags: ignoreversion
Source: "..\demo_param.bat"; DestDir: "{app}\examples"; Flags: ignoreversion

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{group}\{cm:UninstallProgram,{#MyAppName}}"; Filename: "{uninstallexe}"
Name: "{group}\Examples"; Filename: "{app}\examples"

[Registry]
; Add to PATH
Root: HKLM; Subkey: "SYSTEM\CurrentControlSet\Control\Session Manager\Environment"; \
    ValueType: expandsz; ValueName: "PATH"; ValueData: "{olddata};{app}"; \
    Check: NeedsAddPath('{app}')

; Context menu for .bat files
Root: HKCR; Subkey: "batfile\shell\bat2exe"; ValueType: string; \
    ValueName: ""; ValueData: "Convert to EXE with bat2exe"; \
    Tasks: contextmenu; Flags: uninsdeletekey
Root: HKCR; Subkey: "batfile\shell\bat2exe"; ValueType: string; \
    ValueName: "Icon"; ValueData: "{app}\{#MyAppExeName},0"; \
    Tasks: contextmenu
Root: HKCR; Subkey: "batfile\shell\bat2exe\command"; ValueType: string; \
    ValueName: ""; ValueData: """{app}\{#MyAppExeName}"" -input ""%1"""; \
    Tasks: contextmenu; Flags: uninsdeletekey

; Also handle .cmd files
Root: HKCR; Subkey: "cmdfile\shell\bat2exe"; ValueType: string; \
    ValueName: ""; ValueData: "Convert to EXE with bat2exe"; \
    Tasks: contextmenu; Flags: uninsdeletekey
Root: HKCR; Subkey: "cmdfile\shell\bat2exe"; ValueType: string; \
    ValueName: "Icon"; ValueData: "{app}\{#MyAppExeName},0"; \
    Tasks: contextmenu
Root: HKCR; Subkey: "cmdfile\shell\bat2exe\command"; ValueType: string; \
    ValueName: ""; ValueData: """{app}\{#MyAppExeName}"" -input ""%1"""; \
    Tasks: contextmenu; Flags: uninsdeletekey

[Code]

function NeedsAddPath(Param: string): boolean;
var
  OrigPath: string;
begin
  if not RegQueryStringValue(HKLM, 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment', 'PATH', OrigPath) then
  begin
    Result := True;
    exit;
  end;
  Result := Pos(';' + Param + ';', ';' + OrigPath + ';') = 0;
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  ResultCode: Integer;
begin
  if CurStep = ssPostInstall then
  begin
    if WizardIsTaskSelected('contextmenu') then
    begin
      MsgBox('Context menu entry added!' + #13#10 +
             'Right-click any .bat or .cmd file and select "Convert to EXE with bat2exe".',
             mbInformation, MB_OK);
    end;
  end;
end;

[Run]
Filename: "{app}\{#MyAppExeName}"; Parameters: "--version"; \
    Description: "Verify installation"; Flags: postinstall nowait skipifsilent shellexec
