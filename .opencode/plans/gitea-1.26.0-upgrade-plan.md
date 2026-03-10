# Gitea 1.26.0 Upgrade - Implementation Plan

**Status:** Plan Phase (Read-Only)  
**Target:** bigbox (100.106.66.7)  
**Image:** `ghcr.io/terraphim/gitea:1.26.0`  
**Robot API:** Enabled  
**CLI Tool:** Test in bigbox (not in container)

---

## Phase 1: Local Testing

### Step 1.1: Verify New Image Availability

```bash
# Pull new image locally
docker pull ghcr.io/terraphim/gitea:1.26.0

# Verify image details
docker images ghcr.io/terraphim/gitea:1.26.0
```

**Expected Output:**
- Image size: ~177 MB (regular)
- Created: Recent (should show today's date)

### Step 1.2: Create Local Test Environment

```bash
# Create test directory
mkdir -p ~/gitea-test-1.26.0
cd ~/gitea-test-1.26.0

# Create minimal docker-compose for testing
cat > docker-compose.yml << 'EOF'
version: "3"

networks:
  gitea:
    external: false

services:
  server:
    image: ghcr.io/terraphim/gitea:1.26.0
    container_name: gitea-test
    environment:
      - USER_UID=1000
      - USER_GID=1000
      - GITEA__server__DOMAIN=localhost
      - GITEA__server__ROOT_URL=http://localhost:3000/
      - GITEA__database__DB_TYPE=sqlite3
      - GITEA__actions__ENABLE=true
      # Robot API Configuration
      - GITEA__issue_graph__ENABLED=true
      - GITEA__issue_graph__DAMPING_FACTOR=0.85
      - GITEA__issue_graph__ITERATIONS=100
      - GITEA__issue_graph__PAGERANK_CACHE_TTL=300
      - GITEA__issue_graph__AUDIT_LOG=true
    restart: always
    networks:
      - gitea
    volumes:
      - ./gitea:/data
    ports:
      - "3000:3000"
EOF

# Start test instance
docker compose up -d

# Wait for startup
sleep 30
```

### Step 1.3: Verify Robot API Locally

```bash
# Check version
curl -s http://localhost:3000/api/v1/version | jq

# Expected: "version": "1.26.0"

# Test Robot API endpoints (should return 401 without auth)
curl -s http://localhost:3000/api/v1/robot/triage?owner=test\&repo=test
# Expected: 401 Unauthorized or 404 Not Found (not "404 page not found")

# Check if endpoint exists (404 means endpoint exists but repo doesn't)
# "404 page not found" means endpoint doesn't exist (bad)

# Verify app.ini has Robot config
docker compose exec server cat /data/gitea/conf/app.ini | grep -A 10 "issue_graph"

# Check logs for migration
docker compose logs --tail=50 server | grep -i "migration\|robot\|pagerank"
```

### Step 1.4: Test Basic Functionality Locally

```bash
# Create admin user through web UI or API
# http://localhost:3000/user/install

# Create test repository
# Create test issues with dependencies
# Test Robot API with authentication

# Stop local test
docker compose down
```

**Local Test Success Criteria:**
- [ ] Gitea 1.26.0 starts without errors
- [ ] Robot API endpoints respond (401/404 expected)
- [ ] Database migrations complete
- [ ] Configuration section `[issue_graph]` present in app.ini

---

## Phase 2: Update Infrastructure Code

### Step 2.1: Modify docker-compose.yml.template

**File:** `~/gitea-infrastructure/docker-compose.yml.template`

**Changes:**
1. Update image line 18:
   ```yaml
   # FROM:
   image: gitea/gitea:1.22.6
   
   # TO:
   image: ghcr.io/terraphim/gitea:1.26.0
   ```

2. Add Robot API environment variables (after line 43):
   ```yaml
   # Robot API Configuration
   - GITEA__issue_graph__ENABLED=true
   - GITEA__issue_graph__DAMPING_FACTOR=0.85
   - GITEA__issue_graph__ITERATIONS=100
   - GITEA__issue_graph__PAGERANK_CACHE_TTL=300
   - GITEA__issue_graph__AUDIT_LOG=true
   - GITEA__issue_graph__STRICT_MODE=false
   ```

### Step 2.2: Update Documentation

**Files to update:**
1. `~/gitea-infrastructure/README.md`
   - Update version number: 1.22.6 → 1.26.0
   - Add Robot API to features list

2. `~/gitea-infrastructure/HANDOVER.md`
   - Update version in architecture section

### Step 2.3: Commit Changes

```bash
cd ~/gitea-infrastructure
git add docker-compose.yml.template README.md HANDOVER.md
git commit -m "chore: Upgrade Gitea to 1.26.0 with Robot API

- Update Gitea image from 1.22.6 to 1.26.0
- Enable Robot API with PageRank algorithm
- Add issue_graph configuration section
- Update documentation"

git push origin main
```

---

## Phase 3: Pre-Deployment Preparation (bigbox)

### Step 3.1: SSH to bigbox and Backup

```bash
# SSH to bigbox
ssh bigbox

# Create backup directory with timestamp
BACKUP_DIR="~/gitea-backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p $BACKUP_DIR

cd ~/gitea-stack

# Stop Gitea gracefully
docker compose stop server

# Backup database
docker compose exec db pg_dump -U gitea gitea > $BACKUP_DIR/gitea-db.sql
echo "Database backup: $BACKUP_DIR/gitea-db.sql"

# Backup Gitea data
docker run --rm -v gitea_gitea:/data -v $BACKUP_DIR:/backup alpine \
  tar czf /backup/gitea-data.tar.gz -C /data .
echo "Data backup: $BACKUP_DIR/gitea-data.tar.gz"

# Backup current docker-compose.yml
cp docker-compose.yml $BACKUP_DIR/
echo "Config backup: $BACKUP_DIR/docker-compose.yml"

# List backups
ls -lh $BACKUP_DIR/
```

### Step 3.2: Verify Backup Integrity

```bash
# Check database backup size
ls -lh ~/gitea-backup-*/gitea-db.sql

# Should be > 1MB (depending on data)

# Verify data backup
ls -lh ~/gitea-backup-*/gitea-data.tar.gz
```

---

## Phase 4: Deploy to bigbox

### Step 4.1: Transfer Updated Configuration

```bash
# From local machine
cd ~/gitea-infrastructure

# Inject secrets into docker-compose.yml
op inject --in-file docker-compose.yml.template --out-file docker-compose.yml

# Transfer to bigbox
rsync -avz docker-compose.yml bigbox:~/gitea-stack/

# Verify transfer
ssh bigbox "cat ~/gitea-stack/docker-compose.yml | grep 'image:'"
# Should show: ghcr.io/terraphim/gitea:1.26.0
```

### Step 4.2: Execute Deployment

```bash
# SSH to bigbox
ssh bigbox

cd ~/gitea-stack

# Pull new image explicitly
docker compose pull server

# Verify image downloaded
docker images | grep gitea

# Start with new image
docker compose up -d server

# Watch logs for migration
docker compose logs -f server
```

### Step 4.3: Monitor Startup

**Watch for these log messages:**
```
# Good signs:
"Migration: migrate to 1.26.0"
"ORM engine initialization successful"
"Listen: http://0.0.0.0:3000"

# Bad signs:
"Migration failed"
"Database connection error"
"Permission denied"
```

**Wait time:** 60-90 seconds for full startup

---

## Phase 5: Post-Deployment Verification (bigbox)

### Step 5.1: Basic Health Checks

```bash
# On bigbox
cd ~/gitea-stack

# Check container status
docker compose ps

# Verify version
curl -s http://localhost:3000/api/v1/version | jq
# Expected: {"version": "1.26.0", ...}

# Check health endpoint
curl -s http://localhost:3000/api/healthz
# Expected: {"status": "pass"}

# Check container logs for errors
docker compose logs --tail=20 server | grep -i error
```

### Step 5.2: Verify Robot API

```bash
# Test Robot API endpoints (from bigbox)

# Without auth - should return 401 or 404 (not "404 page not found")
curl -s -o /dev/null -w "%{http_code}" \
  http://localhost:3000/api/v1/robot/triage?owner=test\&repo=test
# Expected: 401 or 404

# With auth - create token first through web UI, then:
# curl -H "Authorization: token YOUR_TOKEN" \
#   http://localhost:3000/api/v1/robot/triage?owner=terraphim\&repo=gitea

# Check if endpoint is registered
curl -s http://localhost:3000/api/v1/swagger | grep -i robot
# Should show robot endpoints
```

### Step 5.3: Verify Configuration

```bash
# Check app.ini has Robot config
docker compose exec server cat /data/gitea/conf/app.ini | grep -A 10 "\\[issue_graph\\]"

# Expected output:
# [issue_graph]
# ENABLED = true
# DAMPING_FACTOR = 0.85
# ITERATIONS = 100
# ...
```

### Step 5.4: Test gitea-robot CLI (in bigbox)

```bash
# SSH to bigbox
ssh bigbox

# Install Go if not present (or use existing)
# Or download pre-built gitea-robot binary

# Option 1: Build from source
cd /tmp
git clone https://github.com/AlexMikhalev/gitea.git
cd gitea
go build -o /usr/local/bin/gitea-robot ./cmd/gitea-robot

# Option 2: Copy from local build
# scp gitea-robot bigbox:/usr/local/bin/

# Configure CLI
export GITEA_URL="https://git.terraphim.cloud"
export GITEA_TOKEN="your-api-token-from-web-ui"

# Test CLI
gitea-robot triage --owner terraphim --repo gitea
gitea-robot ready --owner terraphim --repo gitea
gitea-robot graph --owner terraphim --repo gitea
```

### Step 5.5: Full Functionality Test

**Manual Tests:**
1. Access https://git.terraphim.cloud - Should load web UI
2. Login with existing credentials - Should work
3. Browse repositories - All repos visible
4. Create test issue - Should succeed
5. Create dependency between issues - Should work
6. Test Robot API with token - Should return JSON

**Automated Test Script:**
```bash
#!/bin/bash
# save as ~/test-gitea-1.26.0.sh

set -e

echo "=== Gitea 1.26.0 Verification Tests ==="

# Test 1: Version
echo -n "Test 1: Version check... "
VERSION=$(curl -s http://localhost:3000/api/v1/version | jq -r '.version')
if [ "$VERSION" = "1.26.0" ]; then
    echo "PASS (version: $VERSION)"
else
    echo "FAIL (expected: 1.26.0, got: $VERSION)"
    exit 1
fi

# Test 2: Health
echo -n "Test 2: Health check... "
HEALTH=$(curl -s http://localhost:3000/api/healthz | jq -r '.status')
if [ "$HEALTH" = "pass" ]; then
    echo "PASS"
else
    echo "FAIL (status: $HEALTH)"
    exit 1
fi

# Test 3: Robot API endpoint exists
echo -n "Test 3: Robot API endpoint... "
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  http://localhost:3000/api/v1/robot/triage?owner=test\&repo=test)
if [ "$HTTP_CODE" = "401" ] || [ "$HTTP_CODE" = "404" ]; then
    echo "PASS (HTTP $HTTP_CODE)"
else
    echo "FAIL (HTTP $HTTP_CODE)"
    exit 1
fi

# Test 4: Configuration
echo -n "Test 4: Robot configuration... "
cd ~/gitea-stack
if docker compose exec -T server cat /data/gitea/conf/app.ini | grep -q "ENABLED = true"; then
    echo "PASS"
else
    echo "FAIL"
    exit 1
fi

echo ""
echo "=== All tests PASSED ==="
```

---

## Phase 6: Rollback Plan (If Needed)

### Rollback Procedure

```bash
# SSH to bigbox
ssh bigbox

# Identify backup to restore
BACKUP_DIR="~/gitea-backup-YYYYMMDD-HHMMSS"  # Use actual timestamp

cd ~/gitea-stack

# Step 1: Stop Gitea
docker compose stop server

# Step 2: Restore database
docker compose exec -T db psql -U gitea gitea < $BACKUP_DIR/gitea-db.sql

# Step 3: Restore Gitea data
docker run --rm -v gitea_gitea:/data -v $BACKUP_DIR:/backup alpine \
  sh -c "cd /data && rm -rf * && tar xzf /backup/gitea-data.tar.gz"

# Step 4: Restore configuration
cp $BACKUP_DIR/docker-compose.yml .

# Step 5: Start with old version
docker compose up -d server

# Step 6: Verify rollback
curl -s http://localhost:3000/api/v1/version | jq -r '.version'
# Expected: "1.22.6"
```

**Rollback Time:** ~5 minutes

---

## Phase 7: Documentation Update

### Update Files

1. **~/gitea-infrastructure/README.md**
   - Update version badge: 1.22.6 → 1.26.0
   - Add Robot API to features list
   - Add Robot API usage section

2. **~/gitea-infrastructure/HANDOVER.md**
   - Update version in architecture diagram
   - Add Robot API endpoints to services table

3. **Create ~/gitea-infrastructure/.docs/ROBOT-API-USAGE.md**
   - API endpoint documentation
   - CLI usage examples
   - Configuration reference

---

## Timeline Estimate

| Phase | Duration | Cumulative |
|-------|----------|------------|
| Phase 1: Local Testing | 30 min | 30 min |
| Phase 2: Code Updates | 20 min | 50 min |
| Phase 3: Backup (bigbox) | 15 min | 65 min |
| Phase 4: Deploy (bigbox) | 10 min | 75 min |
| Phase 5: Verification | 20 min | 95 min |
| Phase 6: Documentation | 15 min | 110 min |
| **Total** | **~2 hours** | |

**Recommended Window:** Schedule 3-hour maintenance window

---

## Success Criteria

- [ ] Gitea 1.26.0 running on bigbox
- [ ] Version API returns "1.26.0"
- [ ] All repositories accessible
- [ ] Robot API endpoints respond (401/404 expected)
- [ ] Database migrations completed
- [ ] Configuration section `[issue_graph]` present
- [ ] gitea-robot CLI works from bigbox
- [ ] No errors in logs
- [ ] Web UI fully functional
- [ ] HTTPS working via Caddy

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Database migration fails | Low | High | Full backup before upgrade |
| Robot API not working | Low | Medium | Local testing first |
| Image pull fails | Low | High | Multiple registry mirrors |
| SSH connection lost | Low | High | Have console access ready |
| Data corruption | Very Low | Critical | Tested backup/restore |
| Downtime extended | Medium | Medium | Clear rollback plan |

---

## Approval Checklist

Before executing this plan:

- [ ] Plan reviewed by stakeholder
- [ ] Maintenance window scheduled
- [ ] Users notified of downtime
- [ ] Backup storage space verified (>2GB free)
- [ ] Rollback procedure tested (if possible)
- [ ] Emergency contact available

**Approved by:** _________________  
**Date:** _________________  
**Execute during:** _________________ (time window)

---

## Post-Deployment Tasks

After successful deployment:

1. Monitor logs for 24 hours
2. Check error rates
3. Verify backup jobs still work
4. Update monitoring dashboards
5. Notify users of new features
6. Schedule follow-up review in 1 week
