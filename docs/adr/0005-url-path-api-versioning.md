# AD-5: API versioning is URL path-based

**Date**: 2026-08-19
**Status**: accepted

## Context

Every `/v1` endpoint needs one consistent versioning scheme across modules built independently — mixed path-versioned and header-versioned endpoints would be a worse outcome than either scheme alone. CAP-2's own success criterion is that Pokedex functionality be "fully usable and demoable via API calls alone," with no UI in v1.

## Decision

Every HTTP endpoint is versioned in the path (`/v1/...`). No content-negotiation or header-based versioning. A path is something a person can read off a terminal or curl command directly, which matters more here than REST purism when there's no UI to hide the URL behind.

## Consequences

### Positive
- Version is visible in every log line, every curl command, every browser address bar — no header inspection needed to know what's being called.

### Negative
- None significant at this scale — the usual header-versioning argument (content negotiation, cleaner URLs) doesn't apply when there's no UI client to serve multiple versions to simultaneously.
