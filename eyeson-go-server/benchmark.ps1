$baseUrl = "http://localhost:5000"
$authUrl = "$baseUrl/api/v1/auth/login"
$simsUrl = "$baseUrl/api/v1/sims"

Write-Host "DEBUG: SimsUrl is '$simsUrl'"

Write-Host "Authenticating..."
$body = @{ username = "admin"; password = "admin" } | ConvertTo-Json
try {
    $loginRes = Invoke-RestMethod -Uri $authUrl -Method Post -Body $body -ContentType "application/json"
    $token = $loginRes.token
} catch {
    Write-Error "Login failed: $_"
    exit
}

$headers = @{ Authorization = "Bearer $token" }

$total = 1000
$percentages = @(10, 30, 50, 75, 100)
$results = @()

Write-Host "Starting Benchmark..."
foreach ($p in $percentages) {
    $limit = [int]($total * ($p / 100))
    $url = $simsUrl + "?start=0&limit=" + $limit
    
    Write-Host "Testing $p% (Limit: $limit) - URL: $url"

    try {
        $time = Measure-Command {
            $rawResponse = Invoke-WebRequest -Uri $url -Method Get -Headers $headers -UseBasicParsing
        }
        
        $content = $rawResponse.Content | ConvertFrom-Json
        $sizeBytes = $rawResponse.Content.Length
        $received = 0
        if ($content.data) {
             $received = $content.data.length
        }
        
        $results += [PSCustomObject]@{
            Percentage = "$p%"
            Requested = $limit
            Received = $received
            SizeKB = [math]::Round($sizeBytes / 1KB, 2)
            TimeMS = [math]::Round($time.TotalMilliseconds, 2)
        }
    } catch {
        Write-Error "Request failed: $_"
    }
}

$results | Format-Table -AutoSize