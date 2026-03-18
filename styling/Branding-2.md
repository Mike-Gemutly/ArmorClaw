CPO Review: Branding Kit for Component Catch  
Version: v2.1.1 – February 2026  
Focus: Revenue Leverage, Differentiation, Margin Expansion

***

## Executive Summary

ArmorClaw (the turquoise hermit crab) is a **strategic** asset, not a decorative mascot.  
It encodes three monetizable positioning pillars:

- “Catch” – the claw literally grabs and inspects components, mirroring the core product function.  
- “Hermit” – the reusable shell maps to design systems, tokens, and code portability across stacks.  
- “Playful but precise” – distinctive in a crowded, hyper-technical AI tooling market where most brands feel cold and interchangeable.

This system can:

- Increase memorability with AI-native developer communities, improving top-of-funnel efficiency and referral lift.  
- Reduce perceived risk when upselling to premium, because the brand feels established, intentional, and cohesive.  
- Deepen emotional affinity and “tool loyalty,” supporting retention and expansion once teams adopt it.

However:

- Brand will not independently deliver a 35–50% conversion lift.  
- Conversion and ARR are still driven by:
  - Clear performance delta (Node vs Go, Go vs Rust).  
  - Premium-only capabilities (optimized extraction, Rust acceleration, agent-native workflows).  
  - Reduced activation friction (MCP setup flow, templates, defaults).

ArmorClaw amplifies these drivers; it does not replace them.  
Verdict: Keep and formalize ArmorClaw as a core brand system. Use it explicitly to differentiate premium experiences and storytelling, while keeping the underlying product value proposition performance-first.

***

## 1. Logo Hierarchy & Governance

We have three useful variants; the missing piece is firm governance and usage rules.

### 1.1 Primary Logo (System Default)

Asset: PrecisionMark2 (circular crab shell with face)  
Use as the default identity everywhere a small, compact mark is required:

- Browser and OS surfaces: Chrome Web Store icon, extension icon, favicon.  
- Developer ecosystems: GitHub org/avatar, MCP server identity visuals, CLI or server splash if any.  
- Product UI: top-left app badge, modal headers, documentation footer mark.

Why this is the canonical mark:

- Clean, high-contrast silhouette that survives at 16px.  
- Shell is dominant and recognizable even when details are lost.  
- Eyes add personality without introducing extra complexity or tiny elements.

Governance rule: if there is only room for one mark, use this one.

### 1.2 Secondary Logo (Hero / Marketing)

Asset: ArmorClaw-Guardians-2 (circular frame, full crab with claw)  
Use in contexts where storytelling and emotion matter:

- Home/landing page hero and above-the-fold sections.  
- Pricing page, especially near premium callouts.  
- Upgrade flows and in-app “aha” moments (empty states, success banners).  
- Campaign assets: launch emails, partner pages, and case-study covers.

Why it works:

- Full claw strongly reinforces “Catch” and the idea of grabbing components.  
- More expressive and dynamic, better suited for narrative and premium-feel visuals.

Governance rule: use this where you have space and need personality, but keep the primary mark as the anchor in navigation and small UI.

### 1.3 Tertiary Logo (Platform Containers)

Asset: ArmorClaw-Guardians-3 (rounded square with full crab)  
Use where a square or tile format is mandatory:

- Chrome Web Store listing tile and promo screenshots.  
- App directories (Cursor, Claude Desktop, Gemini, Windsurf, etc.).  
- Social preview cards (OpenGraph/Twitter images) and badges.  
- Notion and documentation embeds where square thumbnails perform better.

Governance rule: this is not a “fun extra”; it is the required shape for stores and directories. Keep it tightly aligned with the primary/secondary color and line-weight system.

### 1.4 Variation Rules (What to Avoid)

Do NOT:

- Create multiple color schemes or brand “skins.”  
- Introduce seasonal, meme, or holiday variants into product surfaces.  
- Add 3D, gradients-on-gradients, or skeuomorphic styles.  
- Allow teams to improvise drop shadows, strokes, or outlines.

If we need limited variations, they must be documented explicitly (e.g., monochrome white for dark video overlays, monochrome teal for print) and added to a simple brand spec (one-pager) so they don’t proliferate.

***

## 2. Color System (Refined & Commercially Aligned)

The color system should support readability, developer trust, and premium differentiation.

### 2.1 Core Brand Palette

| Role           | Color   | Hex      | Purpose                                                                 |
|----------------|---------|----------|-------------------------------------------------------------------------|
| Primary Teal   | Teal    | #14F0C8 | Brand anchor, primary CTAs, key highlights in product and marketing.   |
| Dark Background| Navy    | #0A1428 | Default background, dark-mode-first dev aesthetic.                      |
| Accent Glow    | Light   | #67F5D8 | Hover states, subtle glows, borders for premium emphasis and focus.    |

These must be consistent across:

- Chrome extension UI.  
- Landing and marketing pages.  
- Docs and MCP server-related assets.

Governance rule: if a designer introduces a new “primary accent,” it must go through review; the teal is non-negotiable for brand recognition.

### 2.2 Extended Functional Palette

| Role              | Color | Hex      | Purpose                                                                 |
|-------------------|-------|----------|-------------------------------------------------------------------------|
| Premium Success   | Green | #22C55E | Upgrade confirmations, successful premium flows, and upsell moments.   |
| AI Precision      | Blue  | #0EA5E9 | Technical diagrams, “precision” callouts, and agent-native visuals.     |
| Warning           | Amber | #F59E0B | Guardrails, rate limits, edge-case alerts.                             |
| Neutral Dark      | Gray  | #1E2937 | Panels, headers, and layered UI states.                                |
| Neutral Light     | Gray  | #F8FAFC | Documentation backgrounds, marketing sections, print or slide use.      |

Premium differentiation tactic:

- Use slightly stronger contrast and glow for premium: more Accent Glow on switches and CTAs, combined with Premium Success green in confirmations.  
- Keep free tier slightly flatter and more utilitarian to highlight the “upgrade in feel” without deviating from the core palette.

***

## 3. Typography (Signal of Craft, Not a Distraction)

Typography should signal craft and modernity without adding bundle size or visual noise.

- Display headings (marketing, hero): Inter Bold or Space Grotesk (for subtle character, if we accept an extra font).  
- Product UI/body copy: Inter (already widely adopted, performant, and familiar to devs).  
- Code and technical snippets: JetBrains Mono for a clear, dev-native look.

Rules:

- Avoid adding more display fonts or playful scripts.  
- Do not mix multiple monospace fonts across surfaces.  
- Document line-height and letter-spacing defaults for headings and body so screens look consistent across marketing and app.

***

## 4. Messaging & Tagline Strategy

We need one core narrative with a few controlled variations for context.

### 4.1 Primary Tagline (Landing + Chrome Store)

“Catch. Inspect. Ship.”

- Verb-first, reflecting the workflow from element to production-ready code.  
- Short enough to live in Chrome Store titles, hero banners, and stickers.  
- Easy to reuse in spoken pitches and social posts.

Optional supporting line under hero:

“The AI-native inspector that lets your agents grab and modify real components.”

### 4.2 Premium-Specific Messaging

Example:  
“Power your AI agents with a faster claw.”

Use in:

- Premium feature pages and pricing sections.  
- MCP-focused sales collateral and partner docs.  
- Upgrade modals in app.

It should explicitly connect:

- Go/Rust performance gains.  
- MCP capabilities.  
- “Agent-native” positioning (this is a key differentiator vs generic devtools).

### 4.3 Technical Positioning (Docs & Repos)

Core line:  
“Universal MCP-powered component inspection with live modification.”

Use in:

- GitHub READMEs.  
- Documentation intros.  
- Developer partner decks.

Support with a clear one-paragraph explanation emphasizing:

- Multi-agent support (Cursor, Claude, Windsurf, Gemini).  
- Inspection and live modification in real browser contexts.  
- Extensibility for design systems and tokens.

***

## 5. Monetization Alignment

Brand must support conversion, expansion, and pricing power.

### 5.1 Free Tier (Acquisition Engine)

Free tier brand characteristics:

- Same core mascot and colors as premium to minimize friction and build trust.  
- Static crab icon across extension UI, Chrome store, and docs.  
- Functional and “no surprises” tone.

Free tier experiences:

- Core inspection tools and MCP basics.  
- Node.js server or baseline performance tier.  
- Minimal friction signup and setup flows.

Goal: maximize installation, retention through first real use, and MCP configuration completion.

### 5.2 Premium Tier (Margin and ARPU Expansion)

Premium should feel like “the same tool, just sharper and more alive.”

Visual and motion upgrades:

- Subtle animations in premium success modals (confetti + small claw pinch).  
- Micro-animation during optimized extraction or Go/Rust-powered operations (e.g., claw squeezing and releasing).  
- Slight, consistent glow treatment on premium toggles and advanced panels.

Rules:

- Do not hide the mascot from free; identity must stay shared to avoid confusion.  
- Do not create an entirely separate visual brand for premium (“Pro” color scheme).  
- Make sure visual delight correlates with real feature and performance gains, not just cosmetic gating.

***

## 6. What Actually Drives ARR

Brand is a multiplier on product-market fit, not the engine.

Primary ARR drivers:

- Performance advantage: 10–250x faster Go server, Rust-backed extraction, and overall latency reduction in AI workflows.  
- Premium-only capabilities: optimized extraction flows, advanced selectors, live modifications at scale, auto-generated design tokens.  
- Workflow advantage: watch mode, agent-native integration, and tight loops with Cursor, Claude, Windsurf, and Gemini.

Brand contribution:

- Increases trust that the tool is serious and maintained (reduces churn from “toy” perception).  
- Improves recall so devs actually re-open and recommend the extension.  
- Makes conference and community activities more effective per dollar spent.

When reviewing funnel metrics, attribute improvements conservatively to brand unless we can clearly isolate a brand-only experiment (e.g., upgraded icons on Chrome store vs control).

***

## 7. High-ROI Brand Activations

Prioritized by commercial impact, not aesthetics.

1. Chrome Web Store  
   - Tertiary square icon as listing icon.  
   - Hero screenshot: crab visually “catching” a real UI element or component overlay.  
   - Copy: headline “Catch. Inspect. Ship.” and a short subline about AI-native inspection.  
   - Goal: maximize install rate and convey “modern, trustworthy, AI-native” in one glance.

2. Landing Page  
   - Secondary hero logo, lightly animated claw on scroll or hover.  
   - Immediate performance callout: “Up to 250x faster MCP server for your AI coding agents.”  
   - Clear pricing and premium differentiation above the fold.

3. Premium Success Modal  
   - Visual highlight of the upgrade moment: animation, premium green + glow, and a single, confident line about what just unlocked.  
   - Example headline: “Your claw just got sharper.” with a direct link to a key premium feature.

4. Conference and Community Assets (Q2+)  
   - Sticker sets with the primary, secondary, and a small number of pose variants (full crab, claw-only, shell-only).  
   - T-shirts or laptop decals for early adopters / “Crab Club.”  
   - QR code landing to a “Agent-native inspection” explainer, not a generic homepage.

***

## 8. Guardrails: What Not to Build

To protect focus and avoid brand dilution:

- No 3D or hyper-rendered mascot versions for core product surfaces.  
- No animation-heavy landing pages that slow performance or feel gimmicky.  
- No additional animal characters or “mascot universe.”  
- No light-mode-first rebrand; maintain dark-mode default with acceptable light-mode fallbacks only where needed (docs/print).

If we ever explore 3D or playful variants, limit them to specific campaigns or physical merch—never into product UI by default.

***

## 9. ICP & Positioning Fit

Where the brand is a force multiplier:

- Tier 1 ICP (high LTV): AI-native frontend engineers and teams actively using Cursor, Claude Desktop, Windsurf, Gemini, etc.  
  - They care about tools that feel “agent-native,” polished, and opinionated.  
- Tier 2 ICP: design-system consultants, agencies, and frontend leads working on complex design systems.  
  - They instinctively understand the “hermit shell = reusable system” metaphor.

Where the brand matters less (but still shouldn’t clash):

- Enterprise compliance, procurement, and security buyers.  
- Non-technical PMs who optimize for risk and cost over aesthetics.

Tactical implication: let dev-facing materials lean into the mascot and playful-precise tone; keep enterprise and procurement assets cleaner, more conservative, and performance/security-first with only light brand presence.

***

## 10. Roadmap Alignment (CPO Recommendations)

### Immediate (This Sprint)

- Finalize and export canonical SVGs for all three logo variants with size presets (16, 32, 64, 128, 512).  
- Replace existing ad-hoc icons across extension, website, and GitHub with the approved primary mark.  
- Refresh Chrome Web Store listing with correct tertiary logo and aligned screenshots.

### Next Sprint

- Implement premium micro-animations (loading + success) tied to actual premium flows.  
- Introduce subtle premium glow states for advanced panels and toggles.  
- Add a small but clear “Powered by ArmorClaw” or similar label in MCP docs to unify the narrative.

### Q2

- Produce sticker pack and a simple physical brand kit for conferences.  
- Launch a “Crab Club” early access or advocate program if traction confirms repeat users and referability.  
- Add a short “Brand + Use” section to internal docs so contributors know what to do without guessing.
