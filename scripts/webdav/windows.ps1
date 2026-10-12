# Real Windows WebClient acceptance; run on a disposable FileX draft over trusted HTTPS.
param([Parameter(Mandatory=$true)][uri]$Url)
$ErrorActionPreference = 'Stop'
if ($env:OS -ne 'Windows_NT') { throw 'This check requires Windows.' }
if ($Url.Scheme -ne 'https' -or $Url.UserInfo -or $Url.Query -or $Url.Fragment -or $Url.AbsolutePath -notmatch '^/api/v1/webdav/[0-9a-f-]+/$') {
    throw 'Supply the HTTPS draft URL shown in FileX, without credentials or query.'
}
$service = Get-Service WebClient -ErrorAction Stop
if ($service.Status -ne 'Running') { throw 'WebClient is not running. Ask the workstation administrator to enable it; this script does not change system settings.' }
$credential = Get-Credential -UserName filex -Message 'Enter the separate FileX WebDAV connection password'
$authority = $Url.DnsSafeHost + '@SSL'
if (-not $Url.IsDefaultPort) { $authority += '@' + $Url.Port }
$unc = '\\' + $authority + '\DavWWWRoot' + $Url.AbsolutePath.Replace('/', '\').TrimEnd('\')
$drive = 'FX' + [guid]::NewGuid().ToString('N').Substring(0,6)
$folder = $null
try {
    New-PSDrive -Name $drive -PSProvider FileSystem -Root $unc -Credential $credential | Out-Null
    $folder = Join-Path ($drive + ':\') ('acceptance-' + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $folder | Out-Null
    $path = Join-Path $folder 'договор.txt'
    $bytes = [Text.Encoding]::UTF8.GetBytes(('Native Windows WebClient test' + "`n") * 180000)
    [IO.File]::WriteAllBytes($ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($path), $bytes)
    $expected = [BitConverter]::ToString([Security.Cryptography.SHA256]::Create().ComputeHash($bytes)).Replace('-','').ToLowerInvariant()
    if ((Get-FileHash -Path $path -Algorithm SHA256).Hash.ToLowerInvariant() -ne $expected) { throw 'Uploaded bytes differ.' }
    Set-Content -Path $path -Value 'Windows replacement' -Encoding UTF8
    Rename-Item -Path $path -NewName 'готово.txt'
    $renamed = Join-Path $folder 'готово.txt'
    if ((Get-Content -Raw $renamed).Trim() -ne 'Windows replacement') { throw 'Replacement or rename failed.' }
    Remove-Item $renamed
    Remove-Item $folder
    $folder = $null
    [pscustomobject]@{ status='passed'; client='Windows WebClient'; os=[Environment]::OSVersion.VersionString; uploadedBytes=$bytes.Length; sha256=$expected; operations=@('connect','mkdir','upload','read','replace','rename','delete','rmdir') } | ConvertTo-Json
} finally {
    if ($folder -and (Test-Path $folder)) { Remove-Item $folder -Recurse }
    Remove-PSDrive $drive -ErrorAction SilentlyContinue
}
