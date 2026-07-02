# Run Leanote in dev mode against MongoDB on leanote_server.
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
$env:Path = (Join-Path (go env GOPATH) "bin") + ";" + $env:Path

$devEnv = Join-Path $PSScriptRoot "dev.env"
if (Test-Path $devEnv) {
    Get-Content $devEnv | ForEach-Object {
        $line = $_.Trim()
        if ($line -eq "" -or $line.StartsWith("#")) { return }
        $parts = $line -split "=", 2
        if ($parts.Length -eq 2) {
            $name = $parts[0].Trim()
            $value = $parts[1].Trim()
            Set-Item -Path "Env:$name" -Value $value
            Write-Host "Loaded env: $name"
        }
    }
}

if (-not (Get-Command revel -ErrorAction SilentlyContinue)) {
    Write-Host "revel not found. Run .\scripts\setup-dev.ps1 first."
    exit 1
}

Write-Host "Starting Leanote at http://localhost:9000"
Write-Host "MongoDB host from conf/app.conf: leanote_server (override with scripts/dev.env MONGODB_URL + db.urlEnv)"
revel run -a .
