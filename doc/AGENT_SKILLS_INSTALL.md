# Oz Agent Skills — Developer Setup

Quick installation instructions for using Oz agent skills in your `mobilecli` development workflow.

## Prerequisites
- **Warp terminal** with Oz agent enabled
- **Go 1.19+** (for building mobilecli from source)
- **adb** in PATH (for Android device support)
- **ImageMagick** (optional, required only for `asset-genie` skill)

## Installation

### 1. Install as Dev Dependency (Recommended)
The agent skills are already bundled in this repository under `.agents/skills/`. Warp's Oz agent automatically discovers and uses them.

No additional installation needed — just ensure you're running Warp in this project directory.

### 2. Verify Skills are Available
From the Warp terminal in this project root, ask Oz:
```
@oz list available skills
```

You should see:
- `codebase-map` — Generate comprehensive codebase documentation
- `ux-review` — UI screen inventory and screenshot capture
- `adb-sleuth` — Logcat debugging and crash analysis
- `asset-genie` — App icon and asset generation
- `a11y-audit` — Accessibility compliance auditing
- `play-store-prep` — Play Store submission preparation
- `design-bridge` — Design-to-code component mapping

### 3. Optional: Install ImageMagick (for `asset-genie`)
**Windows (winget):**
```powershell
winget install ImageMagick.ImageMagick
```

**Windows (Chocolatey):**
```powershell
choco install imagemagick
```

**macOS:**
```bash
brew install imagemagick
```

**Linux:**
```bash
sudo apt install imagemagick  # Debian/Ubuntu
sudo dnf install ImageMagick  # Fedora
```

Verify installation:
```powershell
magick --version
```

## Usage Examples

### Map the Codebase (Recommended First Step)
```
@oz map the codebase
```

### Run UX Review
```
@oz run a UX review on this Android project
```

### Debug a Crash
```
@oz check logs for crashes on my connected device
```

### Generate App Icons
```
@oz generate app icons from source.png
```

### Audit Accessibility
```
@oz run an accessibility audit
```

### Prepare Play Store Assets
```
@oz prepare for Play Store submission
```

### Map Design to Code
```
@oz implement this design (attach screenshot or Figma export)
```

## Project Structure
```
.agents/
└── skills/
    ├── codebase-map/
    │   └── SKILL.md
    ├── ux-review/
    │   └── SKILL.md
    ├── adb-sleuth/
    │   └── SKILL.md
    ├── asset-genie/
    │   └── SKILL.md
    ├── a11y-audit/
    │   └── SKILL.md
    ├── play-store-prep/
    │   └── SKILL.md
    └── design-bridge/
        └── SKILL.md
```

## Output Directories
Skills generate deliverables in the project root:
- `CodebaseMap/` — LLM-readable codebase documentation (used by other skills)
- `UXReview/` — UI inventory and screenshots
- `AdbSleuth/` — Debug reports and sanity checks
- `AssetGenie/` — Generated icons and assets
- `A11yAudit/` — Accessibility audit reports
- `PlayStorePrep/` — Store listing and mockup prompts
- `DesignBridge/` — Component mapping and implementation code

## Tips
- **Run `@oz map the codebase` first** — generates `CodebaseMap/CODEBASE_MAP.md` that other skills use for context
- Skills work best when run from the project root directory
- For `adb-sleuth`, ensure at least one Android device is connected
- For `ux-review` and `play-store-prep`, screenshots require a running app on a connected device
- The `design-bridge` skill benefits from having a `component-catch` MCP server configured

## Troubleshooting

### Skill not found
Ensure you're running Warp from the project root (`E:\Micha\.LocalCode\mobilecli`). Skills are discovered via the `.agents/skills/` directory.

### ImageMagick errors (asset-genie)
On Windows, ensure `magick` (not `convert`) is in your PATH. Restart Warp after installation.

### adb not found (adb-sleuth, ux-review)
Install Android SDK platform-tools:
- Windows: `winget install Google.PlatformTools`
- macOS: `brew install --cask android-platform-tools`
- Linux: `sudo apt install adb`

### No devices connected
Connect an Android device via USB or start an emulator:
```bash
mobilecli devices --include-offline
mobilecli device boot --device <device-id>
```

## Learn More
- [Warp Oz Agent Documentation](https://docs.warp.dev/features/ai/oz)
- [mobilecli README](../README.md)
- [Lessons Learned](./LESSONS_LEARNED.md)
