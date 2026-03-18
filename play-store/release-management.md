# Release Management Guide

> Last Updated: 2026-02-25
> App: ArmorChat
> Package: com.armorclaw.app

---

## 1. Version Strategy

### Version Code Calculation

**Current:** `versionCode = 1`, `versionName = "1.0.0"`

**Recommended Strategy:** Multiplier-based for granular control

```
versionCode = MAJOR * 10000 + MINOR * 100 + PATCH

Example:
- 1.0.0 → 10000
- 1.0.1 → 10001
- 1.1.0 → 10100
- 2.0.0 → 20000
```

**Benefits:**
- Each release has unique, incrementing code
- Easy to calculate from version name
- Room for 99 minor versions and 99 patches per major

### Version Naming Convention

**Format:** `MAJOR.MINOR.PATCH` (Semantic Versioning)

| Change Type | Example | When to Use |
|-------------|---------|-------------|
| **MAJOR** | 2.0.0 | Breaking changes, major redesign |
| **MINOR** | 1.1.0 | New features, enhancements |
| **PATCH** | 1.0.1 | Bug fixes, minor improvements |

### Pre-release Tags

| Tag | Example | Use Case |
|-----|---------|----------|
| `-alpha` | 1.1.0-alpha | Internal testing |
| `-beta` | 1.1.0-beta | External beta testers |
| `-rc` | 1.1.0-rc | Release candidate |

---

## 2. Testing Tracks

### Track Hierarchy

```
Internal → Alpha → Beta → Production
    ↓         ↓       ↓        ↓
   Team    Closed   Open    Public
           Group   Users
```

### Internal Track

**Purpose:** Quick internal testing, CI/CD builds

| Setting | Recommendation |
|---------|----------------|
| Testers | Development team only |
| Rollout | 100% |
| Update frequency | Every build |
| Duration | 1-2 days |

**Setup:**
```
Play Console → Testing → Internal testing
Add testers: dev-team@armorclaw.app (Google Group)
```

### Alpha Track (Closed Testing)

**Purpose:** Feature validation with trusted users

| Setting | Recommendation |
|---------|----------------|
| Testers | 10-50 trusted users |
| Rollout | 100% |
| Update frequency | Weekly or per feature |
| Duration | 3-7 days |

**Setup:**
```
Play Console → Testing → Closed testing → Create track
Name: "Alpha - Trusted Testers"
Add email list or Google Group
```

**Tester Recruitment:**
- Existing users who opt-in
- Team members' contacts
- Security-focused communities
- GitHub community

### Beta Track (Open Testing)

**Purpose:** Broader testing before production

| Setting | Recommendation |
|---------|----------------|
| Testers | Open (anyone with link) |
| Rollout | 100% |
| Update frequency | Before major releases |
| Duration | 5-14 days |

**Setup:**
```
Play Console → Testing → Open testing
Enable: "Allow anyone to join"
```

### Production Track

**Purpose:** Public release

| Setting | Recommendation |
|---------|----------------|
| Rollout | Staged (see below) |
| Update frequency | As needed |
| Review time | 1-3 days typical |

---

## 3. Rollout Strategy

### Staged Rollout Phases

| Phase | Percentage | Duration | Monitoring |
|-------|------------|----------|------------|
| 1 | 5% | 24-48 hours | Crash rates, ANRs |
| 2 | 20% | 24-48 hours | User feedback, ratings |
| 3 | 50% | 24 hours | Performance metrics |
| 4 | 100% | Full release | Continue monitoring |

### Rollout Decision Matrix

| Metric | Threshold | Action if Exceeded |
|--------|-----------|-------------------|
| Crash rate | < 0.5% | Pause rollout |
| ANR rate | < 0.2% | Investigate |
| 1-star reviews | < 5% | Read and respond |
| Uninstall rate | < 10% increase | Investigate |

### Emergency Rollback

If critical issue found:

1. **Play Console:** Production → Create release → Deactivate
2. **Promote previous stable release** from Beta/Alpha
3. **Fix issue** in development
4. **New release** with version bump

---

## 4. Release Workflow

### Pre-Release Checklist

```markdown
## Code Quality
- [ ] All tests passing: `./gradlew test`
- [ ] Static analysis clean: `./gradlew detekt`
- [ ] No compiler warnings
- [ ] Code reviewed and approved

## Version Management
- [ ] Version code incremented
- [ ] Version name updated
- [ ] CHANGELOG.md updated
- [ ] Release notes written

## Build Verification
- [ ] Clean build: `./gradlew clean`
- [ ] Release build: `./gradlew bundleRelease`
- [ ] Install and test on physical device
- [ ] Verify signing: `jarsigner -verify app.aab`

## Store Listing
- [ ] Screenshots updated (if UI changed)
- [ ] Descriptions updated (if features added)
- [ ] Privacy policy updated (if data practices changed)
- [ ] Data safety updated (if collection changed)

## Documentation
- [ ] README.md updated
- [ ] API docs updated (if applicable)
- [ ] Migration guide (if breaking changes)
```

### Release Day Procedure

```
1. MORNING (avoid Friday releases)
   ├── Verify all tests pass
   ├── Build final release AAB
   └── Test on device

2. UPLOAD
   ├── Upload AAB to Internal track
   ├── Test download and install
   └── Verify critical flows

3. PROMOTE
   ├── Promote to Alpha (if testing)
   └── OR promote to Production

4. MONITOR
   ├── Watch crash reports (Sentry, Firebase)
   ├── Monitor Play Console metrics
   └── Check for user feedback

5. COMMUNICATE
   ├── GitHub release created
   ├── Changelog published
   └── Users notified (if major release)
```

---

## 5. CI/CD Integration

### Recommended Pipeline

```yaml
# .github/workflows/release.yml (example)

name: Release Build

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up JDK 17
        uses: actions/setup-java@v4
        with:
          java-version: '17'
          distribution: 'temurin'

      - name: Run tests
        run: ./gradlew test

      - name: Build release AAB
        run: ./gradlew bundleRelease
        env:
          KEYSTORE_PASSWORD: ${{ secrets.KEYSTORE_PASSWORD }}
          KEY_ALIAS: ${{ secrets.KEY_ALIAS }}
          KEY_PASSWORD: ${{ secrets.KEY_PASSWORD }}

      - name: Upload to Play Store (Internal)
        uses: r0adkll/upload-google-play@v1
        with:
          serviceAccountJsonPlainText: ${{ secrets.PLAY_SERVICE_ACCOUNT }}
          packageName: com.armorclaw.app
          releaseFiles: androidApp/build/outputs/bundle/release/androidApp-release.aab
          track: internal
          status: completed
```

### GitHub Secrets Required

| Secret | Description |
|--------|-------------|
| `KEYSTORE_PASSWORD` | Keystore store password |
| `KEY_ALIAS` | Key alias name |
| `KEY_PASSWORD` | Key password |
| `PLAY_SERVICE_ACCOUNT` | Google Play Service Account JSON |

---

## 6. Play Store Service Account Setup

### Create Service Account

1. **Google Cloud Console** → IAM & Admin → Service Accounts
2. **Create Service Account**
   - Name: `play-store-armorchat`
   - Role: No role (set in Play Console)
3. **Create Key** → JSON format
4. **Download** and store securely

### Grant Play Console Access

1. **Play Console** → Setup → API access
2. **Link** Google Cloud project
3. **Add** service account email
4. **Grant permissions:**
   - View app information
   - View financial data (optional)
   - Manage production releases
   - Manage testing tracks

---

## 7. Monitoring & Alerts

### Key Metrics to Track

| Metric | Source | Alert Threshold |
|--------|--------|-----------------|
| Crash-free users | Firebase/Play Console | < 99% |
| ANR rate | Play Console | > 0.1% |
| Bad reviews | Play Console | Any < 3 stars |
| Uninstalls | Play Console | > 20% in 7 days |
| API errors | Sentry | > 1% error rate |

### Alert Setup

**Play Console Alerts:**
- Setup → Preferences → Email notifications
- Enable: Crashes, ANRs, Reviews

**Firebase Alerts:**
- Crashlytics → App → Alerts
- Enable: New fatal issues, regressed issues

**Sentry Alerts:**
- Project → Alerts → Create alert
- Trigger: Error rate > 1% in 1 hour

---

## 8. Release Calendar Template

### Monthly Release Cycle

```
Week 1: Feature freeze
├── Code complete
├── Internal testing begins
└── Alpha release (Thursday)

Week 2: Stabilization
├── Bug fixes only
├── Alpha feedback incorporated
└── Beta release (Thursday)

Week 3: Final testing
├── Beta feedback addressed
├── Release candidate built
└── Production release (Wednesday)

Week 4: Monitoring & hotfixes
├── Monitor rollout
├── Address any critical issues
└── Plan next release
```

### Release Windows

**Best days:** Tuesday, Wednesday, Thursday
**Avoid:** Friday, weekends, holidays

**Best times:** Morning (9-11 AM local time)
**Avoid:** End of day, before weekends

---

## 9. Hotfix Procedure

### When to Hotfix

- Critical crash affecting > 1% of users
- Security vulnerability
- Data loss bug
- App store rejection

### Hotfix Workflow

```
1. ASSESS
   ├── Confirm severity (critical?)
   ├── Identify root cause
   └── Estimate fix complexity

2. FIX
   ├── Create hotfix branch from release tag
   ├── Implement minimal fix
   ├── Test thoroughly
   └── Version bump (PATCH only)

3. RELEASE
   ├── Fast-track through Internal
   ├── Skip Alpha/Beta if critical
   ├── Rollout to 100% if critical
   └── Otherwise follow staged rollout

4. COMMUNICATE
   ├── Update Play Store release notes
   ├── Respond to affected reviews
   └── Document in changelog
```

---

## 10. Version History Template

### Track Releases

| Version | Date | Track | Notes |
|---------|------|-------|-------|
| 1.0.0 | TBD | Production | Initial release |
| 1.0.1 | TBD | Production | Bug fixes |
| 1.1.0 | TBD | Beta | Feature X, Y, Z |

---

## Quick Reference

### Build Commands
```bash
# Debug build
./gradlew assembleDebug

# Release APK
./gradlew assembleRelease

# Release AAB (for Play Store)
./gradlew bundleRelease

# Run tests
./gradlew test

# Static analysis
./gradlew detekt
```

### Version Update
```kotlin
// androidApp/build.gradle.kts
versionCode = 10001  // Increment!
versionName = "1.0.1"
```

### Play Console URLs
- Dashboard: `play.google.com/console`
- ArmorChat: `play.google.com/console/developers/.../app/...`
- Testing: `play.google.com/console/developers/.../app/.../testing`
- Releases: `play.google.com/console/developers/.../app/.../releases`

---

*Release management guide for ArmorChat*
