# ArmorClaw Developer Guide

> Developer guide for ArmorClaw - Secure E2E Encrypted Chat Application

## 📋 Prerequisites

### Required Tools

- **Android Studio** Hedgehog (2023.1.1) or later
- **JDK** 17
- **Android SDK** 34
- **Gradle** 8.2
- **Git** - For version control

### Required Knowledge

- **Kotlin** - Language (KMP)
- **Jetpack Compose** - UI framework
- **Clean Architecture** - Architecture pattern
- **MVVM** - Design pattern
- **Room** - Database
- **Ktor** - Networking
- **Coroutines** - Asynchronous programming

## 🔨 Building the Project

### Clone Repository

```bash
git clone https://github.com/armorclaw/ArmorClaw.git
cd ArmorClaw
```

### Open in Android Studio

1. **Open Android Studio**
2. **File → Open** and select `ArmorClaw` directory
3. **Wait** for Gradle sync to complete

### Build Project

```bash
# Clean build
./gradlew clean

# Build debug
./gradlew assembleDebug

# Build release
./gradlew assembleRelease

# Install debug APK
./gradlew installDebug
```

### Run Tests

```bash
# Run unit tests
./gradlew test

# Run instrumented tests
./gradlew connectedAndroidTest

# Run specific test
./gradlew test --tests "com.armorclaw.app.*"
```

---

## 🏗️ Project Structure

### Module Overview

```
ArmorClaw/
├── shared/                    # KMP shared module
│   ├── domain/               # Domain layer
│   ├── platform/             # Platform interfaces
│   └── ui/                  # Shared UI components
└── androidApp/              # Android application
    ├── screens/              # Compose screens
    ├── data/                 # Data layer
    ├── platform/             # Platform implementations
    └── release/              # Release configuration
```

### Shared Module (`shared/`)

**Domain Layer (`domain/`)**
- `model/` - Domain models (Message, Room, User)
- `repository/` - Repository interfaces
- `usecase/` - Use case interfaces

**Platform Layer (`platform/`)**
- `BiometricAuth.kt` - Expect interface
- `SecureClipboard.kt` - Expect interface
- `NotificationManager.kt` - Expect interface
- `NetworkMonitor.kt` - Expect interface

**UI Layer (`ui/`)**
- `theme/` - Design system (Theme, Colors, Typography, Shapes)
- `components/atom/` - Atomic components (Button, InputField, Card)
- `components/molecule/` - Molecular components
- `base/` - Base classes (BaseViewModel, UiState, UiEvent)

### Android Module (`androidApp/`)

**Screens (`screens/`)**
- `onboarding/` - Onboarding flow screens
- `home/` - Home screen
- `chat/` - Chat screens
- `profile/` - Profile screen
- `settings/` - Settings screen
- `auth/` - Authentication screens
- `room/` - Room management screens
- `splash/` - Splash screen

**Data Layer (`data/`)**
- `persistence/` - DataStore persistence
- `database/` - Room database (SQLCipher)
- `offline/` - Offline sync

**Platform (`platform/`)**
- Platform implementations (BiometricAuth, SecureClipboard, etc.)

**Performance (`performance/`)**
- `PerformanceProfiler.kt` - Performance profiling
- `MemoryMonitor.kt` - Memory monitoring

**Accessibility (`accessibility/`)**
- `AccessibilityConfig.kt` - Accessibility configuration
- `AccessibilityExtensions.kt` - Compose accessibility modifiers

**Release (`release/`)**
- `ReleaseConfig.kt` - Release configuration

---

## 🎨 Design System

### Color Palette

```kotlin
object AppColors {
    val AccentColor = Color(0xFF6750A4)
    val PrimaryColor = Color(0xFF6200EE)
    val SecondaryColor = Color(0xFF03DAC6)
    val ErrorColor = Color(0xFFB00020)
    val BackgroundColor = Color(0xFFFFFBFE)
    val SurfaceColor = Color(0xFFFFFBFE)
}
```

### Typography

```kotlin
object AppTypography {
    val DisplayLarge = TextStyle(fontSize = 57.sp, fontWeight = FontWeight.Bold)
    val DisplayMedium = TextStyle(fontSize = 45.sp, fontWeight = FontWeight.Bold)
    val DisplaySmall = TextStyle(fontSize = 36.sp, fontWeight = FontWeight.Bold)
    val HeadlineLarge = TextStyle(fontSize = 32.sp, fontWeight = FontWeight.Bold)
    val HeadlineMedium = TextStyle(fontSize = 28.sp, fontWeight = FontWeight.Bold)
    val HeadlineSmall = TextStyle(fontSize = 24.sp, fontWeight = FontWeight.Bold)
    val TitleLarge = TextStyle(fontSize = 22.sp, fontWeight = FontWeight.SemiBold)
    val TitleMedium = TextStyle(fontSize = 16.sp, FontWeight = FontWeight.SemiBold)
    val TitleSmall = TextStyle(fontSize = 14.sp, FontWeight.SemiBold)
    val BodyLarge = TextStyle(fontSize = 16.sp, fontWeight = FontWeight.Normal)
    val BodyMedium = TextStyle(fontSize = 14.sp, fontWeight = FontWeight.Normal)
    val BodySmall = TextStyle(fontSize = 12.sp, fontWeight = FontWeight.Normal)
    val LabelLarge = TextStyle(fontSize = 14.sp, fontWeight = FontWeight.Medium)
    val LabelMedium = TextStyle(fontSize = 12.sp, fontWeight = FontWeight.Medium)
    val LabelSmall = TextStyle(fontSize = 11.sp, fontWeight = FontWeight.Medium)
}
```

### Shapes

```kotlin
object AppShapes {
    val ExtraSmall = RoundedCornerShape(4.dp)
    val Small = RoundedCornerShape(8.dp)
    val Medium = RoundedCornerShape(12.dp)
    val Large = RoundedCornerShape(16.dp)
    val ExtraLarge = RoundedCornerShape(28.dp)
    val Full = RoundedCornerShape(50)
}
```

---

## 🔌 Platform Integration

### Adding New Platform Feature

1. **Define expect interface** in `shared/platform/`:

```kotlin
// shared/platform/NewFeature.kt
expect class NewFeature {
    suspend fun doSomething(): Result
}
```

2. **Implement actual class** in `androidApp/platform/`:

```kotlin
// androidApp/platform/NewFeatureImpl.kt
actual class NewFeature {
    actual suspend fun doSomething(): Result {
        // Android-specific implementation
        return Result.Success
    }
}
```

3. **Inject** using Koin:

```kotlin
// Provide in Koin module
single<NewFeature> { NewFeature() }
```

4. **Use** in ViewModel:

```kotlin
class MyViewModel(
    private val newFeature: NewFeature
) : BaseViewModel<UiState, UiEvent>() {
    fun onDoSomething() {
        viewModelScope.launch {
            val result = newFeature.doSomething()
            // Handle result
        }
    }
}
```

---

## 🗄️ Database

### Adding New Entity

1. **Define entity** in `androidApp/data/database/`:

```kotlin
@Entity(
    tableName = "my_table",
    indices = [
        Index("field1"),
        Index("field2", "field3")
    ]
)
data class MyEntity(
    @PrimaryKey val id: String,
    val field1: String,
    val field2: Int,
    val field3: Boolean
)
```

2. **Define DAO** in `androidApp/data/database/`:

```kotlin
@Dao
interface MyDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(entity: MyEntity)

    @Update
    suspend fun update(entity: MyEntity)

    @Delete
    suspend fun delete(entity: MyEntity)

    @Query("SELECT * FROM my_table")
    fun getAll(): Flow<List<MyEntity>>

    @Query("SELECT * FROM my_table WHERE id = :id")
    suspend fun getById(id: String): MyEntity?
}
```

3. **Add to database**:

```kotlin
@Database(
    entities = [MessageEntity::class, RoomEntity::class, SyncQueueEntity::class, MyEntity::class],
    version = 2,
    exportSchema = true
)
abstract class AppDatabase : RoomDatabase() {
    abstract fun myDao(): MyDao
}
```

4. **Create migration** (if needed):

```kotlin
val MIGRATION_1_2 = object : Migration(1, 2) {
    override fun migrate(database: SupportSQLiteDatabase) {
        database.execSQL("CREATE TABLE IF NOT EXISTS my_table (id TEXT PRIMARY KEY NOT NULL, field1 TEXT NOT NULL, field2 INTEGER NOT NULL, field3 INTEGER NOT NULL)")
    }
}
```

---

## 🔄 Offline Sync

### Adding New Operation Type

1. **Add operation type enum**:

```kotlin
enum class OperationType(val value: String) {
    SEND_MESSAGE("send_message"),
    UPDATE_MESSAGE("update_message"),
    DELETE_MESSAGE("delete_message"),
    ADD_REACTION("add_reaction"),
    REMOVE_REACTION("remove_reaction"),
    MARK_READ("mark_read"),
    NEW_OPERATION("new_operation")
}
```

2. **Add enqueue method** to `OfflineQueue`:

```kotlin
suspend fun enqueueNewOperation(
    param1: String,
    param2: Int,
    priority: OperationPriority = OperationPriority.MEDIUM
): String {
    val operation = SyncQueueEntity(
        id = generateId(),
        roomId = param1,
        operationType = OperationType.NEW_OPERATION,
        priority = priority,
        status = "pending",
        // Add additional parameters as JSON
        data = """{"param1": "$param1", "param2": $param2}"""
    )
    syncQueueDao.enqueue(operation)
    return operation.id
}
```

3. **Add execute logic** to `SyncEngine`:

```kotlin
private suspend fun executeNewOperation(operation: SyncQueueEntity): Result {
    return try {
        // Parse data
        val data = Json.decodeFromString<NewOperationData>(operation.data)
        
        // Execute operation
        val result = repository.newOperation(data.param1, data.param2)
        
        // Return success
        Result.Success
    } catch (e: Exception) {
        // Return failure
        Result.Failure(e.message)
    }
}
```

---

## 🧪 Testing

### Writing Unit Test

```kotlin
class MyViewModelTest {
    
    @get:Rule
    val instantTaskExecutorRule = InstantTaskExecutorRule()
    
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()
    
    private lateinit var viewModel: MyViewModel
    private val repository: MyRepository = mockk()
    
    @Before
    fun setup() {
        viewModel = MyViewModel(repository)
    }
    
    @Test
    fun `onDoSomething emits success`() = runTest {
        // Given
        coEvery { repository.doSomething() } returns Result.Success
        
        // When
        viewModel.onDoSomething()
        
        // Then
        assertEquals(Result.Success, viewModel.uiState.value.result)
    }
}
```

### Writing Compose UI Test

```kotlin
class MyScreenTest {
    
    @get:Rule
    val composeTestRule = createComposeRule()
    
    @Test
    fun screen_displays_text() {
        // Given
        composeTestRule.setContent {
            MyScreen(text = "Hello, World!")
        }
        
        // Then
        composeTestRule.onNodeWithText("Hello, World!")
            .assertIsDisplayed()
    }
    
    @Test
    fun button_click_triggers_event() {
        // Given
        var clicked = false
        composeTestRule.setContent {
            MyScreen(
                onClick = { clicked = true }
            )
        }
        
        // When
        composeTestRule.onNodeWithText("Click me")
            .performClick()
        
        // Then
        assertTrue(clicked)
    }
}
```

---

## 🚀 Release

### Building Release APK

```bash
# Build release APK
./gradlew assembleRelease

# Find APK in androidApp/build/outputs/apk/release/
```

### Signing Release APK

1. **Generate keystore** (if needed):

```bash
keytool -genkey -v -keystore armorclaw.keystore -alias armorclaw -keyalg RSA -keysize 2048 -validity 10000
```

2. **Configure signing** in `androidApp/build.gradle.kts`:

```kotlin
android {
    signingConfigs {
        create("release") {
            storeFile = file("armorclaw.keystore")
            storePassword = System.getenv("KEYSTORE_PASSWORD")
            keyAlias = "armorclaw"
            keyPassword = System.getenv("KEY_PASSWORD")
        }
    }
    
    buildTypes {
        release {
            signingConfig = signingConfigs.getByName("release")
            isMinifyEnabled = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
        }
    }
}
```

### Publishing to Play Store

1. **Generate AAB** (Android App Bundle):

```bash
./gradlew bundleRelease
```

2. **Upload AAB** to Google Play Console

3. **Fill in store listing**:
   - App icons (ldpi, mdpi, hdpi, xhdpi, xxhdpi, xxxhdpi)
   - Feature graphic (1024x500)
   - Screenshots (phone, 7-inch, 10-inch)
   - Short description (80 chars)
   - Full description (4000 chars)
   - Privacy policy URL

4. **Submit for review**

---

## 📊 Performance

### Profiling

```kotlin
val profiler = PerformanceProfiler(enabled = true)

// Trace block
profiler.trace("sendMessage") {
    sendMessage(message)
}

// Track allocations
val result = profiler.trackAllocations("sendMessage") {
    sendMessage(message)
}

// Dump heap
profiler.dumpHeap(File("heap.hprof"))
```

### Memory Monitoring

```kotlin
val memoryMonitor = MemoryMonitor(context)

// Watch memory pressure
memoryMonitor.memoryPressure.collect { pressure ->
    when (pressure) {
        MemoryPressure.NORMAL -> println("Memory OK")
        MemoryPressure.HIGH -> println("Memory High")
        MemoryPressure.CRITICAL -> println("Memory Critical")
    }
}

// Get memory info
val memoryInfo = memoryMonitor.getCurrentMemoryInfo()
```

---

## 🔐 Security

### Encryption

```kotlin
// Encrypt message
val encrypted = encrypt(message, publicKey)

// Decrypt message
val decrypted = decrypt(encrypted, privateKey)
```

### Biometric Auth

```kotlin
val biometricAuth = getBiometricAuth()

// Authenticate
val result = biometricAuth.authenticate(
    title = "Unlock ArmorClaw",
    subtitle = "Authenticate to access your messages"
)

when (result) {
    is BiometricResult.Success -> println("Success")
    is BiometricResult.Failure -> println("Failure")
    is BiometricResult.Error -> println("Error: ${result.message}")
    is BiometricResult.Cancelled -> println("Cancelled")
}
```

---

## ❓ Getting Help

- **Documentation:** [doc/](doc/)
- **Issues:** [GitHub Issues](https://github.com/armorclaw/ArmorClaw/issues)
- **Discussions:** [GitHub Discussions](https://github.com/armorclaw/ArmorClaw/discussions)
- **Email:** dev@armorclaw.app

---

**Happy Coding!**

*ArmorClaw - Secure. Private. Encrypted.*
