# ArmorClaw Android App - Compose Multiplatform Implementation Plan

> **Document Purpose:** Implementation plan for Android app using Compose Multiplatform with maximum SwiftUI (iOS) reusability
> **Date Created:** 2026-02-10
> **Technology Stack:** Kotlin Multiplatform + Compose Multiplatform (CMP)
> **Goal:** 80%+ UI code shared between Android and iOS

---

## 1. Architecture Overview

### 1.1 Technology Stack

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Compose Multiplatform Stack                      │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │              shared module (KMP)                            │   │
│  │  ┌──────────────────────────────────────────────────────┐  │   │
│  │  │  commonMain (shared code)                           │  │   │
│  │  │  ├─ Design System (Theme, Colors, Typography)       │  │   │
│  │  │  ├─ UI Components (Atoms, Molecules, Organisms)     │  │   │
│  │  │  ├─ Screen Components (Onboarding, Chat, etc.)      │  │   │
│  │  │  ├─ Navigation (Route definitions, deep links)       │  │   │
│  │  │  ├─ State Management (ViewModels)                    │  │   │
│  │  │  └─ Business Logic (Use Cases, Repositories)        │  │   │
│  │  └──────────────────────────────────────────────────────┘  │   │
│  │                                                              │   │
│  │  ┌─────────────────────┐  ┌────────────────────────────┐    │   │
│  │  │  androidMain        │  │  iosMain                   │    │   │
│  │  │  ├─ Android specifics│  │  ├─ iOS specifics         │    │   │
│  │  │  └─ Platform APIs   │  │  └─ Platform APIs         │    │   │
│  │  └─────────────────────┘  └────────────────────────────┘    │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌──────────────────────────┐  ┌──────────────────────────────┐      │
│  │  Android App Module     │  │  iOS App Module (SwiftUI)   │      │
│  │  ├─ Jetpack Compose     │  │ ├─ SwiftUI (via CMP)        │      │
│  │  ├─ Material Design 3   │  │ ├─ iOS Design              │      │
│  │  └─ Android APIs       │  │ └─ iOS APIs                │      │
│  └──────────────────────────┘  └──────────────────────────────┘      │
└─────────────────────────────────────────────────────────────────────┘
```

### 1.2 Module Structure

```
ArmorClaw/
├── gradle/
│   └── libs.versions.toml           # Dependency versions
├── shared/                          # KMP shared module
│   └── composeApp/
│       ├── src/
│       │   ├── commonMain/
│       │   │   ├── kotlin/
│       │   │   │   ├── ui/
│       │   │   │   │   ├── components/    # Reusable UI components
│       │   │   │   │   ├── screens/      # Full screen layouts
│       │   │   │   │   ├── theme/        # Design system
│       │   │   │   │   └── navigation/   # Navigation
│       │   │   │   ├── domain/          # Business logic
│       │   │   │   │   ├── model/
│       │   │   │   │   ├── repository/
│       │   │   │   │   └── usecase/
│       │   │   │   └── platform/        # Platform interfaces
│       │   │   │       ├── biometric/
│       │   │   │       ├── clipboard/
│       │   │   │       └── notification/
│       │   │   └── resources/          # Shared resources
│       │   ├── androidMain/
│       │   │   ├── kotlin/
│       │   │   │   ├── platform/       # Android implementations
│       │   │   │   └── di/            # Android DI setup
│       │   │   └── res/
│       │   ├── iosMain/
│       │   │   ├── kotlin/
│       │   │   │   ├── platform/       # iOS implementations
│       │   │   │   └── di/            # iOS DI setup
│       │   │   └── resources/
│       │   ├── commonTest/
│       │   └── build.gradle.kts
│       └── build.gradle.kts
├── androidApp/                      # Android app module
│   ├── src/
│   │   ├── main/
│   │   │   ├── kotlin/
│   │   │   │   └── com/armorclaw/app/
│   │   │   │       ├── MainActivity.kt
│   │   │   │       ├── ArmorClawApplication.kt
│   │   │   │       └── di/            # Android DI modules
│   │   │   ├── res/
│   │   │   └── AndroidManifest.xml
│   │   └── test/
│   └── build.gradle.kts
├── iosApp/                         # iOS app module (Xcode)
│   ├── ArmorClaw/
│   │   ├── ArmorClawApp.swift
│   │   ├── ContentView.swift
│   │   └── Info.plist
│   └── ArmorClaw.xcodeproj/
├── gradle.properties
├── settings.gradle.kts
└── build.gradle.kts
```

---

## 2. Shared UI Component Architecture

### 2.1 Design System (100% Shared)

```kotlin
// commonMain/kotlin/ui/theme/Theme.kt
@Composable
fun ArmorClawTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit
) {
    val colors = if (darkTheme) DarkColorPalette else LightColorPalette
    val typography = ArmorClawTypography
    val shapes = ArmorClawShapes
    
    MaterialTheme(
        colors = colors,
        typography = typography,
        shapes = shapes,
        content = content
    )
}

// Compatible with both Material 3 (Android) and SwiftUI design tokens
object ArmorClawTheme {
    val Colors: Colors
        @Composable get() = MaterialTheme.colors
    
    val Typography: Typography
        @Composable get() = MaterialTheme.typography
    
    val Shapes: Shapes
        @Composable get() = MaterialTheme.shapes
}

// Design tokens (can be exported to SwiftUI)
object DesignTokens {
    object Spacing {
        val xs = 4.dp
        val sm = 8.dp
        val md = 16.dp
        val lg = 24.dp
        val xl = 32.dp
    }
    
    object Radius {
        val sm = 4.dp
        val md = 8.dp
        val lg = 16.dp
        val xl = 24.dp
    }
    
    object Elevation {
        val none = 0.dp
        val sm = 2.dp
        val md = 4.dp
        val lg = 8.dp
        val xl = 16.dp
    }
}
```

### 2.2 Atomic Components (100% Shared)

```kotlin
// commonMain/kotlin/ui/components/atom/Button.kt
@Composable
fun ArmorClawButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    variant: ButtonVariant = ButtonVariant.Primary,
    size: ButtonSize = ButtonSize.Medium,
    enabled: Boolean = true,
    loading: Boolean = false
) {
    val colors = when (variant) {
        ButtonVariant.Primary -> ButtonDefaults.buttonColors()
        ButtonVariant.Secondary -> ButtonDefaults.buttonColors(
            containerColor = MaterialTheme.colors.secondary,
            contentColor = MaterialTheme.colors.onSecondary
        )
        ButtonVariant.Outline -> ButtonDefaults.outlinedButtonColors()
        ButtonVariant.Text -> ButtonDefaults.textButtonColors()
    }
    
    val buttonSize = when (size) {
        ButtonSize.Small -> ButtonDefaults.smallButtonSizes()
        ButtonSize.Medium -> ButtonDefaults.buttonSizes()
        ButtonSize.Large -> ButtonDefaults.largeButtonSizes()
    }
    
    Button(
        onClick = onClick,
        modifier = modifier,
        enabled = enabled && !loading,
        colors = colors,
        contentPadding = buttonSize.contentPadding
    ) {
        if (loading) {
            CircularProgressIndicator(
                modifier = Modifier.size(buttonSize.iconSize),
                strokeWidth = 2.dp
            )
        } else {
            Text(
                text = text,
                style = when (size) {
                    ButtonSize.Small -> MaterialTheme.typography.buttonSmall
                    ButtonSize.Medium -> MaterialTheme.typography.button
                    ButtonSize.Large -> MaterialTheme.typography.buttonLarge
                }
            )
        }
    }
}

enum class ButtonVariant { Primary, Secondary, Outline, Text }
enum class ButtonSize { Small, Medium, Large }
```

### 2.3 Molecular Components (100% Shared)

```kotlin
// commonMain/kotlin/ui/components/molecule/InputField.kt
@Composable
fun InputField(
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    label: String? = null,
    placeholder: String? = null,
    error: String? = null,
    leadingIcon: ImageVector? = null,
    trailingIcon: (@Composable () -> Unit)? = null,
    keyboardType: KeyboardType = KeyboardType.Text,
    imeAction: ImeAction = ImeAction.Next,
    imeActionHandler: (() -> Unit)? = null,
    maxLength: Int? = null,
    showCharacterCount: Boolean = false
) {
    val isError = error != null
    
    OutlinedTextField(
        value = value,
        onValueChange = { 
            val newValue = if (maxLength != null) {
                it.take(maxLength)
            } else {
                it
            }
            onValueChange(newValue)
        },
        modifier = modifier,
        label = label?.let { { Text(it) } },
        placeholder = placeholder?.let { { Text(it) } },
        leadingIcon = leadingIcon?.let { 
            { Icon(imageVector = it, contentDescription = null) } 
        },
        trailingIcon = trailingIcon,
        isError = isError,
        keyboardOptions = KeyboardOptions(
            keyboardType = keyboardType,
            imeAction = imeAction
        ),
        keyboardActions = KeyboardActions(
            onAny = { imeActionHandler?.invoke() }
        ),
        supportingText = {
            Column {
                if (isError) {
                    Text(
                        text = error!!,
                        color = MaterialTheme.colors.error,
                        style = MaterialTheme.typography.caption
                    )
                }
                if (showCharacterCount && maxLength != null) {
                    Text(
                        text = "${value.length} / $maxLength",
                        color = if (isError) 
                            MaterialTheme.colors.error 
                        else 
                            MaterialTheme.colors.onSurface.copy(alpha = 0.6f),
                        style = MaterialTheme.typography.caption,
                        textAlign = TextAlign.End,
                        modifier = Modifier.fillMaxWidth()
                    )
                }
            }
        },
        singleLine = true
    )
}

// commonMain/kotlin/ui/components/molecule/Card.kt
@Composable
fun ArmorClawCard(
    modifier: Modifier = Modifier,
    onClick: (() -> Unit)? = null,
    elevation: CardElevation = CardDefaults.cardElevation(),
    colors: CardColors = CardDefaults.cardColors(),
    content: @Composable ColumnScope.() -> Unit
) {
    Card(
        onClick = { onClick?.invoke() },
        modifier = modifier,
        elevation = elevation,
        colors = colors,
        shape = MaterialTheme.shapes.medium
    ) {
        Column(
            modifier = Modifier.padding(DesignTokens.Spacing.md),
            content = content
        )
    }
}
```

### 2.4 Organism Components (100% Shared)

```kotlin
// commonMain/kotlin/ui/components/organism/MessageBubble.kt
@Composable
fun MessageBubble(
    message: Message,
    modifier: Modifier = Modifier,
    isOutgoing: Boolean
) {
    val bubbleColor = if (isOutgoing) {
        MaterialTheme.colors.primary
    } else {
        MaterialTheme.colors.surface
    }
    
    val textColor = if (isOutgoing) {
        MaterialTheme.colors.onPrimary
    } else {
        MaterialTheme.colors.onSurface
    }
    
    Surface(
        modifier = modifier,
        color = bubbleColor,
        shape = RoundedCornerShape(
            topStart = if (isOutgoing) DesignTokens.Radius.md else 0.dp,
            topEnd = if (isOutgoing) 0.dp else DesignTokens.Radius.md,
            bottomStart = DesignTokens.Radius.md,
            bottomEnd = DesignTokens.Radius.md
        ),
        elevation = DesignTokens.Elevation.sm
    ) {
        Column(
            modifier = Modifier.padding(
                horizontal = DesignTokens.Spacing.md,
                vertical = DesignTokens.Spacing.sm
            )
        ) {
            Text(
                text = message.content,
                color = textColor,
                style = MaterialTheme.typography.body1
            )
            
            Spacer(modifier = Modifier.height(4.dp))
            
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.End
            ) {
                Text(
                    text = message.timestamp.formatTime(),
                    color = textColor.copy(alpha = 0.7f),
                    style = MaterialTheme.typography.caption
                )
                
                if (isOutgoing) {
                    Spacer(modifier = Modifier.width(4.dp))
                    MessageStatusIcon(status = message.status)
                }
            }
        }
    }
}

// commonMain/kotlin/ui/components/organism/EmptyState.kt
@Composable
fun EmptyState(
    icon: ImageVector,
    title: String,
    message: String,
    modifier: Modifier = Modifier,
    action: EmptyAction? = null
) {
    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(DesignTokens.Spacing.xl),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            modifier = Modifier.size(80.dp),
            tint = MaterialTheme.colors.onSurface.copy(alpha = 0.3f)
        )
        
        Spacer(modifier = Modifier.height(DesignTokens.Spacing.lg))
        
        Text(
            text = title,
            style = MaterialTheme.typography.h6,
            color = MaterialTheme.colors.onSurface
        )
        
        Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))
        
        Text(
            text = message,
            style = MaterialTheme.typography.body2,
            color = MaterialTheme.colors.onSurface.copy(alpha = 0.7f),
            textAlign = TextAlign.Center
        )
        
        if (action != null) {
            Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))
            action()
        }
    }
}
```

---

## 3. Screen Components (100% Shared)

### 3.1 Onboarding Screens

```kotlin
// commonMain/kotlin/ui/screens/onboarding/WelcomeScreen.kt
@Composable
fun WelcomeScreen(
    onGetStarted: () -> Unit,
    onSkip: () -> Unit
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(DesignTokens.Spacing.xl)
            .verticalScroll(rememberScrollState()),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        // Logo
        Icon(
            imageVector = Icons.Default.Shield,
            contentDescription = null,
            modifier = Modifier.size(120.dp),
            tint = MaterialTheme.colors.primary
        )
        
        Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))
        
        // Title
        Text(
            text = "Welcome to ArmorClaw",
            style = MaterialTheme.typography.h4,
            textAlign = TextAlign.Center
        )
        
        Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))
        
        // Subtitle
        Text(
            text = "Secure AI agents in your pocket",
            style = MaterialTheme.typography.subtitle1,
            color = MaterialTheme.colors.onSurface.copy(alpha = 0.7f),
            textAlign = TextAlign.Center
        )
        
        Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))
        
        // Features
        FeatureList(
            features = listOf(
                Feature(
                    icon = Icons.Default.Security,
                    title = "Enterprise Security",
                    description = "Your API keys never leave secure container"
                ),
                Feature(
                    icon = Icons.Default.Chat,
                    title = "Chat Anywhere",
                    description = "Connect to agents from anywhere with Matrix"
                ),
                Feature(
                    icon = Icons.Default.Lock,
                    title = "Zero-Trust",
                    description = "Protected even if agent is compromised"
                )
            )
        )
        
        Spacer(modifier = Modifier.weight(1f))
        
        // Actions
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
        ) {
            OutlinedButton(
                onClick = onSkip,
                modifier = Modifier.weight(1f)
            ) {
                Text("Skip")
            }
            
            Button(
                onClick = onGetStarted,
                modifier = Modifier.weight(1f)
            ) {
                Text("Get Started")
            }
        }
    }
}

@Composable
private fun FeatureList(features: List<Feature>) {
    Column(
        verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
    ) {
        features.forEach { feature ->
            FeatureCard(feature)
        }
    }
}

@Composable
private fun FeatureCard(feature: Feature) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
    ) {
        Icon(
            imageVector = feature.icon,
            contentDescription = null,
            tint = MaterialTheme.colors.primary
        )
        
        Column {
            Text(
                text = feature.title,
                style = MaterialTheme.typography.subtitle2
            )
            Text(
                text = feature.description,
                style = MaterialTheme.typography.body2,
                color = MaterialTheme.colors.onSurface.copy(alpha = 0.7f)
            )
        }
    }
}

data class Feature(
    val icon: ImageVector,
    val title: String,
    val description: String
)
```

### 3.2 Chat Screen

```kotlin
// commonMain/kotlin/ui/screens/chat/ChatScreen.kt
@Composable
fun ChatScreen(
    viewModel: ChatViewModel,
    onNavigateBack: () -> Unit
) {
    val uiState by viewModel.uiState.collectAsState()
    
    Scaffold(
        topBar = {
            ChatTopBar(
                roomName = uiState.roomName,
                onNavigateBack = onNavigateBack,
                syncState = uiState.syncState
            )
        },
        bottomBar = {
            MessageInputBar(
                message = uiState.inputMessage,
                onMessageChange = { viewModel.onInputChanged(it) },
                onSend = { viewModel.sendMessage() },
                onAttach = { viewModel.showAttachmentMenu() },
                onVoice = { viewModel.startVoiceInput() },
                canSend = uiState.canSend
            )
        }
    ) { padding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
        ) {
            when (uiState.loadState) {
                LoadState.Loading -> LoadingSkeleton()
                LoadState.Success -> MessageList(
                    messages = uiState.messages,
                    onRetryMessage = { viewModel.retryMessage(it) }
                )
                LoadState.Error -> ErrorState(
                    error = uiState.error ?: "Unknown error",
                    onRetry = { viewModel.retryLoad() }
                )
                is LoadState.Empty -> EmptyState(
                    icon = Icons.Default.ChatBubbleOutline,
                    title = "No messages yet",
                    message = "Send a message to start the conversation"
                )
            }
        }
    }
}

@Composable
private fun ChatTopBar(
    roomName: String,
    onNavigateBack: () -> Unit,
    syncState: SyncState
) {
    TopAppBar(
        title = {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                Text(text = roomName)
                SyncStatusIndicator(state = syncState)
            }
        },
        navigationIcon = {
            IconButton(onClick = onNavigateBack) {
                Icon(Icons.Default.ArrowBack, contentDescription = "Back")
            }
        },
        actions = {
            IconButton(onClick = { /* Show menu */ }) {
                Icon(Icons.Default.MoreVert, contentDescription = "More")
            }
        }
    )
}

@Composable
private fun MessageList(
    messages: List<Message>,
    onRetryMessage: (String) -> Unit
) {
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        reverseLayout = true,
        contentPadding = PaddingValues(DesignTokens.Spacing.sm)
    ) {
        items(messages, key = { it.id }) { message ->
            MessageBubble(
                message = message,
                isOutgoing = message.isOutgoing,
                modifier = Modifier.padding(
                    horizontal = DesignTokens.Spacing.sm,
                    vertical = DesignTokens.Spacing.xs
                )
            )
        }
    }
}

@Composable
private fun MessageInputBar(
    message: String,
    onMessageChange: (String) -> Unit,
    onSend: () -> Unit,
    onAttach: () -> Unit,
    onVoice: () -> Unit,
    canSend: Boolean
) {
    Surface(
        elevation = DesignTokens.Elevation.md
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(DesignTokens.Spacing.sm),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm)
        ) {
            IconButton(onClick = onAttach) {
                Icon(Icons.Default.AttachFile, contentDescription = "Attach")
            }
            
            OutlinedTextField(
                value = message,
                onValueChange = onMessageChange,
                modifier = Modifier.weight(1f),
                placeholder = { Text("Type a message...") },
                maxLines = 4,
                shape = MaterialTheme.shapes.medium
            )
            
            IconButton(
                onClick = onVoice,
                enabled = message.isEmpty()
            ) {
                Icon(Icons.Default.Mic, contentDescription = "Voice")
            }
            
            IconButton(
                onClick = onSend,
                enabled = canSend
            ) {
                Icon(
                    imageVector = if (message.isEmpty()) Icons.Default.Send else Icons.Default.Send,
                    contentDescription = "Send",
                    tint = if (canSend) MaterialTheme.colors.primary else MaterialTheme.colors.onSurface.copy(alpha = 0.3f)
                )
            }
        }
    }
}
```

---

## 4. Shared Business Logic Layer

### 4.1 Repository Pattern

```kotlin
// commonMain/kotlin/domain/repository/MessageRepository.kt
interface MessageRepository {
    suspend fun getMessages(roomId: String, limit: Int): Result<List<Message>>
    suspend fun sendMessage(roomId: String, content: String): Result<Message>
    suspend fun retryMessage(messageId: String): Result<Message>
    fun observeMessages(roomId: String): Flow<List<Message>>
}

// commonMain/kotlin/domain/repository/SyncRepository.kt
interface SyncRepository {
    suspend fun syncWhenOnline(): SyncResult
    fun observeSyncState(): Flow<SyncState>
    fun isOnline(): Boolean
}

// commonMain/kotlin/domain/repository/AuthRepository.kt
interface AuthRepository {
    suspend fun login(config: ServerConfig): Result<UserSession>
    suspend fun logout(): Result<Unit>
    fun getCurrentSession(): UserSession?
    fun observeSession(): Flow<UserSession?>
}
```

### 4.2 Use Cases

```kotlin
// commonMain/kotlin/domain/usecase/SendMessageUseCase.kt
class SendMessageUseCase(
    private val messageRepository: MessageRepository,
    private val syncRepository: SyncRepository
) {
    suspend operator fun invoke(
        roomId: String,
        content: String
    ): Result<Message> {
        // Validate input
        if (content.isBlank()) {
            return Result.failure(ValidationException("Message cannot be empty"))
        }
        
        if (content.length > MAX_MESSAGE_LENGTH) {
            return Result.failure(ValidationException("Message too long"))
        }
        
        // Send message
        return messageRepository.sendMessage(roomId, content)
    }
    
    companion object {
        const val MAX_MESSAGE_LENGTH = 10000
    }
}

// commonMain/kotlin/domain/usecase/LoadMessagesUseCase.kt
class LoadMessagesUseCase(
    private val messageRepository: MessageRepository
) {
    suspend operator fun invoke(
        roomId: String,
        limit: Int = 50
    ): Result<List<Message>> {
        return messageRepository.getMessages(roomId, limit)
    }
}
```

### 4.3 ViewModel (Shared)

```kotlin
// commonMain/kotlin/ui/screens/chat/ChatViewModel.kt
class ChatViewModel(
    private val sendMessageUseCase: SendMessageUseCase,
    private val loadMessagesUseCase: LoadMessagesUseCase,
    private val syncRepository: SyncRepository,
    private val roomId: String
) : ViewModel() {
    
    private val _uiState = MutableStateFlow(ChatUiState())
    val uiState: StateFlow<ChatUiState> = _uiState.asStateFlow()
    
    init {
        loadMessages()
        observeSyncState()
    }
    
    private fun loadMessages() {
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(loadState = LoadState.Loading)
            
            loadMessagesUseCase(roomId)
                .onSuccess { messages ->
                    _uiState.value = _uiState.value.copy(
                        loadState = LoadState.Success,
                        messages = messages
                    )
                }
                .onFailure { error ->
                    _uiState.value = _uiState.value.copy(
                        loadState = LoadState.Error(error.message),
                        error = error.message
                    )
                }
        }
    }
    
    private fun observeSyncState() {
        viewModelScope.launch {
            syncRepository.observeSyncState().collect { syncState ->
                _uiState.value = _uiState.value.copy(syncState = syncState)
            }
        }
    }
    
    fun onInputChanged(input: String) {
        _uiState.value = _uiState.value.copy(
            inputMessage = input,
            canSend = input.isNotBlank()
        )
    }
    
    fun sendMessage() {
        viewModelScope.launch {
            val state = _uiState.value
            sendMessageUseCase(roomId, state.inputMessage)
                .onSuccess { message ->
                    _uiState.value = state.copy(
                        inputMessage = "",
                        messages = listOf(message) + state.messages
                    )
                }
                .onFailure { error ->
                    _uiState.value = state.copy(
                        error = error.message
                    )
                }
        }
    }
}

data class ChatUiState(
    val loadState: LoadState = LoadState.Loading,
    val messages: List<Message> = emptyList(),
    val inputMessage: String = "",
    val canSend: Boolean = false,
    val syncState: SyncState = SyncState.Idle,
    val error: String? = null,
    val roomName: String = "ArmorClaw Agent"
)

sealed class LoadState {
    object Loading : LoadState()
    object Success : LoadState()
    data class Error(val message: String?) : LoadState()
    data class Empty(val message: String) : LoadState()
}

sealed class SyncState {
    object Idle : SyncState()
    object Syncing : SyncState()
    object Offline : SyncState()
    data class Error(val message: String) : SyncState()
}
```

---

## 5. Platform-Specific Integrations

### 5.1 Platform Interfaces

```kotlin
// commonMain/kotlin/platform/biometric/BiometricAuth.kt
expect class BiometricAuth() {
    suspend fun authenticate(prompt: String): Result<String>
    fun isAvailable(): Boolean
}

// commonMain/kotlin/platform/clipboard/SecureClipboard.kt
expect class SecureClipboard() {
    fun copySensitive(data: String, autoClearAfter: Duration)
    fun clear()
}

// commonMain/kotlin/platform/notification/NotificationManager.kt
expect class NotificationManager() {
    suspend fun registerDevice(token: String)
    fun showNotification(notification: Notification)
}
```

### 5.2 Android Implementations

```kotlin
// androidMain/kotlin/platform/biometric/BiometricAuth.android.kt
actual class BiometricAuth(
    private val activity: ComponentActivity
) {
    private val promptInfo = BiometricPrompt.PromptInfo.Builder()
        .setTitle("Authentication Required")
        .setSubtitle("Use biometric to continue")
        .setNegativeButtonText("Cancel")
        .build()
    
    actual suspend fun authenticate(prompt: String): Result<String> {
        return suspendCancellableCoroutine { continuation ->
            val prompt = BiometricPrompt(
                activity,
                ContextCompat.getMainExecutor(activity),
                object : BiometricPrompt.AuthenticationCallback() {
                    override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
                        continuation.resume(Result.success("authenticated"))
                    }
                    
                    override fun onAuthenticationFailed() {
                        continuation.resume(Result.failure(Exception("Authentication failed")))
                    }
                    
                    override fun onError(error: Int, errString: CharSequence) {
                        continuation.resume(Result.failure(Exception(errString.toString())))
                    }
                }
            )
            
            prompt.authenticate(promptInfo)
        }
    }
    
    actual fun isAvailable(): Boolean {
        return BiometricManager.from(activity)
            .canAuthenticate(BiometricPrompt.Authenticators.BIOMETRIC_STRONG) ==
            BiometricManager.BIOMETRIC_SUCCESS
    }
}

// androidMain/kotlin/platform/clipboard/SecureClipboard.android.kt
actual class SecureClipboard(
    private val context: Context,
    private val scope: CoroutineScope
) {
    private val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
    
    actual fun copySensitive(data: String, autoClearAfter: Duration) {
        val clip = ClipData.newPlainText("ArmorClaw", data)
        clipboard.setPrimaryClip(clip)
        
        scope.launch {
            delay(autoClearAfter)
            clear()
        }
    }
    
    actual fun clear() {
        clipboard.setPrimaryClip(ClipData.newPlainText("", ""))
    }
}
```

### 5.3 iOS Implementations

```kotlin
// iosMain/kotlin/platform/biometric/BiometricAuth.ios.kt
actual class BiometricAuth {
    private val context = LAContext()
    
    actual suspend fun authenticate(prompt: String): Result<String> {
        return suspendCoroutine { continuation ->
            val errorPtr = objc_referencedPointer(objc_class?.alloc)
            
            val success = context.canEvaluatePolicy(
                LAPolicy.deviceOwnerAuthenticationWithBiometrics,
                errorPtr
            )
            
            if (success) {
                context.evaluatePolicy(
                    LAPolicy.deviceOwnerAuthenticationWithBiometrics,
                    prompt
                ) { success, error ->
                    if (success) {
                        continuation.resume(Result.success("authenticated"))
                    } else {
                        continuation.resume(Result.failure(Exception(error?.localizedDescription)))
                    }
                }
            } else {
                continuation.resume(Result.failure(Exception("Biometric not available")))
            }
        }
    }
    
    actual fun isAvailable(): Boolean {
        return LAContext().canEvaluatePolicy(
            LAPolicy.deviceOwnerAuthenticationWithBiometrics,
            null
        )
    }
}

// iosMain/kotlin/platform/clipboard/SecureClipboard.ios.kt
actual class SecureClipboard(
    private val scope: CoroutineScope
) {
    private val pasteboard = UIPasteboard.generalPasteboard
    
    actual fun copySensitive(data: String, autoClearAfter: Duration) {
        pasteboard.string = data
        
        scope.launch {
            delay(autoClearAfter)
            clear()
        }
    }
    
    actual fun clear() {
        pasteboard.string = ""
    }
}
```

---

## 6. Navigation (Shared)

```kotlin
// commonMain/kotlin/ui/navigation/NavRoutes.kt
sealed class NavRoute(val route: String) {
    object Welcome : NavRoute("welcome")
    object Security : NavRoute("security")
    object Connect : NavRoute("connect")
    object Permissions : NavRoute("permissions")
    object Complete : NavRoute("complete")
    object Home : NavRoute("home")
    
    object Chat : NavRoute("chat/{roomId}") {
        fun create(roomId: String) = "chat/$roomId"
    }
    
    object Settings : NavRoute("settings")
}

// commonMain/kotlin/ui/navigation/ArmorClawNavHost.kt
@Composable
fun ArmorClawNavHost(
    navController: NavController,
    startDestination: NavRoute
) {
    NavHost(
        navController = navController,
        startDestination = startDestination.route
    ) {
        composable(NavRoute.Welcome.route) {
            WelcomeScreen(
                onGetStarted = { navController.navigate(NavRoute.Security.route) },
                onSkip = { navController.navigate(NavRoute.Home.route) }
            )
        }
        
        composable(NavRoute.Security.route) {
            SecurityExplanationScreen(
                onNext = { navController.navigate(NavRoute.Connect.route) },
                onBack = { navController.popBackStack() }
            )
        }
        
        composable(NavRoute.Connect.route) {
            ConnectServerScreen(
                onConnected = { navController.navigate(NavRoute.Permissions.route) },
                onBack = { navController.popBackStack() }
            )
        }
        
        composable(NavRoute.Permissions.route) {
            PermissionsScreen(
                onComplete = { navController.navigate(NavRoute.Complete.route) },
                onBack = { navController.popBackStack() }
            )
        }
        
        composable(NavRoute.Complete.route) {
            CompletionScreen(
                onStartChatting = { 
                    navController.navigate(NavRoute.Home.route) {
                        popUpTo(NavRoute.Welcome.route) { inclusive = true }
                    }
                }
            )
        }
        
        composable(NavRoute.Home.route) {
            HomeScreen(
                onRoomClick = { roomId ->
                    navController.navigate(NavRoute.Chat.create(roomId))
                }
            )
        }
        
        composable(
            route = NavRoute.Chat.route,
            arguments = listOf(navArgument("roomId") { type = NavType.StringType })
        ) { backStackEntry ->
            val roomId = backStackEntry.arguments?.getString("roomId") ?: return@composable
            ChatScreen(
                viewModel = getViewModel(ChatViewModel::class.java) { parametersOf(roomId) },
                onNavigateBack = { navController.popBackStack() }
            )
        }
        
        composable(NavRoute.Settings.route) {
            SettingsScreen(
                onBack = { navController.popBackStack() }
            )
        }
    }
}
```

---

## 7. Android App Integration

```kotlin
// androidApp/src/main/kotlin/com/armorclaw/app/MainActivity.kt
class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        
        setContent {
            ArmorClawTheme {
                val navController = rememberNavController()
                val startDestination = if (hasCompletedOnboarding()) {
                    NavRoute.Home.route
                } else {
                    NavRoute.Welcome.route
                }
                
                ArmorClawNavHost(
                    navController = navController,
                    startDestination = NavRoute.Welcome
                )
            }
        }
    }
    
    private fun hasCompletedOnboarding(): Boolean {
        return getSharedPreferences("prefs", Context.MODE_PRIVATE)
            .getBoolean("onboarding_completed", false)
    }
}

// androidApp/src/main/kotlin/com/armorclaw/app/ArmorClawApplication.kt
class ArmorClawApplication : Application() {
    val appContainer: AppContainer by lazy { AppContainer(this) }
}

// androidApp/src/main/kotlin/com/armorclaw/app/di/AppContainer.kt
class AppContainer(context: Context) {
    private val database: ArmorClawDatabase by lazy {
        Room.databaseBuilder(
            context,
            ArmorClawDatabase::class.java,
            "armorclaw.db"
        ).build()
    }
    
    private val matrixClient: MatrixClient by lazy {
        MatrixClientImpl()
    }
    
    val messageRepository: MessageRepository by lazy {
        MessageRepositoryImpl(database, matrixClient)
    }
    
    val syncRepository: SyncRepository by lazy {
        SyncRepositoryImpl(matrixClient, context)
    }
    
    val authRepository: AuthRepository by lazy {
        AuthRepositoryImpl(context, matrixClient)
    }
    
    fun provideViewModel(viewModelFactory: ViewModelFactory): ViewModelProvider.Factory {
        return viewModelFactory
    }
}
```

---

## 8. Implementation Roadmap

### Phase 1: Foundation (2-3 weeks)

**Week 1: Project Setup**
- [ ] Initialize Compose Multiplatform project
- [ ] Set up Gradle configuration
- [ ] Configure shared module structure
- [ ] Setup Android app module
- [ ] Setup iOS app module (Xcode project)
- [ ] Configure version catalog (libs.versions.toml)

**Week 2: Design System**
- [ ] Implement design tokens (colors, typography, spacing)
- [ ] Create Material Theme for CMP
- [ ] Implement atomic components (Button, Input, Card)
- [ ] Implement molecular components
- [ ] Setup navigation structure

**Week 3: Core Business Logic**
- [ ] Define repository interfaces
- [ ] Implement data models
- [ ] Create use cases
- [ ] Setup dependency injection

### Phase 2: Onboarding (2 weeks)

- [ ] Welcome screen
- [ ] Security explanation screen
- [ ] Connect server screen
- [ ] Permissions screen
- [ ] Completion screen
- [ ] Onboarding state management

### Phase 3: Chat Foundation (2 weeks)

- [ ] Chat screen layout
- [ ] Message list component
- [ ] Message bubble component
- [ ] Message input bar
- [ ] Sync state indicators
- [ ] Pull-to-refresh

### Phase 4: Platform Integrations (2 weeks)

**Android:**
- [ ] Biometric auth implementation
- [ ] Secure clipboard implementation
- [ ] Push notifications (FCM)
- [ ] Certificate pinning
- [ ] Crash reporting (Sentry)

**iOS:**
- [ ] Biometric auth (Face ID/Touch ID)
- [ ] Secure clipboard
- [ ] Push notifications (APNs)
- [ ] Certificate pinning
- [ ] Crash reporting

### Phase 5: Offline Sync (2 weeks)

- [ ] Local database (SQLCipher via SQLDelight)
- [ ] Offline queue implementation
- [ ] Sync state machine
- [ ] Conflict resolution
- [ ] Background sync worker

### Phase 6: Polish & Launch (1-2 weeks)

- [ ] Performance optimization
- [ ] App size optimization
- [ ] Accessibility audit
- [ ] E2E testing
- [ ] Store submission assets

---

## 9. Build Configuration

### 9.1 libs.versions.toml

```toml
[versions]
compose = "1.5.0"
kotlin = "1.9.20"
kmp = "1.5.0"
coroutines = "1.7.3"
koin = "3.5.0"
ktor = "2.3.5"
sqlDelight = "2.0.0"
sentry = "6.34.0"

[libraries]
compose-ui = { module = "androidx.compose.ui:ui", version.ref = "compose" }
compose-material = { module = "androidx.compose.material:material", version.ref = "compose" }
compose-foundation = { module = "androidx.compose.foundation:foundation", version.ref = "compose" }
compose-runtime = { module = "androidx.compose.runtime:runtime", version.ref = "compose" }

kotlinx-coroutines-core = { module = "org.jetbrains.kotlinx:kotlinx-coroutines-core", version.ref = "coroutines" }
kotlinx-coroutines-android = { module = "org.jetbrains.kotlinx:kotlinx-coroutines-android", version.ref = "coroutines" }

koin-core = { module = "io.insert-koin:koin-core", version.ref = "koin" }
koin-compose = { module = "io.insert-koin:koin-androidx-compose", version.ref = "koin" }

ktor-core = { module = "io.ktor:ktor-client-core", version.ref = "ktor" }
ktor-cio = { module = "io.ktor:ktor-client-cio", version.ref = "ktor" }
ktor-logging = { module = "io.ktor:ktor-client-logging", version.ref = "ktor" }

sqlDelight-android = { module = "app.cash.sqldelight:android-driver", version.ref = "sqlDelight" }
sqlDelight-coroutines = { module = "app.cash.sqldelight:coroutines-extensions", version.ref = "sqlDelight" }

sentry-kotlin = { module = "io.sentry:sentry-kotlin-multiplatform", version.ref = "sentry" }

[plugins]
kotlin-multiplatform = { id = "org.jetbrains.kotlin.multiplatform", version.ref = "kotlin" }
compose-multiplatform = { id = "org.jetbrains.compose", version.ref = "kmp" }
```

### 9.2 shared/build.gradle.kts

```kotlin
plugins {
    kotlin("multiplatform")
    id("org.jetbrains.compose")
    id("com.android.library")
}

kotlin {
    androidTarget()
    iosX64()
    iosArm64()
    iosSimulatorArm64()
    
    sourceSets {
        val commonMain by getting {
            dependencies {
                implementation(compose.ui)
                implementation(compose.material)
                implementation(compose.foundation)
                implementation(compose.runtime)
                implementation(libs.kotlinx.coroutines.core)
                implementation(libs.koin.core)
                implementation(libs.ktor.core)
                implementation(libs.sqlDelight.coroutines)
                implementation(libs.sentry.kotlin)
            }
        }
        
        val androidMain by getting {
            dependencies {
                implementation(libs.kotlinx.coroutines.android)
                implementation(libs.sqlDelight.android)
            }
        }
        
        val iosMain by getting {
            dependencies {
                // iOS-specific dependencies
            }
        }
    }
}

android {
    namespace = "com.armorclaw.shared"
    compileSdk = 34
}
```

---

## 10. Code Reusability Metrics

### Target Metrics

| Component Type | Target Shared % | Notes |
|---------------|---------------|-------|
| Design System | 100% | Colors, typography, spacing, shapes |
| Atomic UI | 100% | Buttons, inputs, icons, cards |
| Molecular UI | 100% | Form fields, headers, footers |
| Organism UI | 100% | Message bubbles, lists, empty states |
| Screen Layouts | 90% | ~10% platform-specific adaptations |
| ViewModels | 100% | All business logic shared |
| Use Cases | 100% | Pure Kotlin business logic |
| Repositories | 80% | Interfaces shared, implementations 20% platform |
| Platform Integrations | 0% | expect/actual implementations |

**Overall Target: ~85% shared code**

---

## 11. SwiftUI Integration Notes

### 11.1 Design System Mapping

| CMP Component | SwiftUI Equivalent | Mapping Strategy |
|---------------|------------------|------------------|
| MaterialTheme | Environment | Color palette via @Environment |
| Text | Text | String mapping |
| Button | Button | Action handlers mapped |
| TextField | TextField | State binding mapped |
| LazyColumn | List | Lazy loading adapter |
| Navigation | NavigationStack | Route mapping |
| @Composable | View | ComposeView on Android |

### 11.2 Shared State

```kotlin
// State that works with both CMP and SwiftUI
class SharedState<T>(initialValue: T) {
    private val _value = MutableStateFlow(initialValue)
    val value: StateFlow<T> = _value.asStateFlow()
    
    fun update(newValue: T) {
        _value.value = newValue
    }
}

// SwiftUI wrapper (iOS)
@available(iOS 13.0, *)
class ObservableState<T: AnyObject>: ObservableObject {
    @Published var value: T
    private var flowJob: Job? = nil
    
    init(stateFlow: StateFlow<T>) {
        self.value = stateFlow.value as! T
        self.flowJob = stateFlow.collect { newValue in
            DispatchQueue.main.async {
                self.value = newValue as! T
            }
        }
    }
}
```

---

**Document Version:** 1.0.0
**Last Updated:** 2026-02-10
**Status:** Ready for Implementation
