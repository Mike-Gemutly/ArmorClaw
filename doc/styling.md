Here is the clean, developer-focused step-by-step plan to implement the unified ArmorClaw visual theme across **ArmorClaw** and **ArmorTerminal**.

**Goal**  
Make both apps visually feel like they belong to the same family as Component Catch (teal accents #14F0C8 on navy #0A1428 background, Inter + JetBrains Mono typography, subtle crab mascot in high-signal locations only, dark mode by default).

**Constraint**  
Dark mode is the only default experience. Light mode is optional and hidden behind a build flag.

### Execution Steps (Developer Checklist)

#### Phase 0 – Preparation (1 day)

1. Create new Kotlin Multiplatform module at root level  
   Folder: `:armorclaw-ui`

2. Set up basic `build.gradle.kts` for the new module

   ```kotlin
   plugins {
       kotlin("multiplatform")
       id("org.jetbrains.compose")
   }

   kotlin {
       androidTarget()
       // ios()               // uncomment later when needed
       // jvm("desktop")      // uncomment later when needed

       sourceSets {
           val commonMain by getting {
               dependencies {
                   implementation(compose.runtime)
                   implementation(compose.foundation)
                   implementation(compose.material3)
                   implementation(compose.ui)
                   implementation(compose.uiToolingPreview)
               }
           }
       }
   }

   android {
       compileSdk = 35
       sourceSets["main"].manifest.srcFile("src/androidMain/AndroidManifest.xml")
       defaultConfig {
           minSdk = 26
       }
   }
   ```

3. Add module to root `settings.gradle.kts`

   ```kotlin
   include(":armorclaw-ui")
   ```

4. Sync Gradle

#### Phase 1 – Define shared theme (1–2 days)

5. Create package & files

   ```
   armorclaw-ui/src/commonMain/kotlin/com/armorclaw/ui/theme/
   ├── ArmorClawColor.kt
   ├── ArmorClawTypography.kt
   ├── ArmorClawShapes.kt
   └── ArmorClawTheme.kt
   ```

6. Define colors (use exact Component Catch values)

   ```kotlin
   // ArmorClawColor.kt
   import androidx.compose.ui.graphics.Color

   val Teal       = Color(0xFF14F0C8)
   val TealGlow   = Color(0xFF67F5D8)
   val Navy       = Color(0xFF0A1428)
   val PrecisionBlue = Color(0xFF0EA5E9)
   val SuccessGreen  = Color(0xFF22C55E)
   val WarningAmber  = Color(0xFFF59E0B)

   val ArmorClawDarkColorScheme = darkColorScheme(
       primary = Teal,
       onPrimary = Navy,
       primaryContainer = TealGlow,
       inversePrimary = Teal,
       secondary = PrecisionBlue,
       onSecondary = Color.White,
       tertiary = SuccessGreen,
       error = Color(0xFFEF4444),
       background = Navy,
       onBackground = Color(0xFFE2E8F0),
       surface = Color(0xFF111827),
       onSurface = Color(0xFFE2E8F0),
       surfaceVariant = Color(0xFF1E293B),
       onSurfaceVariant = Color(0xFFCBD5E1)
   )
   ```

7. Define typography

   ```kotlin
   // ArmorClawTypography.kt
   import androidx.compose.ui.text.TextStyle
   import androidx.compose.ui.text.font.FontFamily
   import androidx.compose.ui.text.font.FontWeight
   import androidx.compose.ui.unit.sp

   val Inter = FontFamily(/* load fonts via compose multiplatform font loader or expect/actual */)
   val JetBrainsMono = FontFamily(/* same */)

   val ArmorClawTypography = Typography(
       displayLarge = TextStyle(fontFamily = Inter, fontWeight = FontWeight.Bold, fontSize = 57.sp),
       headlineLarge = TextStyle(fontFamily = Inter, fontWeight = FontWeight.Bold, fontSize = 32.sp),
       titleLarge    = TextStyle(fontFamily = Inter, fontWeight = FontWeight.SemiBold, fontSize = 22.sp),
       bodyLarge     = TextStyle(fontFamily = Inter, fontSize = 16.sp),
       bodyMedium    = TextStyle(fontFamily = Inter, fontSize = 14.sp),
       labelMedium   = TextStyle(fontFamily = Inter, fontSize = 12.sp),
       // Monospace for terminals & code
       labelSmall    = TextStyle(fontFamily = JetBrainsMono, fontSize = 12.sp)
   )
   ```

8. Define shapes (add subtle crab-shell inspired radius if designer provides custom path)

   ```kotlin
   // ArmorClawShapes.kt
   val ArmorClawShapes = Shapes(
       small  = RoundedCornerShape(8.dp),
       medium = RoundedCornerShape(12.dp),
       large  = RoundedCornerShape(16.dp)
   )
   ```

9. Create theme wrapper

   ```kotlin
   // ArmorClawTheme.kt
   @Composable
   fun ArmorClawTheme(content: @Composable () -> Unit) {
       MaterialTheme(
           colorScheme = ArmorClawDarkColorScheme,
           typography = ArmorClawTypography,
           shapes = ArmorClawShapes,
           content = content
       )
   }
   ```

#### Phase 2 – Integrate into both apps (2–4 days)

10. In **ArmorClaw** and **ArmorTerminal** replace root theme usages

    Before:
    ```kotlin
    MaterialTheme(...) { ... }
    ```

    After:
    ```kotlin
    ArmorClawTheme {
        // existing content
    }
    ```

11. Update `MainActivity` / entry composable in both apps

12. Replace hardcoded colors with semantic tokens where possible  
    Examples:
    - `backgroundColor = MaterialTheme.colorScheme.background`
    - `contentColor = MaterialTheme.colorScheme.onBackground`
    - `accent buttons` → `MaterialTheme.colorScheme.primary`

13. Add extension for teal glow (optional but high-impact)

    ```kotlin
    fun Modifier.glowTeal() = this
        .shadow(8.dp, shape = CircleShape, ambientColor = TealGlow, spotColor = TealGlow)
        .background(Teal.copy(alpha = 0.08f), CircleShape)
    ```

#### Phase 3 – Mascot placement (1–2 days)

14. Add vector asset `ic_crab.xml` (use existing Component Catch crab asset, 48–96 dp variants)

15. Place crab **only** in these high-signal locations (do **not** put in chat bubbles or message lists):

    - Splash screen
    - First-time onboarding / welcome screen
    - Empty state (no chats / no agents)
    - Settings → About screen (small version)
    - Premium / advanced feature unlocked success screen

16. Keep usage sparse – max 1–2 crabs visible at once

#### Phase 4 – Validation & Polish (2 days)

17. Run side-by-side comparison (emulator + browser with Component Catch extension)

18. Take screenshots of key flows in both apps

19. Check:
    - Contrast ratio ≥ 4.5:1 (use Accessibility Scanner)
    - TalkBack reading order correct
    - Cold start time not increased > 100 ms
    - No visual jump / flash on theme application

20. Create 3–5 snapshot tests (Papparazzi / Roborazzi) for:
    - Onboarding screen
    - Chat/terminal empty state
    - Settings screen
    - Premium success modal

#### Phase 5 – Final Checks & Merge

21. Add `:armorclaw-ui` dependency to both app modules

22. Verify no new lint / detekt issues

23. Build & run both apps in dark mode

24. (Optional) Add build flag for light mode experimentation

    ```kotlin
    // local.properties or gradle.properties
    enableLightMode=false
    ```

    ```kotlin
    val enableLightMode by extra(false)
    ```

25. Create PR titled:  
    `[Theme] Introduce shared armorclaw-ui module & unified branding`

26. Request review from at least one other Android engineer + UI/UX responsible person

**Estimated total effort:** 8–12 working days (1–2 weeks) with 2–3 engineers parallelizing integration.

Start with **Phase 0 today**.  
Once the shared module exists and compiles, split Phase 2 work between ArmorClaw and ArmorTerminal teams.