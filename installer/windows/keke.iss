; Keke CLI Installer for Windows
; Compile with Inno Setup

#define MyAppName "Keke CLI"
#define MyAppVersion "2.0.0"
#define MyAppPublisher "Aimable2002"
#define MyAppURL "https://github.com/Aimable2002/keke_aia"
#define MyAppExeName "keke.exe"

[Setup]
; NOTE: The value of AppId uniquely identifies this application.
AppId={{F5E4D3C2-B1A0-4F3E-8D2C-1B0A9F8E7D6C}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}/issues
AppUpdatesURL={#MyAppURL}/releases
DefaultDirName={autopf}\Keke
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
OutputDir=..\..\dist
OutputBaseFilename=keke-installer-windows-{#MyAppVersion}
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
ChangesEnvironment=yes
PrivilegesRequired=lowest
ArchitecturesInstallIn64BitMode=x64compatible
ArchitecturesAllowed=x64compatible
UninstallDisplayIcon={app}\{#MyAppExeName}

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
Name: "addtopath"; Description: "Add to PATH environment variable (recommended)"; GroupDescription: "Installation options:"; Flags: checkedonce

[Files]
Source: "..\..\dist\keke.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{group}\{cm:UninstallProgram,{#MyAppName}}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#MyAppExeName}"; Parameters: "--help"; Description: "View help"; Flags: nowait postinstall skipifsilent shellexec unchecked

[Registry]
Root: HKCU; Subkey: "Environment"; ValueType: expandsz; ValueName: "Path"; ValueData: "{olddata};{app}"; Tasks: addtopath; Check: NeedsAddPath(ExpandConstant('{app}'))

[Code]
function NeedsAddPath(Param: string): boolean;
var
  OrigPath: string;
begin
  if not RegQueryStringValue(HKCU, 'Environment', 'Path', OrigPath) then
  begin
    Result := True;
    exit;
  end;
  Result := Pos(';' + Param + ';', ';' + OrigPath + ';') = 0;
end;

function RemovePathEntry(Path: string; Entry: string): string;
var
  Parts: TArrayOfString;
  i: Integer;
  NewPath: string;
begin
  NewPath := '';
  Parts := [];
  
  if Pos(';', Path) > 0 then
  begin
    // Split path manually
    while Length(Path) > 0 do
    begin
      i := Pos(';', Path);
      if i > 0 then
      begin
        SetArrayLength(Parts, GetArrayLength(Parts) + 1);
        Parts[GetArrayLength(Parts) - 1] := Copy(Path, 1, i - 1);
        Delete(Path, 1, i);
      end
      else
      begin
        SetArrayLength(Parts, GetArrayLength(Parts) + 1);
        Parts[GetArrayLength(Parts) - 1] := Path;
        Path := '';
      end;
    end;
    
    // Rebuild path without the entry
    for i := 0 to GetArrayLength(Parts) - 1 do
    begin
      if CompareText(Parts[i], Entry) <> 0 then
      begin
        if NewPath = '' then
          NewPath := Parts[i]
        else
          NewPath := NewPath + ';' + Parts[i];
      end;
    end;
  end
  else
  begin
    if CompareText(Path, Entry) <> 0 then
      NewPath := Path;
  end;
  
  Result := NewPath;
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
  begin
    if WizardIsTaskSelected('addtopath') then
    begin
      // Environment variables updated
    end;
  end;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  EnvPath: string;
  AppPath: string;
  NewPath: string;
begin
  if CurUninstallStep = usPostUninstall then
  begin
    AppPath := ExpandConstant('{app}');
    if RegQueryStringValue(HKCU, 'Environment', 'Path', EnvPath) then
    begin
      NewPath := RemovePathEntry(EnvPath, AppPath);
      if NewPath <> EnvPath then
      begin
        RegWriteStringValue(HKCU, 'Environment', 'Path', NewPath);
      end;
    end;
  end;
end;