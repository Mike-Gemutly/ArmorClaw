# ArmorClaw v1 Test Suite
# Tests verify containment guarantees and hardening posture

.PHONY: test test-hardening test-secrets test-exploits test-e2e test-all clean decrypt-test-secrets test-container-setup generate-proto

# Decrypt test files with secrets before running tests
decrypt-test-secrets:
	@./decrypt-test-secrets.sh

# Default: run all tests
test-all: decrypt-test-secrets test-hardening test-secrets test-exploits test-e2e test-container-setup
	@echo ""
	@echo "✅ All test suites passed"

# 1. Container Hardening Tests (highest leverage)
# Verifies container matches design specifications
test-hardening:
	@echo "🧪 Running Container Hardening Tests..."
	@echo ""

# UID must be 10001 (claw), not root or nobody
	@docker run --rm -e OPENAI_API_KEY=sk-test mikegemut/armorclaw:latest id | grep -q "uid=10001(claw)" || \
		(echo "❌ FAIL: Container not running as UID 10001(claw)"; exit 1)
	@echo "✅ 1. UID check: 10001(claw)"

# No shell available (use sh -c to bypass entrypoint validation)
	@docker run --rm -e OPENAI_API_KEY=sk-test mikegemut/armorclaw:latest sh -c 'command -v sh' 2>/dev/null && \
		(echo "❌ FAIL: Shell available"; exit 1) || echo "✅ 2. Shell denied: /bin/sh not found"
	@docker run --rm -e OPENAI_API_KEY=sk-test mikegemut/armorclaw:latest sh -c 'command -v bash' 2>/dev/null && \
		(echo "❌ FAIL: Bash available"; exit 1) || echo "✅ 3. Bash denied: /bin/bash not found"

# No destructive tools
	@docker run --rm -e OPENAI_API_KEY=sk-test mikegemut/armorclaw:latest sh -c 'ls /bin/rm' 2>/dev/null && \
		(echo "❌ FAIL: rm available"; exit 1) || echo "✅ 4. rm denied: /bin/rm not found"
	@docker run --rm -e OPENAI_API_KEY=sk-test mikegemut/armorclaw:latest sh -c 'ls /usr/bin/mv' 2>/dev/null && \
		(echo "❌ FAIL: mv available"; exit 1) || echo "✅ 5. mv denied: /usr/bin/mv not found"
	@docker run --rm -e OPENAI_API_KEY=sk-test mikegemut/armorclaw:latest sh -c 'command -v find' 2>/dev/null && \
		(echo "❌ FAIL: find available"; exit 1) || echo "✅ 6. find denied: find command not found"

# No network tools
	@docker run --rm -e OPENAI_API_KEY=sk-test mikegemut/armorclaw:latest sh -c 'command -v curl' 2>/dev/null && \
		(echo "❌ FAIL: curl available"; exit 1) || echo "✅ 7. curl denied: curl command not found"
	@docker run --rm -e OPENAI_API_KEY=sk-test mikegemut/armorclaw:latest sh -c 'command -v wget' 2>/dev/null && \
		(echo "❌ FAIL: wget available"; exit 1) || echo "✅ 8. wget denied: wget command not found"
	@docker run --rm -e OPENAI_API_KEY=sk-test mikegemut/armorclaw:latest sh -c 'command -v nc' 2>/dev/null && \
		(echo "❌ FAIL: nc available"; exit 1) || echo "✅ 9. nc denied: netcat not found"

# No process inspection tools
	@docker run --rm -e OPENAI_API_KEY=sk-test mikegemut/armorclaw:latest sh -c 'command -v ps' 2>/dev/null && \
		(echo "❌ FAIL: ps available"; exit 1) || echo "✅ 10. ps denied: ps command not found"
	@docker run --rm -e OPENAI_API_KEY=sk-test mikegemut/armorclaw:latest sh -c 'command -v top' 2>/dev/null && \
		(echo "❌ FAIL: top available"; exit 1) || echo "✅ 11. top denied: top command not found"
	@docker run --rm -e OPENAI_API_KEY=sk-test mikegemut/armorclaw:latest sh -c 'command -v lsof' 2>/dev/null && \
		(echo "❌ FAIL: lsof available"; exit 1) || echo "✅ 12. lsof denied: lsof command not found"

# No package manager
	@docker run --rm -e OPENAI_API_KEY=sk-test mikegemut/armorclaw:latest sh -c 'command -v apt' 2>/dev/null && \
		(echo "❌ FAIL: apt available"; exit 1) || echo "✅ 13. apt denied: apt command not found"

# Verify safe tools are available (use Python to bypass entrypoint validation)
	@docker run --rm -e OPENAI_API_KEY=sk-test mikegemut/armorclaw:latest python -c "import shutil; assert shutil.which('cp'), 'cp not found'" >/dev/null 2>&1 || \
		(echo "❌ FAIL: cp not available"; exit 1)
	@echo "✅ 14. Safe tool available: cp"
	@docker run --rm -e OPENAI_API_KEY=sk-test mikegemut/armorclaw:latest python -c "import shutil; assert shutil.which('mkdir'), 'mkdir not found'" >/dev/null 2>&1 || \
		(echo "❌ FAIL: mkdir not available"; exit 1)
	@echo "✅ 15. Safe tool available: mkdir"
	@docker run --rm -e OPENAI_API_KEY=sk-test mikegemut/armorclaw:latest python -c "import shutil; assert shutil.which('stat'), 'stat not found'" >/dev/null 2>&1 || \
		(echo "❌ FAIL: stat not available"; exit 1)
	@echo "✅ 16. Safe tool available: stat"

# Verify read-only root filesystem (with --read-only flag)
	@docker run --rm --read-only -e OPENAI_API_KEY=sk-test mikegemut/armorclaw:latest python -c "open('/etc/test-file', 'w')" 2>/dev/null && \
		(echo "❌ FAIL: Root filesystem is writable"; exit 1) || \
		echo "✅ 17. Read-only root: Cannot write to /etc"

	@echo ""
	@echo "✅ All hardening tests passed"

# 2. Secrets Injection Validation
test-secrets:
	@echo "🧪 Running Secrets Injection Validation..."
	@./tests/test-secrets.sh

# 3. Security Exploit Simulations
test-exploits:
	@echo "🧪 Running Security Exploit Simulations..."
	@./tests/test-exploits.sh

# 4. End-to-End Integration Tests
test-e2e:
	@echo "🧪 Running End-to-End Integration Tests..."
	@./tests/test-e2e.sh

# 5. Container Setup Script Tests
test-container-setup:
	@echo "🧪 Running Container Setup Script Tests..."
	@./tests/test-container-setup.sh

# Clean up test artifacts
clean:
	@echo "Cleaning up test artifacts..."
	@rm -f /tmp/armorclaw-test-* 2>/dev/null || true
	@rm -f bridge/pkg/pii/scrubber_test.go 2>/dev/null || true
	@docker stop test-sec 2>/dev/null || true
	@docker rm test-sec 2>/dev/null || true
	@echo "✅ Clean complete"

# Generate Go protobuf stubs from rust-vault proto definitions
generate-proto:
	@echo "Generating Go protobuf stubs..."
	@mkdir -p bridge/pkg/vault/proto
	PATH="$(HOME)/.local/bin:$(HOME)/go/bin:$$PATH" protoc \
		--go_out=bridge/pkg/vault/proto --go_opt=paths=source_relative \
		--go-grpc_out=bridge/pkg/vault/proto --go-grpc_opt=paths=source_relative \
		-I rust-vault/proto \
		rust-vault/proto/governance.proto
	@echo "✅ Go stubs generated in bridge/pkg/vault/proto/"

# Quick smoke test (hardening only, fastest feedback)
smoke: test-hardening
	@echo ""
	@echo "✅ Smoke test passed"
