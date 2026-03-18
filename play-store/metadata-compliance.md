# Play Store Metadata & Compliance

> Last Updated: 2026-02-25
> App: ArmorChat
> Package: com.armorclaw.app

---

## 1. Content Rating (IARC Questionnaire)

### Pre-filled Responses for International Age Rating Coalition

#### GENERAL

**Does your app contain any of the following?**

| Category | Response |
|----------|----------|
| Violence | **No** |
| Sexual Content/Nudity | **No** |
| Profanity/Crude Humor | **No** |
| Drugs, Alcohol, Tobacco | **No** |
| Gambling | **No** |
| User Interaction | **Yes** (messaging app) |
| Shares Information | **Yes** (messages, account info) |
| Shares Location | **No** |
| Digital Purchases | **No** |

---

#### USER INTERACTION (Detailed)

**Can users communicate with each other?**
- Response: **Yes**

**Can users exchange personal information?**
- Response: **Yes** (messages may contain personal info)

**Is there moderated/unmoderated chat?**
- Response: **Unmoderated** (E2E encrypted - we cannot moderate)

**Can users share location?**
- Response: **No**

**Are there parental controls?**
- Response: **No**

---

#### DATA COLLECTION

**Does your app collect personal information?**
- Response: **Yes**

**What types of data?**
- [x] Email address (optional for support)
- [x] User-generated content (messages - encrypted)
- [x] Device identifiers (for app stability)
- [x] Crash data (Sentry, Firebase)

**Is data encrypted?**
- Response: **Yes** (AES-256-GCM for messages, SQLCipher for database)

**Can users delete their data?**
- Response: **Yes** (account deletion available)

---

#### EXPECTED RATINGS

Based on responses:
| Region | Expected Rating |
|--------|-----------------|
| PEGI (Europe) | PEGI 3 |
| ESRB (North America) | Everyone |
| USK (Germany) | All ages |
| ClassInd (Brazil) | L (General Audiences) |
| CERO (Japan) | All ages |
| ACB (Australia) | G |

---

## 2. Data Safety Section (Google Play)

### Data Collection Disclosure

#### Personal Info
| Data Type | Collected | Encrypted | Can Request Deletion |
|-----------|-----------|-----------|---------------------|
| Email | Optional | Yes (transit) | Yes |
| Name | Optional | Yes (transit) | Yes |
| User ID | Yes | Yes | Yes |

#### Messages & Content
| Data Type | Collected | Encrypted | Can Request Deletion |
|-----------|-----------|-----------|---------------------|
| Messages | Yes | **E2E Encrypted** | Yes |
| Attachments | Yes | **E2E Encrypted** | Yes |

#### App Activity
| Data Type | Collected | Encrypted | Can Request Deletion |
|-----------|-----------|-----------|---------------------|
| App interactions | Yes (analytics) | Yes | Yes |
| Crash logs | Yes | Yes | Yes |

#### Device Info
| Data Type | Collected | Encrypted | Can Request Deletion |
|-----------|-----------|-----------|---------------------|
| Device ID | Yes | Yes | Yes |
| OS Version | Yes | Yes | Yes |

---

### Data Sharing Disclosure

**Do you share data with third parties?**

| Purpose | Shared | With Whom |
|---------|--------|-----------|
| Analytics | Yes | Firebase (Google) |
| Crash Reporting | Yes | Sentry |
| Advertising | **No** | N/A |
| Data Brokers | **No** | N/A |

---

### Security Practices

**Declare your security practices:**

- [x] Data is encrypted in transit
- [x] Data is encrypted at rest
- [x] You can request data deletion
- [x] Independent security review (optional - mark if applicable)

---

### Data Safety Summary (for Play Store display)

```
This app may collect these data types:
• Personal info (email, name - optional)
• Messages (end-to-end encrypted)
• App activity (analytics)
• Device info

Data is encrypted in transit and at rest.
You can request data deletion at any time.

Data shared with:
• Firebase (analytics, crash reporting)
• Sentry (crash reporting)

No data shared for advertising purposes.
```

---

## 3. Target Audience & Content

### Target Audience Declaration

**Is your app directed at children under 13?**
- Response: **No**

**Does your app attract children even if not targeted at them?**
- Response: **No** (enterprise/team messaging)

**Primary Target Audience:**
- Adults 18+
- Business/Enterprise users
- Security-conscious professionals

---

### Content Declarations

| Content Type | Present |
|--------------|---------|
| Violence | No |
| Sexual Content | No |
| Strong Language | No |
| Controlled Substances | No |
| User-Generated Content | Yes (messages) |
| Financial Transactions | No |

---

## 4. Export Compliance

### Encryption Export Classification

**Does your app use encryption?**
- Response: **Yes**

**Encryption Details:**
- **Algorithm:** AES-256-GCM
- **Key Exchange:** ECDH (Elliptic Curve Diffie-Hellman)
- **Key Size:** 256-bit
- **Database:** SQLCipher (AES-256)
- **Transport:** TLS 1.3

### US Export Administration Regulations (EAR)

**ECCN Classification:** 5D002.c.1
- Mass market encryption products
- Key length > 64 bits
- Subject to reporting requirements

### Google Play Export Compliance

**Questions to answer:**

1. **Does your app use encryption?**
   - Yes

2. **Is the encryption for data at rest?**
   - Yes (SQLCipher database encryption)

3. **Is the encryption for data in transit?**
   - Yes (TLS 1.3, end-to-end encryption)

4. **Is this a mass market product?**
   - Yes (publicly available on Play Store)

5. **Does your app implement standard encryption?**
   - Yes (AES, TLS, industry standards)

**Compliance Status:**
- Encryption registration may be required for some jurisdictions
- Annual reporting to BIS (Bureau of Industry and Security) if applicable
- Most mass-market encryption products are eligible for License Exception ENC

---

## 5. Regional Requirements

### European Union (GDPR)

**Data Protection Officer Contact:** support@armorclaw.app

**GDPR Compliance Checklist:**
- [x] Privacy policy available
- [x] Data minimization (collect only what's needed)
- [x] Purpose limitation (stated in privacy policy)
- [x] Data subject rights (access, deletion, portability)
- [x] Encryption by default
- [x] Data breach notification process

### Germany (JMStV)

**Age Rating:** USK All Ages
**Content Monitoring:** Not required (user-generated content is encrypted)

### South Korea (Korea Game Rating Board)

**Classification:** Not a game - exempt from game rating requirements

### China

**Status:** Not distributing in China mainland (requires separate ICP filing)

---

## 6. Developer Verification

### Identity Verification (Required)

**Documents needed:**
- [ ] Government-issued ID
- [ ] Business registration (if applicable)
- [ ] Address verification

**For Organization Account:**
- [ ] D-U-N-S number (if applicable)
- [ ] Business license
- [ ] Authorization letter

---

## 7. Pre-submission Checklist

### Required Items

- [ ] Privacy policy URL hosted and accessible
- [ ] App content rating questionnaire completed
- [ ] Data safety section filled out
- [ ] Target audience selected
- [ ] Export compliance answered
- [ ] Developer identity verified

### Recommended Items

- [ ] Terms of Service URL
- [ ] Support email verified
- [ ] Website URL provided
- [ ] App access instructions (if login required)

---

## 8. App Access for Review

### Test Account (if required)

Google Play may require a test account to review your app:

**Test Credentials:**
- Username: `[Create dedicated test account]`
- Password: `[Secure password]`

**Instructions for Reviewer:**
```
1. Open ArmorChat
2. Tap "Create Account" on welcome screen
3. Enter test credentials above
4. Verify email (or use pre-verified test account)
5. Access all features

Note: Biometric authentication can be skipped.
```

---

## Quick Reference: Play Console Locations

| Setting | Play Console Location |
|---------|----------------------|
| Content Rating | Policy > App content > Content rating |
| Data Safety | Policy > App content > Data safety |
| Target Audience | Policy > App content > Target audience |
| Privacy Policy | Policy > App content > Privacy policy |
| Export Compliance | Setup > Advanced settings > App signing |
| Developer Verification | Account > Developer account |

---

*Document prepared for ArmorChat Play Store submission*
