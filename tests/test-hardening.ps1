# ArmorClaw Hardening Tests (PowerShell)
# Run with: .\tests\test-hardening.ps1

Write-Host "🧪 Running Container Hardening Tests..."
Write-Host ""

$ErrorActionPreference = "Continue"
$passCount = 0
$failCount = 0

# Helper to log test results
function Test-Result {
    param([string]$Name, [bool]$Passed)

    if ($Passed) {
        Write-Host "✅ $Name" -ForegroundColor Green
        $script:passCount++
    } else {
        Write-Host "❌ FAIL: $Name" -ForegroundColor Red
        $script:failCount++
    }
}

# Helper to run docker with entrypoint override (using empty string)
function Run-DockerTest {
    param([string]$Command)

    # Use --entrypoint with an empty argument
    $result = docker run --rm --entrypoint "" armorclaw/agent:v1 $Command 2>&1
    return $result
}

# ============================================================================
# Test 1: UID check
# ============================================================================
Write-Host "Test 1: User ID Check"
Write-Host "-------------------"

$uidOutput = Run-DockerTest "id"
if ($uidOutput -match "uid=10001\(claw\)") {
    Test-Result "Running as UID 10001(claw)" $true
} else {
    Test-Result "Running as UID 10001(claw)" $false
    Write-Host "Output: $uidOutput"
}

# ============================================================================
# Test 2: Shell removed
# ============================================================================
Write-Host ""
Write-Host "Test 2: Shell Removal"
Write-Host "-------------------"

$shTest = docker run --rm --entrypoint "" armorclaw/agent:v1 sh -c "echo test" 2>&1
if ($shTest -match "executable file not found" -or $shTest -match "not found") {
    Test-Result "Shell (sh) denied - not available" $true
} else {
    Test-Result "Shell (sh) denied - not available" $false
}

$bashTest = docker run --rm --entrypoint "" armorclaw/agent:v1 bash -c "echo test" 2>&1
if ($bashTest -match "executable file not found" -or $bashTest -match "not found") {
    Test-Result "Shell (bash) denied - not available" $true
} else {
    Test-Result "Shell (bash) denied - not available" $false
}

# ============================================================================
# Test 3: Network tools removed
# ============================================================================
Write-Host ""
Write-Host "Test 3: Network Tools Removal"
Write-Host "----------------------------"

$curlTest = docker run --rm --entrypoint "" armorclaw/agent:v1 curl --version 2>&1
if ($curlTest -match "not found") {
    Test-Result "curl denied" $true
} else {
    Test-Result "curl denied" $false
}

$wgetTest = docker run --rm --entrypoint "" armorclaw/agent:v1 wget --version 2>&1
if ($wgetTest -match "not found") {
    Test-Result "wget denied" $true
} else {
    Test-Result "wget denied" $false
}

$ncTest = docker run --rm --entrypoint "" armorclaw/agent:v1 nc -h 2>&1
if ($ncTest -match "not found") {
    Test-Result "nc/netcat denied" $true
} else {
    Test-Result "nc/netcat denied" $false
}

# ============================================================================
# Test 4: Python available (required)
# ============================================================================
Write-Host ""
Write-Host "Test 4: Required Runtime Availability"
Write-Host "------------------------------------"

$pythonTest = docker run --rm --entrypoint "" armorclaw/agent:v1 python --version 2>&1
if ($pythonTest -match "Python 3\.\d+\.\d+") {
    $version = [regex]::Match($pythonTest, "Python (\d+\.\d+\.\d+)").Groups[1].Value
    Test-Result "Python available ($version)" $true
} else {
    Test-Result "Python available" $false
}

# ============================================================================
# Test 5: Destructive tools removed
# ============================================================================
Write-Host ""
Write-Host "Test 5: Destructive Tools Removal"
Write-Host "-------------------------------"

$destructiveTools = @("rm", "mv", "find")
foreach ($tool in $destructiveTools) {
    $toolTest = docker run --rm --entrypoint "" armorclaw/agent:v1 $tool --help 2>&1
    if ($toolTest -match "not found") {
        Test-Result "$tool denied" $true
    } else {
        Test-Result "$tool denied" $false
    }
}

# ============================================================================
# Test 6: Process tools removed
# ============================================================================
Write-Host ""
Write-Host "Test 6: Process Tools Removal"
Write-Host "--------------------------"

$processTools = @("ps", "top", "lsof")
foreach ($tool in $processTools) {
    $toolTest = docker run --rm --entrypoint "" armorclaw/agent:v1 $tool 2>&1
    if ($toolTest -match "not found") {
        Test-Result "$tool denied" $true
    } else {
        Test-Result "$tool denied" $false
    }
}

# ============================================================================
# Test 7: Package manager removed
# ============================================================================
Write-Host ""
Write-Host "Test 7: Package Manager Removal"
Write-Host "-----------------------------"

$aptTest = docker run --rm --entrypoint "" armorclaw/agent:v1 apt-get --version 2>&1
if ($aptTest -match "not found") {
    Test-Result "apt/apt-get denied" $true
} else {
    Test-Result "apt/apt-get denied" $false
}

# ============================================================================
# Summary
# ============================================================================
Write-Host ""
Write-Host "================================"
Write-Host "Hardening Test Summary"
Write-Host "================================"
Write-Host ""
Write-Host "Total Tests: $($passCount + $failCount)"
Write-Host "Passed:      $passCount"
Write-Host "Failed:      $failCount"
Write-Host ""

if ($failCount -eq 0) {
    Write-Host "✅ ALL HARDENING TESTS PASSED" -ForegroundColor Green
    Write-Host ""
    Write-Host "Container hardening verified:"
    Write-Host "  ✅ Non-root user (UID 10001)"
    Write-Host "  ✅ No shell access"
    Write-Host "  ✅ No network tools"
    Write-Host "  ✅ No destructive tools"
    Write-Host "  ✅ No process inspection tools"
    Write-Host "  ✅ No package manager"
    Write-Host "  ✅ Python runtime available"
    exit 0
} else {
    Write-Host "❌ $failCount HARDENING TEST(S) FAILED" -ForegroundColor Red
    exit 1
}
