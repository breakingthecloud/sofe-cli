#!/bin/bash
# SOFE CLI — Serve + Auto-Serve Manual Tests
#
# Run these tests MANUALLY in a terminal (not from automated scripts)
# because serve/stop involve process management that can interfere
# with background processes.
#
# Prerequisites:
#   - sofe-server installed: pip install sofe-server
#   - Go CLI built: go build -o sofe .
#   - AWS profile configured
#   - Nothing running on port 8080

echo "🧪 SOFE Serve Manual Tests"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Run each section manually and verify output:"
echo ""

echo "--- TEST A: sofe serve ---"
echo "$ ./sofe serve"
echo "Expected: ✅ SOFE Server running on :8080 (PID xxxxx)"
echo ""

echo "--- TEST B: sofe health ---"
echo "$ ./sofe health"
echo "Expected: ✅ SOFE Server 0.1.0 (ok)"
echo ""

echo "--- TEST C: sofe evaluate (with server running) ---"
echo "$ ./sofe evaluate -p ../sofe/policies --profile your-profile"
echo "Expected: 📋 ... ☁️ 6 resources | ⚡ 10 findings"
echo ""

echo "--- TEST D: sofe serve stop ---"
echo "$ ./sofe serve stop"
echo "Expected: ✅ Server stopped"
echo "Verify:   curl http://localhost:8080/health → connection refused"
echo ""

echo "--- TEST E: --auto-serve (no server running) ---"
echo "$ ./sofe evaluate -p ../sofe/policies --profile your-profile --auto-serve"
echo "Expected:"
echo "  🔄 Server not running, starting on :8080..."
echo "  ✅ Server started automatically"
echo "  📋 ... ☁️ ... ⚡ 10 findings"
echo "  ⏹ Stopping auto-started server..."
echo "Verify:   curl http://localhost:8080/health → connection refused (after ~2s)"
echo ""

echo "--- TEST F: --auto-serve (server already running) ---"
echo "$ ./sofe serve"
echo "$ ./sofe evaluate -p ../sofe/policies --profile your-profile --auto-serve"
echo "Expected: evaluates normally, does NOT stop server (it didn't start it)"
echo "$ ./sofe serve stop"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "All 6 tests should match expected output."
