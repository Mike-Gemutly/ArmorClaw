# Data Safety Form - Quick Fill Guide

> Google Play Data Safety Section Responses
> App: ArmorChat

---

## Step 1: Data Collection Overview

**Does your app collect or share any of the required user data types?**

✅ **Yes**

---

## Step 2: Collected Data Types

### Personal Info

| Data Type | Collected? | Optional/Required |
|-----------|------------|-------------------|
| Name | ✅ Yes | Optional |
| Email address | ✅ Yes | Optional |
| User IDs | ✅ Yes | Required |
| Address | ❌ No | - |
| Phone number | ❌ No | - |
| Race and ethnicity | ❌ No | - |
| Political or religious beliefs | ❌ No | - |
| Sexual orientation | ❌ No | - |
| Other personal info | ❌ No | - |

### Financial Info

| Data Type | Collected? |
|-----------|------------|
| User payment info | ❌ No |
| Credit score | ❌ No |
| Other financial info | ❌ No |

### Health and Fitness

| Data Type | Collected? |
|-----------|------------|
| Health info | ❌ No |
| Fitness info | ❌ No |

### Messages

| Data Type | Collected? | Encrypted? |
|-----------|------------|------------|
| Emails | ❌ No | - |
| SMS or MMS | ❌ No | - |
| Other in-app messages | ✅ Yes | ✅ E2E Encrypted |

### Photos and Videos

| Data Type | Collected? |
|-----------|------------|
| Photos | ❌ No (user attachments are E2E encrypted) |
| Videos | ❌ No (user attachments are E2E encrypted) |

### Audio Files

| Data Type | Collected? |
|-----------|------------|
| Voice or sound recordings | ❌ No (voice messages are E2E encrypted) |
| Music files | ❌ No |

### Files and Docs

| Data Type | Collected? |
|-----------|------------|
| Files and docs | ❌ No (user attachments are E2E encrypted) |

### Calendar

| Data Type | Collected? |
|-----------|------------|
| Calendar events | ❌ No |

### Contacts

| Data Type | Collected? |
|-----------|------------|
| Contacts | ❌ No |

### App Activity

| Data Type | Collected? |
|-----------|------------|
| App interactions | ✅ Yes |
| In-app search history | ❌ No |
| Installed apps | ❌ No |
| Other user-generated content | ❌ No |
| Other actions | ❌ No |

### Web Browsing

| Data Type | Collected? |
|-----------|------------|
| Web browsing history | ❌ No |

### App Info and Performance

| Data Type | Collected? |
|-----------|------------|
| Crash logs | ✅ Yes |
| Diagnostics | ✅ Yes |
| Other app performance data | ✅ Yes |

### Device Info or Other IDs

| Data Type | Collected? |
|-----------|------------|
| Device or other IDs | ✅ Yes |

---

## Step 3: Data Sharing

**For each collected data type, answer:**

### User ID
- **Shared with third parties?** No
- **Purpose of collection:** App functionality, authentication

### App Interactions
- **Shared with third parties?** Yes - Firebase Analytics
- **Purpose:** Analytics, app improvement

### Crash Logs
- **Shared with third parties?** Yes - Firebase Crashlytics, Sentry
- **Purpose:** App stability, bug fixing

### Diagnostics
- **Shared with third parties?** Yes - Firebase
- **Purpose:** Performance monitoring

### Device IDs
- **Shared with third parties?** Yes - Firebase, Sentry
- **Purpose:** Analytics, crash reporting

### Messages (Other in-app messages)
- **Shared with third parties?** No
- **Purpose:** Core app functionality (E2E encrypted)
- **Encrypted:** Yes, end-to-end

---

## Step 4: Security Practices

### Check all that apply:

| Practice | Selected |
|----------|----------|
| Data is encrypted in transit | ✅ Yes |
| Data is encrypted at rest | ✅ Yes |
| You can request that data be deleted | ✅ Yes |
| Committed to following the Play Families Policy | ❌ Not applicable |
| Independent security review | ❌ No (optional) |

---

## Step 5: Data Handling for Each Type

### Personal Info (Name, Email, User ID)

**Is this data collected?** Yes
**Is this data shared?** No (except Firebase for analytics)
**Is this data processed ephemerally?** No

**Purposes:**
- [x] Account management

### Messages

**Is this data collected?** Yes
**Is this data shared?** No
**Is this data processed ephemerally?** No
**Is this data encrypted?** Yes (E2E)

**Purposes:**
- [x] App functionality
- [x] Messaging

### App Activity (App interactions)

**Is this data collected?** Yes
**Is this data shared?** Yes (Firebase)

**Purposes:**
- [x] Analytics
- [x] App functionality

### App Info & Performance (Crash logs, Diagnostics)

**Is this data collected?** Yes
**Is this data shared?** Yes (Firebase, Sentry)

**Purposes:**
- [x] App functionality
- [x] Analytics

### Device Info (Device IDs)

**Is this data collected?** Yes
**Is this data shared?** Yes (Firebase, Sentry)

**Purposes:**
- [x] App functionality
- [x] Analytics

---

## Final Summary Preview

**What Play Store will display:**

```
Safety starts with understanding how developers collect and share your data.
Data privacy and security practices may vary based on your use, region,
and age. The developer provided this information and may update it over time.

This app may collect these data types:
• Personal info, Messages, App activity, and Device or other IDs

This app may share these data types with third parties:
• App activity, App info and performance, and Device or other IDs

Data is encrypted in transit
You can request that data be deleted
See details

[ arrow/collapse ]

Data collected:
• Personal info: Name, Email, User ID
• Messages: Other in-app messages
• App activity: App interactions
• App info and performance: Crash logs, Diagnostics
• Device or other IDs: Device or other IDs

Data shared:
• App activity: App interactions (Firebase Analytics)
• App info and performance: Crash logs, Diagnostics (Firebase, Sentry)
• Device or other IDs: Device or other IDs (Firebase, Sentry)

Security practices:
• Data is encrypted in transit
• Data is encrypted at rest
• You can request that data be deleted
```

---

## Common Mistakes to Avoid

❌ **Don't forget:**
- Mark messages as "Encrypted" (E2E)
- Include Firebase and Sentry as data sharing
- Check "encrypted in transit" AND "encrypted at rest"
- Enable "request data deletion"

❌ **Don't over-report:**
- User attachments (photos, files) are E2E encrypted - not "collected" in readable form
- Don't list data types you don't actually collect

✅ **Do:**
- Be conservative - only report what's actually collected
- Mark everything that's shared with Firebase/Sentry
- Keep responses consistent with privacy policy

---

*Use this guide while filling out the Data Safety form in Play Console*
