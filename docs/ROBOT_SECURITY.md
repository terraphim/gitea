# Gitea Robot API Security Documentation

**Version**: 1.0.0  
**Date**: 2026-04-08  
**Status**: Production Ready  

---

## Overview

The Gitea Robot API provides secure, agent-optimized access to issue data with PageRank-based prioritization. This document describes the security model, configuration, and best practices.

---

## Security Model

### Authentication

All Robot API endpoints require authentication via:
- **Session cookies** (for web UI users)
- **Access tokens** (for API clients and agents)

Unauthenticated requests to private repositories receive a **404 Not Found** response to prevent repository enumeration.

### Authorization

The Robot API enforces repository-level permissions:

| Permission | Requirement | Behavior on Denial |
|------------|-------------|-------------------|
| Read Issues | `unit.TypeIssues` read access | 404 Not Found |
| Private Repo Access | Authenticated user | 404 Not Found |

### Information Disclosure Prevention

All error conditions return **404 Not Found** to prevent:
- Repository enumeration attacks
- Username enumeration
- Repository existence leaks

Examples:
- Invalid owner/repo → 404
- Repository doesn't exist → 404
- Permission denied → 404
- Validation failure → 404

---

## Rate Limiting & Caching

### PageRank Cache

PageRank calculations are cached to prevent CPU exhaustion:

| Setting | Default | Description |
|---------|---------|-------------|
| `PAGERANK_CACHE_TTL` | 300s (5 min) | Cache expiration time |
| Concurrent computation | Prevented | Only one calculation per repo at a time |

**Behavior**:
- Cache hit → Return cached scores immediately
- Cache miss, not computing → Start calculation, cache results
- Cache miss, already computing → Return "in progress" error

### Configuration

```ini
[issue_graph]
ENABLED = true
PAGERANK_CACHE_TTL = 300
AUDIT_LOG = true
STRICT_MODE = false
```

---

## Audit Logging

All Robot API access is logged with:

```
[ROBOT_AUDIT] status=SUCCESS|DENIED user=username(uid=N) repo=owner/repo endpoint=/api/v1/robot/triage ip=x.x.x.x timestamp=RFC3339 [reason=...]
```

**Log Levels**:
- Successful access → INFO
- Denied access → INFO (with reason)
- Errors → ERROR

**Enable/Disable**:
```ini
[issue_graph]
AUDIT_LOG = true   # Enable
AUDIT_LOG = false  # Disable
```

---

## API Endpoints

### GET /api/v1/robot/triage

Returns PageRank-scored issues for prioritization.

**Parameters**:
- `owner` (required): Repository owner
- `repo` (required): Repository name

**Response** (200 OK):
```json
{
  "quick_ref": {
    "total": 150,
    "open": 42
  },
  "recommendations": [
    {
      "id": 123,
      "index": 42,
      "title": "Fix critical bug",
      "pagerank": 0.0852,
      "status": "open"
    }
  ],
  "project_health": {
    "avg_pagerank": 0.0234,
    "max_pagerank": 0.0852
  }
}
```

**Error Responses**:
- 404: Feature disabled, repo not found, permission denied, or validation error
- 500: Internal server error (rare, also returns 404 in strict mode)

### GET /api/v1/robot/ready

Returns issues ready to be worked on (no blocking dependencies).

**Parameters**:
- `owner` (required): Repository owner
- `repo` (required): Repository name

**Response** (200 OK):
```json
{
  "repo_id": 123,
  "repo_name": "myrepo",
  "total_count": 10,
  "ready_issues": [
    {
      "id": 456,
      "index": 23,
      "title": "Add feature X",
      "page_rank": 0.0456,
      "priority": 25,
      "is_blocked": false,
      "blocker_count": 0
    }
  ]
}
```

### GET /api/v1/robot/graph

Returns the full dependency graph for visualization.

**Parameters**:
- `owner` (required): Repository owner
- `repo` (required): Repository name

**Response** (200 OK):
```json
{
  "repo_id": 123,
  "repo_name": "myrepo",
  "node_count": 50,
  "edge_count": 30,
  "nodes": [...],
  "edges": [...]
}
```

---

## Security Best Practices

### 1. Use Access Tokens

For API clients, use personal access tokens with minimal required scopes:

```bash
curl -H "Authorization: token YOUR_TOKEN" \
  "https://gitea.example.com/api/v1/robot/triage?owner=acme&repo=project"
```

### 2. Enable Audit Logging

Always enable audit logging in production:

```ini
[issue_graph]
AUDIT_LOG = true
```

### 3. Configure Cache TTL Appropriately

Balance freshness vs. performance:

| Use Case | Recommended TTL |
|----------|----------------|
| High-activity repos | 60-120 seconds |
| Normal repos | 300 seconds (default) |
| Low-activity repos | 600-1800 seconds |

### 4. Monitor Cache Stats

Check cache effectiveness:

```go
cache := robot.GetPageRankCache()
entries, computing := cache.GetStats()
log.Info("PageRank cache: %d entries, %d computing", entries, computing)
```

### 5. Use Strict Mode for High-Security Environments

```ini
[issue_graph]
STRICT_MODE = true
```

In strict mode, all errors return 404 (no 500 errors ever exposed).

---

## Input Validation

All inputs are validated for:

| Check | Rule |
|-------|------|
| Owner name | Required, max 40 chars |
| Repo name | Required, max 100 chars |
| Path traversal | `..` blocked |
| Invalid chars | `/\<>:|?*\x00` blocked |

**Note**: Validation failures return 404 to prevent information disclosure.

---

## Deployment Checklist

Before deploying to production:

- [ ] Enable issue graph feature
- [ ] Configure appropriate cache TTL
- [ ] Enable audit logging
- [ ] Test with private repository (verify 404 for unauthorized access)
- [ ] Test with public repository (verify 200 for anonymous access)
- [ ] Verify PageRank caching works (second request should be faster)
- [ ] Check audit logs are being written
- [ ] Configure log rotation for audit logs

---

## Troubleshooting

### Issue: 404 for all requests

**Causes**:
1. Feature disabled → Check `ENABLED = true`
2. Permission denied → Verify user has issue read access
3. Repo doesn't exist → Check owner/repo spelling

### Issue: PageRank recalculation on every request

**Causes**:
1. Cache TTL too low → Increase `PAGERANK_CACHE_TTL`
2. Cache not persisting → Check for errors in logs

### Issue: Slow response times

**Solutions**:
1. Increase cache TTL
2. Check database performance
3. Monitor concurrent computation (should be 1 per repo)

### Issue: Audit logs not appearing

**Solutions**:
1. Verify `AUDIT_LOG = true`
2. Check log level (must be INFO or lower)
3. Verify log destination is writable

---

## Security Changelog

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-04-08 | Initial secure implementation with auth, audit, caching |

---

## References

- [Gitea Permissions Documentation](https://docs.gitea.io/en-us/permissions/)
- [PageRank Algorithm](https://en.wikipedia.org/wiki/PageRank)
- [Gitea API Documentation](https://try.gitea.io/api/swagger)

---

**Document Maintainer**: Terraphim Security Team  
**Last Updated**: 2026-04-08T17:47:00Z
