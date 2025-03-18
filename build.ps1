# Build script for local testing

$version = git describe --tags --always 2>$null
if (-not $version) {
    $version = "development"
}

$commit = git rev-parse HEAD 2>$null
if (-not $commit) {
    $commit = "unknown"
}

Write-Host "🚀 Starting local build process..."
Write-Host "📦 Version: $version"
Write-Host "🔍 Commit:  $commit"
Write-Host ""

Write-Host "📥 Checking for tailwindcss.exe"
if (-not (Test-Path "tailwindcss.exe")) {
    Write-Host "tailwindcss.exe not found"
    Write-Host "Downloading tailwindcss.exe"
    Invoke-WebRequest -Uri "https://github.com/tailwindlabs/tailwindcss/releases/download/latest/tailwindcss-windows-x64.exe" -OutFile "tailwindcss.exe"
}

Write-Host "🔧 Generating Tailwind CSS file..."
.\tailwindcss.exe -i ./static/css/input.css -o ./static/css/output.css --minify

Write-Host "🔧 Generating templ files..."
templ generate
Write-Host ""

Write-Host "📦 Building application..."
$buildCmd = "go build -v -ldflags `"-X gopos/components.Version=$version -X gopos/components.CommitID=$commit`" -o gopos.exe"
Write-Host "Executing: $buildCmd"
Invoke-Expression $buildCmd

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "✅ Build completed successfully!"
    Write-Host "Run ./gopos.exe to start the application"
} else {
    Write-Host ""
    Write-Host "❌ Build failed!"
    exit 1
} 