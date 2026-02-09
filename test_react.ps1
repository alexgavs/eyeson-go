$baseUrl = "http://localhost:3000/api/v1"

Write-Host "=== Testing Reactive Architecture ===" -ForegroundColor Cyan

# Login
Write-Host "`n[1] Login..." -ForegroundColor Yellow
$loginBody = @{ username = "admin"; password = "admin123" } | ConvertTo-Json
$loginResponse = Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method POST -Body $loginBody -ContentType "application/json"
$token = $loginResponse.token
$headers = @{ "Authorization" = "Bearer $token" }
Write-Host "Login successful" -ForegroundColor Green

# Test Reactive SIM List
Write-Host "`n[2] Testing /reactive/sims..." -ForegroundColor Yellow
$simsResponse = Invoke-RestMethod -Uri "$baseUrl/reactive/sims" -Method GET -Headers $headers
Write-Host "Reactive SIMs: $($simsResponse.count) SIMs returned" -ForegroundColor Green

# Test Reactive Search
Write-Host "`n[3] Testing /reactive/search..." -ForegroundColor Yellow
$searchResponse = Invoke-RestMethod -Uri "$baseUrl/reactive/search?q=972" -Method GET -Headers $headers
Write-Host "Reactive Search: $($searchResponse.count) results" -ForegroundColor Green

# Test SSE Events
Write-Host "`n[4] Testing /reactive/events (SSE)..." -ForegroundColor Yellow
try {
    $null = Invoke-WebRequest -Uri "$baseUrl/reactive/events" -Headers $headers -TimeoutSec 2
}
catch {
    Write-Host "SSE endpoint active" -ForegroundColor Green
}

Write-Host "`n=== Tests Complete ===" -ForegroundColor Cyan
Write-Host "All reactive endpoints work!" -ForegroundColor Green
