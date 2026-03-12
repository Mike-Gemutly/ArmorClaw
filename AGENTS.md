# ArmorClaw OMO Guardrails

## Brownfield rule
This is a brownfield system. Preserve the existing architecture unless explicitly asked otherwise.

## Source of truth
Read review.md before planning or modifying code.

## Existing architecture
- Go Bridge
- Matrix Conduit
- SQLCipher keystore
- Agent Studio
- browser-service (TypeScript/Playwright)
- ArmorChat Android client

## Security constraints
- Do not remove SQLCipher
- Do not bypass Matrix as control plane
- Do not weaken approval flow for payments or critical PII
- Do not introduce direct production secret access
- Prefer minimal patches over rewrites

## First priority areas
1. Deployment stability
2. Matrix event/control-plane consistency
3. Browser-service API and queue consistency
4. Android approval/status UX
