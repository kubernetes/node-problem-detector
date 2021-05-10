# This plugin checks to see if windows defender detects any threats to the node.

$windowsDefenderThreats = (Get-MpThreat | Where-Object {$_.IsActive -or $_.DidThreatExecute})

if ($windowsDefenderThreats.length -ne 0) {
    Write-Host $windowsDefenderThreats
    exit 1
} else {
    exit 0
}
