# Privacy Policy Hosting Instructions

> **App:** ArmorTerminal
> **Last Updated:** 2026-03-17

---

## Quick Setup

### Option 1: GitHub Pages (Free, Recommended)

1. Create repository: `armorclaw/privacy`
2. Upload `PRIVACY_POLICY.md` as `index.md`
3. Enable GitHub Pages in repository settings
4. URL: `https://armorclaw.github.io/privacy/`

### Option 2: Netlify (Free)

1. Connect repository to Netlify
2. Deploy as static site
3. URL: `https://armorclaw-privacy.netlify.app/`

### Option 3: Custom Domain
1. Add DNS record: `privacy.armorclaw.app` → GitHub Pages IP
2. URL: `https://privacy.armorclaw.app/`

### Option 4: Vercel (Free)
1. Import project to Vercel
2. Deploy as static site
3. URL: `https://armorclaw-privacy.vercel.app/`

---

## Content to Host

Upload the file at `applications/ArmorTerminal/PRIVACY_POLICY.md`:

### If Privacy Policy doesn't exist:
Use the template from `applications/ArmorChat/PRIVACY_POLICY.md` as reference.

---

## Play Console Setup

1. Go to **App content** → **Privacy policy**
2. Enter URL: `https://armorclaw.github.io/privacy/` (or your chosen URL)
3. Click **Save**

---

## Data Safety Form

After privacy policy is hosted:

1. Go to **App content** → **Data safety**
2. Answer the questions based on your privacy policy content:
   - **Data collected:** Email, Device ID
   - **Data encrypted:** In transit (TLS 1.3), At rest (AES-256)
   - **Data sharing:** No third parties
   - **Data deletion:** Available via app reset

---

## Verification

After hosting, verify:
1. URL is accessible via HTTPS
2. Privacy policy displays correctly
3. All links work
4. Content rating questionnaire can reference the URL

---

## Estimated Time
- **GitHub Pages:** 5 minutes
- **Netlify/Vercel:** 10 minutes
- **Custom domain:** 30 minutes (DNS propagation)

---

## Next Steps
1. Choose hosting option
2. Upload privacy policy
3. Add URL to Play Console
4. Complete Data Safety form
