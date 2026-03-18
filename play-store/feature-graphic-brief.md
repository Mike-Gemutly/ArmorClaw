# Feature Graphic Design Brief

> ArmorChat - Google Play Store Feature Graphic
> Dimensions: 1024 × 500 pixels
> Format: PNG (24-bit, no alpha preferred) or JPEG

---

## 1. Specifications

| Property | Requirement |
|----------|-------------|
| **Width** | 1024 px (exact) |
| **Height** | 500 px (exact) |
| **Aspect Ratio** | ~2.05:1 |
| **Format** | PNG or JPEG |
| **Color Depth** | 24-bit (no transparency) |
| **File Size** | Keep under 1MB for fast loading |
| **Safe Zone** | Keep critical content 80px from edges |

---

## 2. Brand Colors

### Primary Palette

| Color | Hex | RGB | Usage |
|-------|-----|-----|-------|
| **Deep Navy** | `#1A237E` | 26, 35, 126 | Background base |
| **Electric Blue** | `#304FFE` | 48, 79, 254 | Accent, highlights |
| **Teal Accent** | `#00BFA5` | 0, 191, 165 | Security icons |
| **Pure White** | `#FFFFFF` | 255, 255, 255 | Text, icons |

### Gradient (Background)

```
Start: #1A237E (top-left)
End:   #0D1442 (bottom-right)
Type:  Linear, 135° angle
```

### Alternative Dark Mode Palette

| Color | Hex | Usage |
|-------|-----|-------|
| **Surface Dark** | `#121212` | Background |
| **Primary** | `#BB86FC` | Accent |
| **Secondary** | `#03DAC6` | Security icons |

---

## 3. Layout Structure

### Grid System

```
┌─────────────────────────────────────────────────────────────────────┐
│                        1024px width                                  │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ 80px safe margin                                             │ 80px│
│  │                                                               │    │
│  │    ┌──────┐                                                  │    │
│  │    │ APP  │    ArmorChat                                    │    │
│  │    │ICON  │    Secure Team Chat                             │    │
│  │    │128px │                                                  │    │
│  │    └──────┘    ┌─────────────────────────────────┐          │    │
│  │                 │ 🔒 E2E  │  🛡️ Zero-Knowledge  │  🔐 AES-256 │
│  │                 └─────────────────────────────────┘          │    │
│  │                                                               │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                        500px height                                  │
└─────────────────────────────────────────────────────────────────────┘
```

### Zones

| Zone | X Position | Y Position | Content |
|------|------------|------------|---------|
| **Logo Zone** | 80-240px | Centered vertically | App icon + name |
| **Title Zone** | 260-600px | 140-220px | App name + tagline |
| **Features Zone** | 260-944px | 280-380px | Feature badges/icons |
| **Background** | Full | Full | Gradient + subtle pattern |

---

## 4. Content Elements

### A. App Icon (Left Side)

**Size:** 128 × 128 px (displayed)
**Style:** Rounded square with shadow
**Content:** Shield + chat bubble combination

**If no icon exists, create placeholder:**
- Rounded square (28% corner radius)
- Shield icon centered
- Chat bubble overlay
- Gradient fill: #304FFE → #1A237E
- Drop shadow: 4px, 20% opacity, #000000

### B. App Name & Tagline (Center-Left)

**App Name:**
```
Font: Roboto Medium or Product Sans
Size: 72px
Color: #FFFFFF
Weight: 500 (Medium)
Tracking: -0.5%
```

**Tagline:**
```
Font: Roboto Regular
Size: 32px
Color: #FFFFFF with 80% opacity
Weight: 400 (Regular)
Tracking: 0%
```

**Text Content:**
```
ArmorChat
Secure Team Chat
```

### C. Feature Badges (Bottom Area)

**Layout:** Horizontal row of 3 badges

**Each Badge:**
```
┌─────────────────────┐
│   🔒    End-to-End  │
│  icon    Encryption │
└─────────────────────┘
```

**Badge Specifications:**
- Icon size: 32 × 32 px
- Font: Roboto Medium, 18px
- Background: Semi-transparent white (10% opacity)
- Border radius: 8px
- Padding: 12px horizontal, 8px vertical
- Spacing between badges: 24px

**Badge Content (3 options):**

| Badge 1 | Badge 2 | Badge 3 |
|---------|---------|---------|
| 🔒 E2E Encrypted | 🛡️ Zero-Knowledge | 🔐 AES-256 |
| OR | OR | OR |
| 🔒 E2E Encryption | 🛡️ Privacy First | 🔐 Secure |

### D. Background Elements

**Base Gradient:**
```
Linear gradient from top-left to bottom-right
#1A237E → #0D1442
```

**Optional Pattern Overlay:**
- Subtle geometric pattern (hexagons or circuit-like)
- Opacity: 3-5%
- Color: White (#FFFFFF)

**Optional Glow Effect:**
- Soft glow behind app icon
- Color: #304FFE
- Blur: 60px
- Opacity: 30%

---

## 5. Typography

### Font Stack
- **Primary:** Roboto (Google standard)
- **Alternative:** Product Sans, Inter, Poppins

### Text Hierarchy

| Element | Font | Size | Weight | Color |
|---------|------|------|--------|-------|
| App Name | Roboto | 72px | Medium (500) | #FFFFFF |
| Tagline | Roboto | 32px | Regular (400) | #FFFFFF 80% |
| Badge Text | Roboto | 18px | Medium (500) | #FFFFFF |
| Feature Labels | Roboto | 14px | Regular (400) | #FFFFFF 90% |

---

## 6. Design Variations

### Option A: Minimal & Clean
```
┌────────────────────────────────────────────────────────────────┐
│                                                                │
│    ┌────┐                                                      │
│    │ 🛡️ │    ArmorChat                                        │
│    └────┘    Secure Team Chat                                 │
│                                                                │
│              🔒 End-to-End Encrypted  │  🛡️ Zero-Knowledge   │
│                                                                │
└────────────────────────────────────────────────────────────────┘
Background: Solid #1A237E with subtle vignette
```

### Option B: Dynamic & Modern
```
┌────────────────────────────────────────────────────────────────┐
│  ◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇  │
│    ┌────┐     ArmorChat                              ◇        │
│    │ 🛡️ │     Secure Team Chat                               │
│    └────┘                                                      │
│              ✦ E2E  │  ✦ Zero-Knowledge  │  ✦ AES-256  ◇     │
│  ◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇◇  │
└────────────────────────────────────────────────────────────────┘
Background: Gradient + animated particle pattern feel
```

### Option C: Enterprise Focus
```
┌────────────────────────────────────────────────────────────────┐
│                                                                │
│    ┌────┐     ArmorChat                   ┌──────────────────┐│
│    │ 🛡️ │     Enterprise Security         │  ★★★★★          ││
│    └────┘     Encrypted Messaging         │  Trusted by Teams││
│              └──────────────────┘                                │
│    🔒 E2E Encrypted  │  🏢 Team Ready  │  🔐 AES-256         │
│                                                                │
└────────────────────────────────────────────────────────────────┘
Background: Dark professional blue with subtle texture
```

---

## 7. Iconography

### Security Icons (for badges)

**Lock Icon (🔒):**
```
<svg width="32" height="32" viewBox="0 0 24 24" fill="none">
  <rect x="3" y="11" width="18" height="11" rx="2" fill="#00BFA5"/>
  <path d="M7 11V7a5 5 0 0110 0v4" stroke="#00BFA5" stroke-width="2"/>
</svg>
```

**Shield Icon (🛡️):**
```
<svg width="32" height="32" viewBox="0 0 24 24" fill="none">
  <path d="M12 2L3 7v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V7l-9-5z"
        fill="#00BFA5"/>
</svg>
```

**Key Icon (🔐):**
```
<svg width="32" height="32" viewBox="0 0 24 24" fill="none">
  <circle cx="8" cy="8" r="4" stroke="#00BFA5" stroke-width="2" fill="none"/>
  <path d="M10.5 10.5L21 21M15 15l3-3" stroke="#00BFA5" stroke-width="2"/>
</svg>
```

### Icon Colors
- Primary icons: `#00BFA5` (Teal Accent)
- Alternative: `#FFFFFF` with 90% opacity

---

## 8. Export Settings

### For Figma / Sketch / Adobe XD

**Canvas:** 1024 × 500 px
**Export format:** PNG
**Scale:** 1x (exact size)
**Compression:** Medium (balance quality/size)

### For Photoshop

```
File → Export → Export As
Format: PNG
Size: 1024 × 500 px (100%)
Quality: 80-90%
Metadata: None
Color space: sRGB
```

### For Illustrator

```
File → Export → Export for Screens
Format: PNG
Scale: 1x
Artboard: 1024 × 500 px
Anti-aliasing: Type Optimized
```

---

## 9. Quality Checklist

Before export, verify:

- [ ] Canvas is exactly 1024 × 500 px
- [ ] No text smaller than 14px
- [ ] Critical elements within 80px safe zone
- [ ] Text is readable at small sizes (test at 50%)
- [ ] Colors are within brand palette
- [ ] No transparency/alpha channel
- [ ] File size under 1MB
- [ ] Tested on dark and light Play Store themes
- [ ] No copyrighted imagery
- [ ] No device-specific frames/bezels

---

## 10. Design Tools

### Recommended Tools (Free)

| Tool | URL | Best For |
|------|-----|----------|
| **Figma** | figma.com | Collaborative design |
| **Canva** | canva.com | Quick templates |
| **Photopea** | photopea.com | Photoshop alternative |

### Figma Template Prompt

If using Figma with AI, try:
```
"Create a Google Play feature graphic (1024x500px) for an
encrypted messaging app called ArmorChat. Use deep navy blue
gradient background (#1A237E to #0D1442), white text, and teal
accent icons. Include: app icon on left, 'ArmorChat' title in
72px, 'Secure Team Chat' tagline, and 3 feature badges showing
E2E encryption, Zero-Knowledge, and AES-256. Modern, minimal,
enterprise-focused design."
```

---

## 11. Example Mockup (ASCII)

```
╔══════════════════════════════════════════════════════════════════════════╗
║                                                                          ║
║     ┌─────────┐                                                         ║
║     │         │                                                         ║
║     │   🛡️    │     ArmorChat                                          ║
║     │         │     Secure Team Chat                                   ║
║     └─────────┘                                                         ║
║       128px                                                             ║
║                                                                         ║
║                   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    ║
║                   │ 🔒  E2E     │  │ 🛡️  Zero-   │  │ 🔐  AES-256 │    ║
║                   │   Encrypted │  │   Knowledge │  │   Security  │    ║
║                   └─────────────┘  └─────────────┘  └─────────────┘    ║
║                                                                         ║
╚══════════════════════════════════════════════════════════════════════════╝

Background: Linear gradient #1A237E (left) → #0D1442 (right)
Text: White (#FFFFFF)
Icons: Teal (#00BFA5)
Badges: White 10% opacity background
```

---

## 12. Output File

**Save as:** `feature-graphic.png`
**Location:** `play-store/listing/feature-graphic.png`
**Upload to:** Play Console → Main Store Listing → Graphic assets

---

*Design brief for ArmorChat Play Store feature graphic*
