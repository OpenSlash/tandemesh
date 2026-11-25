# Load the shared local toolchain into the current PowerShell session.
# Tools live outside any one project so Android/Harmony builds can be reused.
$ErrorActionPreference = 'Stop'

$root = if ($env:TANDEMESH_TOOLS) { $env:TANDEMESH_TOOLS } else { 'D:\project\_tools' }

$env:JAVA_HOME = Join-Path $root 'jdk-17'
$env:ANDROID_HOME = Join-Path $root 'android-sdk'
$env:ANDROID_SDK_ROOT = $env:ANDROID_HOME
$env:GRADLE_HOME = Join-Path $root 'gradle-9.0.0'

$env:PATH = @(
    (Join-Path $env:JAVA_HOME 'bin'),
    (Join-Path $env:ANDROID_HOME 'platform-tools'),
    (Join-Path $env:ANDROID_HOME 'cmdline-tools\latest\bin'),
    (Join-Path $env:ANDROID_HOME 'emulator'),
    (Join-Path $env:GRADLE_HOME 'bin'),
    $env:PATH
) -join ';'

Write-Host "JAVA_HOME=$env:JAVA_HOME"
Write-Host "ANDROID_HOME=$env:ANDROID_HOME"
Write-Host "GRADLE_HOME=$env:GRADLE_HOME"
