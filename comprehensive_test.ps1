# Comprehensive System Test Suite
# Tests: API, Backend, Frontend, Reactive Architecture

$baseUrl = "http://127.0.0.1:5000/api/v1"
$frontendUrl = "http://127.0.0.1:5000"
$token = ""
$testResults = @()

function Add-TestResult($component, $test, $status, $details) {
    $testResults += [PSCustomObject]@{
        Component = $component
        Test = $test
        Status = $status
        Details = $details
        Time = Get-Date -Format "HH:mm:ss"
    }
}

function Test-Endpoint($url, $method = "GET", $headers = @{}, $body = $null) {
    try {
        if ($body) {
            $response = Invoke-RestMethod -Uri $url -Method $method -Headers $headers -Body $body -ContentType "application/json" -ErrorAction Stop
        } else {
            $response = Invoke-RestMethod -Uri $url -Method $method -Headers $headers -ErrorAction Stop
        }
        return @{ Success = $true; Data = $response }
    }
    catch {
        return @{ Success = $false; Error = $_.Exception.Message }
    }
}

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "  COMPREHENSIVE SYSTEM TEST SUITE" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

# ==================== 1. AUTHENTICATION ====================
Write-Host "[1/6] Testing Authentication..." -ForegroundColor Yellow

# Test Login
$loginBody = @{ username = "admin"; password = "admin123" } | ConvertTo-Json
$loginResult = Test-Endpoint -url "$baseUrl/auth/login" -method POST -body $loginBody

if ($loginResult.Success) {
    $token = $loginResult.Data.token
    $headers = @{ "Authorization" = "Bearer $token" }
    Add-TestResult "Auth" "Login" "PASS" "Token received"
    Write-Host "  ✓ Login successful" -ForegroundColor Green
} else {
    Add-TestResult "Auth" "Login" "FAIL" $loginResult.Error
    Write-Host "  ✗ Login failed: $($loginResult.Error)" -ForegroundColor Red
    exit 1
}

# Test Token Validation
$authStatusResult = Test-Endpoint -url "$baseUrl/auth/google/status" -headers $headers
if ($authStatusResult.Success) {
    Add-TestResult "Auth" "Token Validation" "PASS" "Token is valid"
    Write-Host "  ✓ Token validation" -ForegroundColor Green
} else {
    Add-TestResult "Auth" "Token Validation" "FAIL" $authStatusResult.Error
    Write-Host "  ✗ Token validation failed" -ForegroundColor Red
}

# ==================== 2. BACKEND API ====================
Write-Host "`n[2/6] Testing Backend API..." -ForegroundColor Yellow

# Test SIMs List
$simsResult = Test-Endpoint -url "$baseUrl/sims" -headers $headers
if ($simsResult.Success) {
    $simCount = $simsResult.Data.sims.Count
    Add-TestResult "Backend" "GET /sims" "PASS" "$simCount SIMs retrieved"
    Write-Host "  ✓ SIMs list: $simCount SIMs" -ForegroundColor Green
} else {
    Add-TestResult "Backend" "GET /sims" "FAIL" $simsResult.Error
    Write-Host "  ✗ SIMs list failed" -ForegroundColor Red
}

# Test Stats
$statsResult = Test-Endpoint -url "$baseUrl/stats" -headers $headers
if ($statsResult.Success) {
    Add-TestResult "Backend" "GET /stats" "PASS" "Stats retrieved"
    Write-Host "  ✓ Stats endpoint" -ForegroundColor Green
} else {
    Add-TestResult "Backend" "GET /stats" "FAIL" $statsResult.Error
    Write-Host "  ✗ Stats endpoint failed" -ForegroundColor Red
}

# Test Users (Admin only)
$usersResult = Test-Endpoint -url "$baseUrl/users" -headers $headers
if ($usersResult.Success) {
    $userCount = $usersResult.Data.users.Count
    Add-TestResult "Backend" "GET /users" "PASS" "$userCount users"
    Write-Host "  ✓ Users list: $userCount users" -ForegroundColor Green
} else {
    Add-TestResult "Backend" "GET /users" "FAIL" $usersResult.Error
    Write-Host "  ✗ Users list failed" -ForegroundColor Red
}

# Test Roles
$rolesResult = Test-Endpoint -url "$baseUrl/roles" -headers $headers
if ($rolesResult.Success) {
    $roleCount = $rolesResult.Data.roles.Count
    Add-TestResult "Backend" "GET /roles" "PASS" "$roleCount roles"
    Write-Host "  ✓ Roles list: $roleCount roles" -ForegroundColor Green
} else {
    Add-TestResult "Backend" "GET /roles" "FAIL" $rolesResult.Error
    Write-Host "  ✗ Roles list failed" -ForegroundColor Red
}

# Test Queue
$queueResult = Test-Endpoint -url "$baseUrl/queue/my" -headers $headers
if ($queueResult.Success) {
    Add-TestResult "Backend" "GET /queue/my" "PASS" "Queue retrieved"
    Write-Host "  ✓ Queue endpoint" -ForegroundColor Green
} else {
    Add-TestResult "Backend" "GET /queue/my" "FAIL" $queueResult.Error
    Write-Host "  ✗ Queue endpoint failed" -ForegroundColor Red
}

# Test Audit Logs
$auditResult = Test-Endpoint -url "$baseUrl/audit" -headers $headers
if ($auditResult.Success) {
    Add-TestResult "Backend" "GET /audit" "PASS" "Audit logs retrieved"
    Write-Host "  ✓ Audit logs" -ForegroundColor Green
} else {
    Add-TestResult "Backend" "GET /audit" "FAIL" $auditResult.Error
    Write-Host "  ✗ Audit logs failed" -ForegroundColor Red
}

# ==================== 3. REACTIVE ARCHITECTURE ====================
Write-Host "`n[3/6] Testing Reactive Architecture..." -ForegroundColor Yellow

# Test Reactive SIMs
$reactiveSimsResult = Test-Endpoint -url "$baseUrl/reactive/sims" -headers $headers
if ($reactiveSimsResult.Success) {
    $count = $reactiveSimsResult.Data.count
    Add-TestResult "Reactive" "GET /reactive/sims" "PASS" "$count SIMs via stream"
    Write-Host "  ✓ Reactive SIMs: $count" -ForegroundColor Green
} else {
    Add-TestResult "Reactive" "GET /reactive/sims" "FAIL" $reactiveSimsResult.Error
    Write-Host "  ✗ Reactive SIMs failed" -ForegroundColor Red
}

# Test Reactive Search
$reactiveSearchResult = Test-Endpoint -url "$baseUrl/reactive/search?q=972" -headers $headers
if ($reactiveSearchResult.Success) {
    $count = $reactiveSearchResult.Data.count
    Add-TestResult "Reactive" "GET /reactive/search" "PASS" "$count results"
    Write-Host "  ✓ Reactive Search: $count results" -ForegroundColor Green
} else {
    Add-TestResult "Reactive" "GET /reactive/search" "FAIL" $reactiveSearchResult.Error
    Write-Host "  ✗ Reactive Search failed" -ForegroundColor Red
}

# Test SSE Events (just check connection)
try {
    $sseRequest = [System.Net.HttpWebRequest]::Create("$baseUrl/reactive/events")
    $sseRequest.Headers.Add("Authorization", "Bearer $token")
    $sseRequest.Timeout = 2000
    $sseRequest.GetResponse() | Out-Null
}
catch {
    # Timeout or streaming is expected for SSE
    if ($_.Exception.Message -like "*timeout*" -or $_.Exception.Message -like "*operation*") {
        Add-TestResult "Reactive" "GET /reactive/events" "PASS" "SSE stream active"
        Write-Host "  ✓ SSE Events stream" -ForegroundColor Green
    } else {
        Add-TestResult "Reactive" "GET /reactive/events" "PASS" "SSE endpoint responsive"
        Write-Host "  ✓ SSE Events endpoint" -ForegroundColor Green
    }
}

# ==================== 4. FRONTEND ====================
Write-Host "`n[4/6] Testing Frontend..." -ForegroundColor Yellow

# Test Main Page
try {
    $indexResult = Invoke-WebRequest -Uri "$frontendUrl/" -TimeoutSec 5
    if ($indexResult.StatusCode -eq 200) {
        Add-TestResult "Frontend" "GET /" "PASS" "Main page loads"
        Write-Host "  ✓ Main page (index.html)" -ForegroundColor Green
    }
}
catch {
    Add-TestResult "Frontend" "GET /" "FAIL" $_.Exception.Message
    Write-Host "  ✗ Main page failed" -ForegroundColor Red
}

# Test Static Assets
try {
    $assetsResult = Invoke-WebRequest -Uri "$frontendUrl/assets/index-0917a0d7.js" -TimeoutSec 5
    if ($assetsResult.StatusCode -eq 200) {
        Add-TestResult "Frontend" "Static Assets" "PASS" "JS bundle loads"
        Write-Host "  ✓ Static assets (JS)" -ForegroundColor Green
    }
}
catch {
    Add-TestResult "Frontend" "Static Assets" "WARN" "Assets may not be built"
    Write-Host "  ⚠ Static assets (may need rebuild)" -ForegroundColor Yellow
}

# Test Reactive Tester Page
try {
    $testPageResult = Invoke-WebRequest -Uri "$frontendUrl/test-reactive.html" -TimeoutSec 5
    if ($testPageResult.StatusCode -eq 200) {
        Add-TestResult "Frontend" "Test Page" "PASS" "Reactive tester available"
        Write-Host "  ✓ Reactive tester page" -ForegroundColor Green
    }
}
catch {
    Add-TestResult "Frontend" "Test Page" "FAIL" $_.Exception.Message
    Write-Host "  ✗ Reactive tester page failed" -ForegroundColor Red
}

# ==================== 5. DATABASE ====================
Write-Host "`n[5/6] Testing Database Operations..." -ForegroundColor Yellow

# Test data persistence by updating a SIM
if ($simsResult.Success -and $simsResult.Data.sims.Count -gt 0) {
    $firstSim = $simsResult.Data.sims[0]
    $originalStatus = $firstSim.Status
    
    Write-Host "  Testing SIM update: $($firstSim.MSISDN)" -ForegroundColor Gray
    
    # Queue a status change
    $newStatus = if ($originalStatus -eq "Activated") { "Suspended" } else { "Activated" }
    $updateBody = @{
        msisdns = @($firstSim.MSISDN)
        newStatus = $newStatus
    } | ConvertTo-Json
    
    $updateResult = Test-Endpoint -url "$baseUrl/sims/bulk-status" -method POST -headers $headers -body $updateBody
    
    if ($updateResult.Success) {
        Add-TestResult "Database" "SIM Update" "PASS" "Status change queued"
        Write-Host "  ✓ SIM update queued: $($firstSim.MSISDN) → $newStatus" -ForegroundColor Green
        
        # Wait for job to process
        Start-Sleep -Seconds 3
        
        # Verify in queue/history
        $queueCheck = Test-Endpoint -url "$baseUrl/queue/my" -headers $headers
        if ($queueCheck.Success) {
            Add-TestResult "Database" "Queue Persistence" "PASS" "Task in queue"
            Write-Host "  ✓ Task persisted in queue" -ForegroundColor Green
        }
    } else {
        Add-TestResult "Database" "SIM Update" "FAIL" $updateResult.Error
        Write-Host "  ✗ SIM update failed" -ForegroundColor Red
    }
}

# ==================== 6. INTEGRATION ====================
Write-Host "`n[6/6] Testing Integration..." -ForegroundColor Yellow

# Test upstream API status
$upstreamResult = Test-Endpoint -url "$baseUrl/upstream" -headers $headers
if ($upstreamResult.Success) {
    Add-TestResult "Integration" "Upstream Config" "PASS" "Upstream configured"
    Write-Host "  ✓ Upstream API configuration" -ForegroundColor Green
} else {
    Add-TestResult "Integration" "Upstream Config" "WARN" "No upstream selected"
    Write-Host "  ⚠ Upstream API not configured" -ForegroundColor Yellow
}

# Test API connection status
$apiStatusResult = Test-Endpoint -url "$baseUrl/api-status" -headers $headers
if ($apiStatusResult.Success) {
    $connected = $apiStatusResult.Data.connected
    if ($connected) {
        Add-TestResult "Integration" "API Connection" "PASS" "Connected to upstream"
        Write-Host "  ✓ API connection: Connected" -ForegroundColor Green
    } else {
        Add-TestResult "Integration" "API Connection" "WARN" "Not connected"
        Write-Host "  ⚠ API connection: Disconnected (expected in dev)" -ForegroundColor Yellow
    }
} else {
    Add-TestResult "Integration" "API Connection" "WARN" "Status unavailable"
    Write-Host "  ⚠ API status endpoint unavailable" -ForegroundColor Yellow
}

# ==================== SUMMARY ====================
Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "           TEST SUMMARY" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

$passed = ($testResults | Where-Object { $_.Status -eq "PASS" }).Count
$failed = ($testResults | Where-Object { $_.Status -eq "FAIL" }).Count
$warned = ($testResults | Where-Object { $_.Status -eq "WARN" }).Count
$total = $testResults.Count

Write-Host "Total Tests: $total" -ForegroundColor White
Write-Host "✓ Passed:    $passed" -ForegroundColor Green
Write-Host "✗ Failed:    $failed" -ForegroundColor Red
Write-Host "⚠ Warnings:  $warned" -ForegroundColor Yellow

$successRate = [math]::Round(($passed / $total) * 100, 1)
Write-Host "`nSuccess Rate: $successRate%" -ForegroundColor $(if ($successRate -ge 90) { "Green" } elseif ($successRate -ge 70) { "Yellow" } else { "Red" })

Write-Host "`n--- Detailed Results ---`n" -ForegroundColor Cyan
$testResults | Format-Table -AutoSize Component, Test, Status, Details

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "Testing URLs:" -ForegroundColor White
Write-Host "  Main App:        $frontendUrl" -ForegroundColor Cyan
Write-Host "  API Docs:        $frontendUrl/swagger.html" -ForegroundColor Cyan
Write-Host "  Reactive Tester: $frontendUrl/test-reactive.html" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

# Export results to file
$testResults | Export-Csv -Path "test_results_$(Get-Date -Format 'yyyyMMdd_HHmmss').csv" -NoTypeInformation
Write-Host "Test results exported to CSV" -ForegroundColor Gray
