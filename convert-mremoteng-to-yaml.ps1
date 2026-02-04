# Convert mRemoteNG XML to MremoteGO YAML format
# 
# This script helps migrate your mRemoteNG connections to MremoteGO's YAML format.
# It preserves folder structure and converts connections to the git-friendly format.
#
# Usage:
#   .\convert-mremoteng-to-yaml.ps1 -SourceXml "confCons.xml" -OutputYaml "connections.yaml"
#
# After conversion:
#   1. Review the generated YAML file
#   2. Replace passwords with 1Password references: op://Vault/Item/password
#   3. Or use encrypted passwords with: mremotego encrypt-passwords
#   4. Configure 1Password settings in config.yaml if using 1Password integration
#
param(
    [Parameter(Mandatory=$false)]
    [string]$SourceXml = "confCons.xml",
    
    [Parameter(Mandatory=$false)]
    [string]$OutputYaml = "connections.yaml",
    
    [Parameter(Mandatory=$false)]
    [switch]$Include1PasswordTemplate = $false
)

function Convert-ProtocolToMremoteGO {
    param([string]$protocol)
    
    switch ($protocol) {
        "SSH2" { return "ssh" }
        "RDP" { return "rdp" }
        "VNC" { return "vnc" }
        "HTTP" { return "http" }
        "HTTPS" { return "https" }
        "Telnet" { return "telnet" }
        default { return "unknown" }
    }
}

function Convert-ConnectionNode {
    param($node, $indent = 0)
    
    $spaces = "  " * $indent
    $yaml = ""
    
    # Clean up name - remove trailing spaces
    $cleanName = $node.Name.Trim()
    $yaml += "$spaces- name: `"$cleanName`"`n"
    
    if ($node.Type -eq "Container") {
        $yaml += "$spaces  type: folder`n"
        if ($node.Descr) {
            $cleanDescr = $node.Descr -replace '[\r\n]+', ' '
            $cleanDescr = $cleanDescr.Trim()
            $yaml += "$spaces  description: `"$cleanDescr`"`n"
        }
        
        # Process children
        $children = $node.SelectNodes("Node")
        if ($children.Count -gt 0) {
            $yaml += "$spaces  children:`n"
            foreach ($child in $children) {
                $yaml += Convert-ConnectionNode -node $child -indent ($indent + 2)
            }
        }
    }
    elseif ($node.Type -eq "Connection") {
        $yaml += "$spaces  type: connection`n"
        
        # Protocol
        $protocol = Convert-ProtocolToMremoteGO -protocol $node.Protocol
        $yaml += "$spaces  protocol: $protocol`n"
        
        # Host
        if ($node.Hostname) {
            $yaml += "$spaces  host: `"$($node.Hostname)`"`n"
        }
        
        # Port
        if ($node.Port) {
            $yaml += "$spaces  port: $($node.Port)`n"
        }
        
        # Username
        if ($node.Username) {
            $yaml += "$spaces  username: `"$($node.Username)`"`n"
        }
        
        # Password - placeholder for 1Password or encrypted
        if ($node.Password -and $node.Password -ne "") {
            if ($Include1PasswordTemplate) {
                # Add 1Password template
                $yaml += "$spaces  password: `"op://Private/$cleanName/password`"  # TODO: Update vault and item names`n"
            } else {
                # Add placeholder for manual entry
                $yaml += "$spaces  password: `"`"  # TODO: Set password or use 1Password reference`n"
            }
        }
        
        # Domain
        if ($node.Domain) {
            $yaml += "$spaces  domain: `"$($node.Domain)`"`n"
        }
        
        # Description
        if ($node.Descr) {
            # Clean up description - remove newlines
            $cleanDescr = $node.Descr -replace '[\r\n]+', ' '
            $cleanDescr = $cleanDescr.Trim()
            $yaml += "$spaces  description: `"$cleanDescr`"`n"
        }
        
        # Tags (optional - you can add custom logic here)
        # $yaml += "$spaces  tags:`n"
        # $yaml += "$spaces    - imported`n"
        
        # RDP specific settings
        if ($protocol -eq "rdp") {
            if ($node.UseCredSsp -eq "true") {
                $yaml += "$spaces  use_credssp: true`n"
            }
            if ($node.Resolution) {
                $yaml += "$spaces  resolution: `"$($node.Resolution)`"`n"
            }
            if ($node.Colors) {
                $yaml += "$spaces  colors: `"$($node.Colors)`"`n"
            }
        }
        
        # SSH specific settings
        if ($protocol -eq "ssh") {
            # Add any SSH-specific settings here if needed
        }
    }
    
    return $yaml
}

# Load the XML
Write-Host "Loading mRemoteNG XML from: $SourceXml" -ForegroundColor Cyan

if (-not (Test-Path $SourceXml)) {
    Write-Host "ERROR: Source file not found: $SourceXml" -ForegroundColor Red
    Write-Host ""
    Write-Host "Usage: .\convert-mremoteng-to-yaml.ps1 -SourceXml `"your-mremoteng-config.xml`"" -ForegroundColor Yellow
    exit 1
}

[xml]$xml = Get-Content $SourceXml

# Get root node
$rootNode = $xml.SelectSingleNode("//mrng:Connections | //Connections", $null)
if (-not $rootNode) {
    Write-Host "ERROR: Could not find Connections root node in XML" -ForegroundColor Red
    exit 1
}

# Get all top-level nodes (folders and connections)
$topLevelNodes = $rootNode.SelectNodes("Node")

Write-Host "Found $($topLevelNodes.Count) top-level items (folders and connections)" -ForegroundColor Green
Write-Host ""

# Start building YAML
$yaml = "# MremoteGO Configuration File`n"
$yaml += "# Generated from mRemoteNG XML: $SourceXml`n"
$yaml += "# Date: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')`n"
$yaml += "#`n"
$yaml += "# Next steps:`n"
$yaml += "#   1. Review this file and update passwords`n"
$yaml += "#   2. Use 1Password references: op://VaultName/ItemName/password`n"
$yaml += "#   3. Or encrypt passwords: mremotego encrypt-passwords`n"
$yaml += "#   4. Configure settings in config.yaml`n"
$yaml += "`n"
$yaml += "version: `"1.0`"`n"
$yaml += "connections:`n"

# Convert each top-level node
foreach ($node in $topLevelNodes) {
    $yaml += Convert-ConnectionNode -node $node -indent 1
}

# Save to file
$yaml | Out-File -FilePath $OutputYaml -Encoding UTF8 -NoNewline

Write-Host "✓ Successfully converted to YAML: $OutputYaml" -ForegroundColor Green
Write-Host ""
Write-Host "IMPORTANT NOTES:" -ForegroundColor Yellow
Write-Host "  • Passwords are NOT migrated from mRemoteNG (encrypted format incompatible)" -ForegroundColor Yellow
Write-Host "  • You need to:" -ForegroundColor Yellow
Write-Host "    1. Manually set passwords in YAML" -ForegroundColor White
Write-Host "    2. Use 1Password integration (recommended)" -ForegroundColor White
Write-Host "       Example: password: `"op://DevOps/ServerName/password`"" -ForegroundColor Gray
Write-Host "    3. Or use MremoteGO's encryption" -ForegroundColor White
Write-Host ""
Write-Host "1Password Setup:" -ForegroundColor Cyan
Write-Host "  • See docs/1PASSWORD-SETUP.md for configuration guide" -ForegroundColor White
Write-Host "  • Configure account in config.yaml settings.onePasswordAccount" -ForegroundColor White
Write-Host "  • Use vault name mappings for easier references" -ForegroundColor White
Write-Host ""
Write-Host "Next Command:" -ForegroundColor Cyan
Write-Host "  mremotego.exe  # Launch GUI and review connections" -ForegroundColor White
