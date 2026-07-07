param(
    [string]$BaseUrl = "http://127.0.0.1/",
    [string]$Domain = "localhost",
    [string]$Scheme = "https",
    [switch]$UseDirectDns
)

$ErrorActionPreference = "Stop"

$checks = @(
    @{ Host = "static"; Expected = "Deployment Success" },
    @{ Host = "node-app"; Expected = "mypaas-sample-node" },
    @{ Host = "python-app"; Expected = "mypaas-sample-fastapi" },
    @{ Host = "go-app"; Expected = "mypaas-sample-go" },
    @{ Host = "crud"; Expected = "Dummy Todos" }
)

$failures = 0

foreach ($check in $checks) {
    $hostName = "$($check.Host).$Domain"
    $uri = $BaseUrl
    $headers = @{ Host = $hostName }
    if ($UseDirectDns) {
        $uri = "${Scheme}://$hostName/"
        $headers = @{}
    }

    try {
        $response = Invoke-WebRequest -Uri $uri -Headers $headers -UseBasicParsing -TimeoutSec 10
    }
    catch {
        Write-Host "FAIL $hostName request failed: $($_.Exception.Message)" -ForegroundColor Red
        $failures++
        continue
    }

    $body = [string]$response.Content
    if ($response.StatusCode -lt 200 -or $response.StatusCode -ge 300) {
        Write-Host "FAIL $hostName returned HTTP $($response.StatusCode)" -ForegroundColor Red
        $failures++
        continue
    }

    if ($body -notlike "*$($check.Expected)*") {
        $preview = ($body -replace "\s+", " ")
        if ($preview.Length -gt 120) {
            $preview = $preview.Substring(0, 120)
        }
        Write-Host "FAIL $hostName missing expected content '$($check.Expected)'. Body: $preview" -ForegroundColor Red
        $failures++
        continue
    }

    Write-Host "OK   $hostName contains '$($check.Expected)'"
}

if ($failures -gt 0) {
    Write-Host "$failures dogfood smoke check(s) failed." -ForegroundColor Red
    exit 1
}

Write-Host "All dogfood smoke checks passed."
