# E2EE Implementation Handoff

**Date:** 2026-03-16
**Status:** Research Complete, Implementation Deferred
**Priority:** Roadmap Item (Not a Blocker for Production)

---

## Summary

End-to-end encryption (E2EE) for Matrix messages was researched and partially implemented. The feature is **not required for initial production deployment** and has been documented as a roadmap item.

---

## Current State

### What's Done
- ✅ Research completed - mautrix-go identified as best option
- ✅ Dependency added: `maunium.net/go/mautrix v0.26.4`
- ✅ E2EE wrapper created: `bridge/pkg/crypto/e2ee.go`
- ✅ Documentation updated: `bridge/doc/LESSONS_LEARNED.md`

### What's Not Done
- ❌ Full E2EE implementation (requires 4-8 weeks)
- ❌ Integration with MatrixAdapter sync flow
- ❌ Device key management
- ❌ Cross-signing support
- ❌ Key backup/restore

### Current Behavior
- Messages are sent in **plaintext over HTTPS**
- Device ID `armorclaw-bridge` is hardcoded
- No E2EE session management
- Works correctly for production use

---

## Research Findings

### Recommended Library: mautrix-go

| Attribute | Value |
|-----------|-------|
| **Package** | `maunium.net/go/mautrix/crypto` |
| **Stars** | 601 |
| **License** | MPL-2.0 |
| **Backend** | Pure Go (goolm) or CGO (libolm) |
| **Last Update** | 2026-03-16 (active) |
| **Used By** | gomuks (1609 stars), mautrix-whatsapp, go-neb |

### Implementation Options

1. **Pure Go (Recommended)**
   ```bash
   go build -tags goolm ./...
   ```
   - No CGO required
   - Easier cross-compilation
   - Slightly slower than CGO

2. **CGO + libolm**
   ```bash
   go build ./...
   ```
   - Requires libolm-dev installed
   - Better performance
   - Cross-compilation challenges

### Estimated Effort

| Phase | Description | Effort |
|-------|-------------|--------|
| Phase 1 | Basic encrypt/decrypt | 2-3 weeks |
| Phase 2 | Device verification | 1-2 weeks |
| Phase 3 | Cross-signing, key backup | 1-2 weeks |
| **Total** | Full E2EE | **4-7 weeks** |

---

## Security Considerations

### Known Issues with Olm/Vodozemac
- 2026 blog post identified potential vulnerabilities
- Matrix.org claims issues are not practically exploitable
- Production clients (Element, gomuks) continue to use these libraries

### Recommendation
- Proceed with mautrix-go for E2EE implementation
- Keep dependencies updated
- Monitor security advisories

---

## Implementation Guide

### Phase 1: Basic E2EE (2-3 weeks)

1. **Fix go.mod dependencies**
   ```bash
   cd bridge
   go get github.com/petermattis/goid
   go mod tidy
   ```

2. **Complete E2EE wrapper** (`bridge/pkg/crypto/e2ee.go`)
   - Implement full `crypto.Store` interface
   - Add persistent storage for OlmAccount
   - Handle session management

3. **Integrate with MatrixAdapter**
   - Add E2EEMachine to MatrixAdapter struct
   - Modify `SendMessage` to encrypt for encrypted rooms
   - Modify `processEvents` to decrypt m.room.encrypted events

4. **Add configuration**
   ```toml
   [matrix.e2ee]
   enabled = true
   device_id = "armorclaw-bridge"
   ```

### Phase 2: Device Verification (1-2 weeks)

1. Implement device list tracking
2. Add SAS verification support
3. Create verification UI/command

### Phase 3: Advanced Features (1-2 weeks)

1. Cross-signing setup
2. Key backup/restore
3. Key export/import

---

## Files Modified

| File | Status | Notes |
|------|--------|-------|
| `bridge/go.mod` | Modified | Added mautrix-go dependency |
| `bridge/go.sum` | Modified | Dependency checksums |
| `bridge/pkg/crypto/e2ee.go` | Created | E2EE wrapper (incomplete) |
| `bridge/doc/LESSONS_LEARNED.md` | Updated | Documented E2EE status |

---

## Testing Checklist

When implementing E2EE, test:

- [ ] Send encrypted message to encrypted room
- [ ] Receive encrypted message from encrypted room
- [ ] Send plaintext message to unencrypted room
- [ ] Key rotation after device re-registration
- [ ] Multi-device key sharing
- [ ] Cross-signing verification
- [ ] Key backup and restore
- [ ] Message decryption after restart

---

## References

- [mautrix-go Documentation](https://pkg.go.dev/maunium.net/go/mautrix)
- [mautrix-go GitHub](https://github.com/maunium/go)
- [Matrix E2EE Guide](https://matrix.org/docs/guides/end-to-end-encryption/)
- [gomuks Source](https://github.com/gomuks/gomuks) (reference implementation)
- [Security Analysis](https://soatok.blog/2026/02/17/cryptographic-issues-in-matrixs-rust-library-vodozemac/)

---

## Decision Points

Before implementing E2EE, decide:

1. **Pure Go vs CGO?**
   - Pure Go: easier builds, no libolm dependency
   - CGO: better performance, requires libolm-dev

2. **Trust Model?**
   - Unset: accept all devices (easiest)
   - Cross-signed: require cross-signing (most secure)
   - Verified: require manual verification

3. **Key Storage?**
   - SQLCipher (recommended - already in use)
   - Memory-only (testing only)
   - File-based (simpler but less secure)

---

## Conclusion

E2EE is a **roadmap item**, not a production blocker. The current plaintext HTTPS implementation is sufficient for initial deployment. Full E2EE can be implemented in 4-7 weeks when needed.

**Recommendation:** Deploy current version, add E2EE in v2.0 release.
