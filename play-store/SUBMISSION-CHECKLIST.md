# Play Store Submission Checklist

> ArmorChat - Complete Submission Guide
> Last Updated: 2026-02-25

---

## Pre-Submission Requirements

### ✅ Build Configuration
- [ ] Keystore created and backed up securely
- [ ] `keystore.properties` configured (not committed to git)
- [ ] Release build succeeds: `./gradlew bundleRelease`
- [ ] AAB signed and verified
- [ ] Version code: `10000` (1.0.0)
- [ ] Version name: `1.0.0`

### ✅ Store Listing Assets
- [ ] App title: `ArmorChat - Secure Team Chat`
- [ ] Short description (80 chars max)
- [ ] Full description (4000 chars max)
- [ ] Feature graphic (1024x500 px) - **CREATE**
- [ ] Phone screenshots (4-8) - **CAPTURE**
- [ ] High-res icon (512x512 px) - **EXPORT**

### ✅ Legal & Compliance
- [ ] Privacy policy hosted at accessible URL
- [ ] Terms of service (optional but recommended)
- [ ] Export compliance answered
- [ ] Content rating questionnaire completed
- [ ] Data safety section filled out
- [ ] Target audience declared (18+, not for children)

### ✅ Developer Account
- [ ] Google Play Developer account ($25 one-time fee)
- [ ] Developer identity verified
- [ ] Payment profile set up (if selling)
- [ ] Merchant account linked (if in-app purchases)

---

## Play Console Setup

### Step 1: Create App
1. Go to [Play Console](https://play.google.com/console)
2. Click "Create app"
3. Fill in:
   - App name: `ArmorChat - Secure Team Chat`
   - Default language: English
   - Free or paid: Free
   - Declarations: ✅ All checked

### Step 2: Store Listing
Navigate: **Main Store Listing**

- [ ] App name entered
- [ ] Short description entered
- [ ] Full description entered
- [ ] App icon uploaded (512x512)
- [ ] Feature graphic uploaded (1024x500)
- [ ] Phone screenshots uploaded (min 2, recommended 4-8)
- [ ] Category: **Communication**
- [ ] Secondary category: **Business** (optional)
- [ ] Tags added: encrypted, messaging, security, chat
- [ ] Privacy policy URL entered

### Step 3: Content Rating
Navigate: **Policy → App content → Content rating**

- [ ] Start questionnaire
- [ ] Select category: **Communication**
- [ ] Answer all questions (see `metadata-compliance.md`)
- [ ] Generate and apply rating certificate

### Step 4: Target Audience
Navigate: **Policy → App content → Target audience**

- [ ] Select: Not directed at children
- [ ] Confirm app doesn't attract children
- [ ] Age selection: 18+

### Step 5: Data Safety
Navigate: **Policy → App content → Data safety**

- [ ] Complete data collection survey (see `data-safety-form.md`)
- [ ] Add all collected data types
- [ ] Mark encryption status
- [ ] Add third-party sharing (Firebase, Sentry)
- [ ] Preview and submit

### Step 6: App Access
Navigate: **Setup → App access**

- [ ] Add test account credentials (if login required)
- [ ] Provide access instructions for reviewers

### Step 7: Export Compliance
Navigate: **Setup → Advanced settings**

- [ ] Answer encryption questions
- [ ] Confirm export compliance

---

## Upload & Release

### Step 1: Upload AAB
Navigate: **Release → Testing → Internal testing**

1. Click "Create new release"
2. Upload: `androidApp-release.aab`
3. Google will process and validate

### Step 2: Release Notes
Enter release notes for this version:
```
Welcome to ArmorChat - secure messaging for teams!

🛡️ End-to-end encryption (ECDH + AES-256-GCM)
🔐 Biometric authentication
💬 Rich messaging with reactions & replies
📱 Full offline support
🌙 Dark mode included

Your messages. Your privacy. Period.
```

### Step 3: Review & Rollout
1. Click "Review release"
2. Check for any warnings or errors
3. Click "Start rollout to Internal"
4. Wait for processing (usually 1-2 hours)

### Step 4: Test Internal Release
1. Add testers (email list or Google Group)
2. Testers receive opt-in link
3. Verify installation and critical flows
4. Monitor crashes in Play Console

### Step 5: Promote to Production
1. After internal testing passes
2. Navigate to Release → Production
3. Click "Create new release"
4. Import from Internal track
5. Set rollout percentage (start with 5-20%)
6. Submit for review

---

## Post-Submission

### Monitoring (First 48 Hours)
- [ ] Check crash reports daily
- [ ] Monitor ANR rate
- [ ] Read and respond to reviews
- [ ] Watch install/uninstall metrics
- [ ] Check Firebase/Sentry dashboards

### Rollout Management
- [ ] Day 1-2: 5-20% rollout
- [ ] Day 3-4: Increase to 50% if stable
- [ ] Day 5+: Full 100% rollout

### If Issues Found
1. Halt rollout (Play Console → Production → Halt)
2. Investigate and fix
3. Increment version code
4. Upload new release
5. Resume rollout

---

## File Locations

```
play-store/
├── SUBMISSION-CHECKLIST.md   ← You are here
├── store-listing.md          ← Copy-paste for store listing
├── privacy-policy.md         ← Host this file
├── release-notes.md          ← Templates for releases
├── metadata-compliance.md    ← IARC & compliance answers
├── data-safety-form.md       ← Data safety quick-fill
└── release-management.md     ← Version & rollout strategy

.github/workflows/
├── ci.yml                    ← Continuous integration
└── release.yml               ← Automated releases
```

---

## Quick Commands

```bash
# Build release AAB
./gradlew clean bundleRelease

# Verify signing
jarsigner -verify -verbose androidApp/build/outputs/bundle/release/androidApp-release.aab

# Run tests before release
./gradlew test

# Static analysis
./gradlew detekt

# Install debug on device
./gradlew installDebug
```

---

## Support Resources

- [Play Console Help](https://support.google.com/googleplay/android-developer/)
- [Data Safety Policy](https://support.google.com/googleplay/android-developer/answer/10787469)
- [Content Rating](https://support.google.com/googleplay/android-developer/answer/9894510)
- [Release Management](https://support.google.com/googleplay/android-developer/topic/9858218)

---

## Emergency Contacts

| Issue | Contact |
|-------|---------|
| App Store Issues | Play Console Help |
| Critical Bugs | GitHub Issues |
| User Support | support@armorclaw.app |

---

**Status:** Ready for submission once assets are created

**Remaining items:**
1. Create feature graphic (1024x500)
2. Capture phone screenshots (4-8)
3. Export high-res icon (512x512)
4. Host privacy policy

---

*Good luck with your release! 🚀*
