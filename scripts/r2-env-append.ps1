# One-shot PO helper (WI-D1): append R2 credentials to VPS /opt/avibe/.env.
# Run once from the repo root after: git pull
#   powershell -NoProfile -ExecutionPolicy Bypass -File scripts\r2-env-append.ps1
# Values are read hidden, never printed, and scrubbed after use. Re-runnable.
$ErrorActionPreference = 'Stop'
$Key = "$HOME\.ssh\avibe_vultr"
$Vps = 'root@158.247.219.159'

if (-not (Test-Path $Key)) { Write-Host "NO-SSH-KEY: $Key"; exit 1 }

$endpoint = 'https://f5c363135efe9b22ca66eb39f2f8e1da.r2.cloudflarestorage.com'
$kid = Read-Host 'R2_ACCESS_KEY_ID 붙여넣기'
$sec = Read-Host 'R2_SECRET_ACCESS_KEY 붙여넣기(입력 안 보임)' -AsSecureString
if (-not $kid) { Write-Host 'EMPTY-KEY-ID-ABORT'; exit 1 }
$pwlen = $sec.Length
if ($pwlen -eq 0) { Write-Host 'EMPTY-SECRET-ABORT'; exit 1 }

$bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($sec)
try {
  $plain = [Runtime.InteropServices.Marshal]::PtrToStringAuto($bstr)
  $snippet = "R2_ENDPOINT=$endpoint`nR2_ACCESS_KEY_ID=$kid`nR2_SECRET_ACCESS_KEY=$plain`nR2_STAGING_BUCKET=staging-private`nR2_PUBLIC_HLS_BUCKET=public-hls`n"
  $tmp = Join-Path $env:TEMP 'avibe-r2env'
  [IO.File]::WriteAllText($tmp, $snippet)
} finally {
  [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)
  $plain = $null
  $sec.Dispose()
}
scp -i $Key $tmp "${Vps}:/tmp/avibe-r2env"
if (-not $?) { Remove-Item $tmp -Force; Write-Host 'SCP-FAILED'; exit 1 }
Remove-Item $tmp -Force
ssh -i $Key -o BatchMode=yes $Vps 'cp -p /opt/avibe/.env /opt/avibe/.env.bak-r2-pre && grep -v ^R2_ /opt/avibe/.env.bak-r2-pre > /opt/avibe/.env && cat /tmp/avibe-r2env >> /opt/avibe/.env && chmod 600 /opt/avibe/.env && rm -f /tmp/avibe-r2env && grep ^R2_ /opt/avibe/.env | cut -d= -f1'
if ($?) { Write-Host 'ENV-OK (5 R2 lines above = success)' } else { Write-Host 'SSH-FAILED'; exit 1 }
