$current = Get-Location
$Env:PATH += ";$current"
[System.Environment]::SetEnvironmentVariable("CALC_BIN", "$current", [System.EnvironmentVariableTarget]::User)
[System.Environment]::SetEnvironmentVariable("PATH", "$Env:PATH", [System.EnvironmentVariableTarget]::User)
"success installed"
Write-Host -NoNewLine 'Press any key to continue...';
$null = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown');