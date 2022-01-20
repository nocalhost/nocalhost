#define AppName "nocalhost"
#define AppPublisher "nocalhost.dev"
#define AppURL "https://nocalhost.dev"
#define AppExeName "nhctl.exe"

[Setup]
; NOTE: The value of AppId uniquely identifies this application. Do not use the same AppId value in installers for other applications.
; (To generate a new GUID, click Tools | Generate GUID inside the IDE.)
AppId={{CF1B1587-7851-4B9C-8997-41C91A81B24B}
AppName={#AppName}
AppVersion={#AppVersion}
OutputBaseFilename=NocalhostInstaller
OutputDir=..\..\build
AppPublisher={#AppPublisher}
AppPublisherURL={#AppURL}
AppSupportURL={#AppURL}
AppUpdatesURL={#AppURL}
DefaultDirName={autopf}\{#AppName}
DisableDirPage=yes
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
LicenseFile=..\..\LICENSE
; Uncomment the following line to run in non administrative install mode (install for current user only.)
;PrivilegesRequired=lowest
Compression=lzma
SolidCompression=yes
WizardStyle=modern
; Tell Windows Explorer to reload the environment
ChangesEnvironment=yes

[Files]
Source: "..\..\build\{#AppExeName}"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\build\kubectl.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\build\windows-amd64\helm.exe"; DestDir: "{app}"; Flags: ignoreversion
; NOTE: Don't use "Flags: ignoreversion" on any shared system files

[Tasks]
Name: "addtopath"; Description: "Add to PATH (requires shell restart)"; GroupDescription: "Other:"

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\{#AppExeName}"
Name: "{group}\{cm:UninstallProgram,{#AppName}}"; Filename: "{uninstallexe}"

[Registry]
#define SoftwareClassesRootKey "HKLM"

; Environment
#define EnvironmentRootKey "HKLM"
#define EnvironmentKey "System\CurrentControlSet\Control\Session Manager\Environment"
#define Uninstall64RootKey "HKLM64"
#define Uninstall32RootKey "HKLM32"

Root: {#EnvironmentRootKey}; Subkey: "{#EnvironmentKey}"; ValueType: expandsz; ValueName: "Path"; ValueData: "{olddata};{app}"; Tasks: addtopath; Check: NeedsAddPath(ExpandConstant('{app}'))



[Code]
function NeedsAddPath(Param: string): boolean;
var
  OrigPath: string;
begin
  if not RegQueryStringValue({#EnvironmentRootKey}, '{#EnvironmentKey}', 'Path', OrigPath)
  then begin
    Result := True;
    exit;
  end;
  Result := Pos(';' + Param + ';', ';' + OrigPath + ';') = 0;
end;


// https://stackoverflow.com/a/23838239/261019
procedure Explode(var Dest: TArrayOfString; Text: String; Separator: String);
var
  i, p: Integer;
begin
  i := 0;
  repeat
    SetArrayLength(Dest, i+1);
    p := Pos(Separator,Text);
    if p > 0 then begin
      Dest[i] := Copy(Text, 1, p-1);
      Text := Copy(Text, p + Length(Separator), Length(Text));
      i := i + 1;
    end else begin
      Dest[i] := Text;
      Text := '';
    end;
  until Length(Text)=0;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  Path: string;
  ThisAppPath: string;
  Parts: TArrayOfString;
  NewPath: string;
  i: Integer;
begin
  if not CurUninstallStep = usUninstall then begin
    exit;
  end;
  if not RegQueryStringValue({#EnvironmentRootKey}, '{#EnvironmentKey}', 'Path', Path)
  then begin
    exit;
  end;
  NewPath := '';
  ThisAppPath := ExpandConstant('{app}')
  Explode(Parts, Path, ';');
  for i:=0 to GetArrayLength(Parts)-1 do begin
    if CompareText(Parts[i], ThisAppPath) <> 0 then begin
      NewPath := NewPath + Parts[i];
      if i < GetArrayLength(Parts) - 1 then begin
        NewPath := NewPath + ';';
      end;
    end;
  end;
  RegWriteExpandStringValue({#EnvironmentRootKey}, '{#EnvironmentKey}', 'Path', NewPath);
end;