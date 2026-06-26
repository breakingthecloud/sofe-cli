#!/bin/bash
# SOFE CLI Integration Tests
# Requires: sofe-server running on localhost:8080
# Usage: ./tests/test_integration.sh

set -e

SOFE="./sofe"
POLICIES="../sofe/policies"
PASS=0
FAIL=0

check() {
  local desc="$1"
  local result=$2
  if [ $result -eq 0 ]; then
    echo "  ✅ $desc"
    PASS=$((PASS + 1))
  else
    echo "  ❌ $desc"
    FAIL=$((FAIL + 1))
  fi
}

echo "🧪 SOFE CLI Integration Tests"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Test 1: Health
echo "1. Health check"
$SOFE health > /dev/null 2>&1
check "sofe health returns success" $?

# Test 2: Policies list
echo "2. Policies list"
OUTPUT=$($SOFE policies -p $POLICIES 2>&1 || true)
echo "$OUTPUT" | grep -q "no-idle-ec2"
check "lists no-idle-ec2 policy" $?
echo "$OUTPUT" | grep -q "require-cost-tags"
check "lists require-cost-tags policy" $?

# Test 3: Evaluate (table output)
echo "3. Evaluate (table)"
OUTPUT=$($SOFE evaluate -p $POLICIES --profile cc-665 2>&1)
echo "$OUTPUT" | grep -q "findings"
check "evaluate returns findings" $?
echo "$OUTPUT" | grep -q "require-cost-tags"
check "finds tag violations" $?

# Test 4: Evaluate (JSON output)
echo "4. Evaluate (json)"
OUTPUT=$($SOFE evaluate -p $POLICIES --profile cc-665 --format json 2>&1)
echo "$OUTPUT" | grep -q "findings_count"
check "json output contains findings_count" $?
echo "$OUTPUT" | grep -q "policy_name"
check "json output contains policy_name" $?

# Test 5: Evaluate (markdown output)
echo "5. Evaluate (markdown)"
OUTPUT=$($SOFE evaluate -p $POLICIES --profile cc-665 --format markdown 2>&1)
echo "$OUTPUT" | grep -q "| Severity |"
check "markdown has table header" $?

# Test 6: --fail-on high (should pass, no high findings)
echo "6. --fail-on high (should pass)"
$SOFE evaluate -p $POLICIES --profile cc-665 --fail-on high > /dev/null 2>&1
check "exit 0 when no high findings" $?

# Test 7: --fail-on medium (should fail, has medium findings)
echo "7. --fail-on medium (should fail)"
set +e
$SOFE evaluate -p $POLICIES --profile cc-665 --fail-on medium > /dev/null 2>&1
RESULT=$?
set -e
[ $RESULT -eq 1 ]
check "exit 1 when medium findings exist" $?

# Test 8: Help
echo "8. Help output"
$SOFE --help | grep -q "evaluate"
check "help shows evaluate command" $?
$SOFE evaluate --help | grep -q "fail-on"
check "evaluate help shows --fail-on flag" $?
$SOFE --help | grep -q "serve"
check "help shows serve command" $?
$SOFE evaluate --help | grep -q "auto-serve"
check "evaluate help shows --auto-serve flag" $?

# Test 9: sofe serve (requires sofe-server installed)
echo "9. Serve commands"
$SOFE serve --help | grep -q "Stop with"
# Note: actual serve/stop tests should be run manually to avoid
# killing processes that may affect the terminal session.
# Manual test:
#   ./sofe serve         → ✅ SOFE Server running on :8080 (PID xxx)
#   ./sofe health        → ✅ SOFE Server 0.1.0 (ok)
#   ./sofe serve stop    → ✅ Server stopped
#   ./sofe evaluate -p ./policies --auto-serve → starts, evaluates, stops
check "serve --help available" $?

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Results: $PASS passed, $FAIL failed"

if [ $FAIL -gt 0 ]; then
  exit 1
fi
echo "✅ All tests passed!"
