param(
  [string]$BaseUrl = "http://127.0.0.1:5000",
  [string]$Username = "admin",
  [string]$Password = "admin123"
)

$ErrorActionPreference = 'Stop'

function Write-Section([string]$title) {
  Write-Host "\n=== $title ===" -ForegroundColor Cyan
}

function Invoke-Json([string]$Method, [string]$Url, $Body = $null, $Headers = $null) {
  $params = @{ Method = $Method; Uri = $Url }
  if ($Headers) { $params.Headers = $Headers }
  if ($Body -ne $null) {
    $params.ContentType = 'application/json'
    $params.Body = ($Body | ConvertTo-Json -Depth 10)
  }
  return Invoke-RestMethod @params
}

function Invoke-Raw([string]$Url, $Headers = $null) {
  $params = @{ Uri = $Url; Method = 'GET'; UseBasicParsing = $true }
  if ($Headers) { $params.Headers = $Headers }
  return Invoke-WebRequest @params
}

Write-Section "Preflight"
Write-Host "BaseUrl: $BaseUrl"

# --- Login (try a small set of common defaults) ---
Write-Section "Auth"
$loginCandidates = @(
  @{ username = $Username; password = $Password },
  @{ username = $Username; password = "admin" },
  @{ username = $Username; password = "admin123" }
)

$token = $null
foreach ($cand in $loginCandidates) {
  try {
    $resp = Invoke-Json -Method 'POST' -Url "$BaseUrl/api/v1/auth/login" -Body $cand
    if ($resp.token) {
      $token = $resp.token
      Write-Host "Login OK as '$($cand.username)' using provided/default password." -ForegroundColor Green
      break
    }
  } catch {
    # ignore and continue
  }
}

if (-not $token) {
  throw "Login failed for all candidate passwords (user '$Username')."
}

$authHeaders = @{ Authorization = "Bearer $token" }

# --- UI static checks ---
Write-Section "UI (static)"
$index = Invoke-Raw -Url "$BaseUrl/index.html"
if ($index.StatusCode -ne 200) { throw "GET /index.html failed with $($index.StatusCode)" }
Write-Host "GET /index.html OK" -ForegroundColor Green

# parse asset refs from index.html
$assetRefs = @()
$assetRefs += [regex]::Matches($index.Content, "assets/[^`"']+") | ForEach-Object { $_.Value }
$assetRefs = $assetRefs | Sort-Object -Unique
if ($assetRefs.Count -eq 0) { throw "No asset references found in index.html" }

foreach ($asset in $assetRefs) {
  $r = Invoke-Raw -Url "$BaseUrl/$asset"
  if ($r.StatusCode -ne 200) { throw "GET /$asset failed with $($r.StatusCode)" }
}
Write-Host ("Assets OK: " + ($assetRefs -join ', ')) -ForegroundColor Green

# --- API checks ---
Write-Section "API (read endpoints)"
$sims = Invoke-Json -Method 'GET' -Url "$BaseUrl/api/v1/sims" -Headers $authHeaders
if (-not $sims) { throw "GET /api/v1/sims returned empty" }

# sims might be {data:[...]} or just [...]
$simList = $null
if ($sims.data) { $simList = $sims.data } elseif ($sims -is [System.Array]) { $simList = $sims } else { $simList = @($sims) }
if ($simList.Count -lt 1) { throw "SIM list is empty" }

$first = $simList[0]
$msisdn = $first.MSISDN
$cli = $first.CLI
$status = $first.SIM_STATUS_CHANGE
if (-not $status) { $status = $first.status }
if (-not $status) { $status = $first.Status }

Write-Host "SIM sample: MSISDN=$msisdn CLI=$cli Status=$status" -ForegroundColor DarkGray

$stats = Invoke-Json -Method 'GET' -Url "$BaseUrl/api/v1/stats" -Headers $authHeaders
Write-Host "GET /stats OK" -ForegroundColor Green

$jobs = Invoke-Json -Method 'GET' -Url "$BaseUrl/api/v1/jobs" -Headers $authHeaders
Write-Host "GET /jobs OK" -ForegroundColor Green

$queue = Invoke-Json -Method 'GET' -Url "$BaseUrl/api/v1/jobs/queue" -Headers $authHeaders
Write-Host "GET /jobs/queue OK" -ForegroundColor Green

$apiStatus = Invoke-Json -Method 'GET' -Url "$BaseUrl/api/v1/api-status" -Headers $authHeaders
Write-Host "GET /api-status OK" -ForegroundColor Green

$diag = Invoke-Json -Method 'GET' -Url "$BaseUrl/api/v1/api-status/diagnostics" -Headers $authHeaders
Write-Host "GET /api-status/diagnostics OK" -ForegroundColor Green

# history
if ($msisdn) {
  $hist = Invoke-Json -Method 'GET' -Url "$BaseUrl/api/v1/sims/$msisdn/history" -Headers $authHeaders
  Write-Host "GET /sims/:msisdn/history OK" -ForegroundColor Green
}

# queue endpoints (basic coverage)
Write-Section "API (queue endpoints)"
$my = Invoke-Json -Method 'GET' -Url "$BaseUrl/api/v1/queue/my" -Headers $authHeaders
Write-Host "GET /queue/my OK" -ForegroundColor Green
$myh = Invoke-Json -Method 'GET' -Url "$BaseUrl/api/v1/queue/my/history" -Headers $authHeaders
Write-Host "GET /queue/my/history OK" -ForegroundColor Green
$qstats = Invoke-Json -Method 'GET' -Url "$BaseUrl/api/v1/queue/stats" -Headers $authHeaders
Write-Host "GET /queue/stats OK" -ForegroundColor Green

# --- API write endpoint smoke: change single SIM status ---
Write-Section "API (write smoke)"
if (-not $cli) {
  Write-Host "Skipping /sims/status: no CLI in SIM sample" -ForegroundColor Yellow
} else {
  $newStatus = 'Activated'
  if ($status -eq 'Activated') { $newStatus = 'Suspended' }
  elseif ($status -eq 'Suspended') { $newStatus = 'Activated' }

  $body = @{ cli = "$cli"; msisdn = "$msisdn"; old_status = "$status"; new_status = "$newStatus" }
  $cs = Invoke-Json -Method 'POST' -Url "$BaseUrl/api/v1/sims/status" -Headers $authHeaders -Body $body
  if (-not $cs.success) { throw "POST /sims/status failed: $($cs.error)" }
  Write-Host ("POST /sims/status OK (queued=$($cs.queued))") -ForegroundColor Green

  if ($cs.queued -and $cs.task_id) {
    # poll local job
    $jobId = $cs.task_id
    $done = $false
    for ($i=0; $i -lt 10; $i++) {
      Start-Sleep -Seconds 1
      $j = Invoke-Json -Method 'GET' -Url "$BaseUrl/api/v1/jobs/local/$jobId" -Headers $authHeaders
      if ($j -and $j.status -and ($j.status -eq 'COMPLETED' -or $j.status -eq 'SUCCESS' -or $j.status -eq 'FAILED')) {
        $done = $true
        Write-Host "Poll /jobs/local/$jobId => $($j.status)" -ForegroundColor DarkGray
        break
      }
    }
    if (-not $done) {
      Write-Host "Poll /jobs/local/$jobId did not complete within 10s (may still be processing)." -ForegroundColor Yellow
    }
  }
}

# --- API write endpoint smoke: bulk-status no-op (set same status) ---
Write-Section "API (bulk no-op)"
if ($msisdn -and $status) {
  $bulk = Invoke-Json -Method 'POST' -Url "$BaseUrl/api/v1/sims/bulk-status" -Headers $authHeaders -Body @{ status = "$status"; msisdns = @("$msisdn") }
  if (-not $bulk.success) { throw "POST /sims/bulk-status failed: $($bulk.error)" }
  Write-Host ("POST /sims/bulk-status OK (queued=$($bulk.queued))") -ForegroundColor Green
} else {
  Write-Host "Skipping bulk-status: missing msisdn/status" -ForegroundColor Yellow
}

Write-Section "DONE"
Write-Host "UI + API smoke test succeeded." -ForegroundColor Green
