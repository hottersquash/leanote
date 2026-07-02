# Install Go tools and download module dependencies for local development.
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")

Write-Host "Go version:" (go version)

Write-Host "Configuring Go module proxy..."
go env -w GOPROXY=https://goproxy.cn,direct

Write-Host "Downloading module dependencies..."
go mod download

Write-Host "Installing revel CLI..."
go install github.com/revel/cmd/revel@v1.0.3

Write-Host "Installing delve debugger..."
go install github.com/go-delve/delve/cmd/dlv@latest

$goBin = Join-Path (go env GOPATH) "bin"
Write-Host ""
Write-Host "Setup complete."
Write-Host "Ensure Go bin is on PATH: $goBin"
Write-Host "Copy scripts/dev.env.example to scripts/dev.env if you need a custom MongoDB URL."
Write-Host "Start the app with: .\scripts\run-dev.ps1"
