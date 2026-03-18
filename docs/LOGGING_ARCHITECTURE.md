# Logging Architecture

## Overview

This document describes the logging architecture that ensures proper separation of concerns, allowing errors to be accurately traced to their source.

## Implementation Status

✅ **Complete** - All layer loggers implemented and integrated.

### Files Created/Updated

| File | Layer | Purpose |
|------|-------|---------|
| `LayerLogger.kt` | All | Layer-specific logger classes |
| `ErrorBoundary.kt` | All | Error handling and wrapping utilities |
| `BaseViewModel.kt` | ViewModel | Base class with built-in logging |
| `ChatViewModel.kt` | ViewModel | Updated with ViewModelLogger |
| `SettingsViewModel.kt` | ViewModel | Updated with ViewModelLogger |
| `HomeViewModel.kt` | ViewModel | Updated with ViewModelLogger |
| `LoginUseCase.kt` | UseCase | Updated with UseCaseLogger |
| `LogoutUseCase.kt` | UseCase | Updated with UseCaseLogger |
| `SendMessageUseCase.kt` | UseCase | Updated with UseCaseLogger |
| `GetRoomsUseCase.kt` | UseCase | Updated with UseCaseLogger |
| `MessageRepositoryImpl.kt` | Repository | Example with RepositoryLogger |
| `AuthRepositoryImpl.kt` | Repository | Example with RepositoryLogger |

## Principles

### 1. Layer-Specific Logging

Each architectural layer has its own logger that adds appropriate context:

| Layer | Logger Class | Purpose |
|-------|--------------|---------|
| Repository | `RepositoryLogger` | Data operations, cache, network, database |
| UseCase | `UseCaseLogger` | Business logic, validation, transformations |
| ViewModel | `ViewModelLogger` | UI state, user actions, navigation |
| Service | `ServiceLogger` | Background tasks, workers |

### 2. Tag Hierarchy

Tags follow the pattern: `Category.Module.Component`

```
LogTag.UI.Chat.MessageList     // UI layer - Chat - MessageList component
LogTag.ViewModel.Chat          // ViewModel layer - Chat
LogTag.UseCase.Message.Send    // UseCase layer - Message - Send
LogTag.Data.Repository.Message // Data layer - Repository - Message
LogTag.Network.Matrix.Sync     // Network layer - Matrix - Sync
```

### 3. Error Boundaries

Errors are caught and logged at layer boundaries:

```
┌─────────────────────────────────────────────────────────────────┐
│                        UI Layer                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   ViewModel                              │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │              UseCase                             │    │    │
│  │  │  ┌─────────────────────────────────────────┐    │    │    │
│  │  │  │          Repository                      │    │    │    │
│  │  │  │  ┌─────────────────────────────────┐    │    │    │    │
│  │  │  │  │      Network/Database           │    │    │    │    │
│  │  │  │  └─────────────────────────────────┘    │    │    │    │
│  │  │  └─────────────────────────────────────────┘    │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘

Error flows up, logged at each boundary
```

## Logger Methods by Layer

### RepositoryLogger
```kotlin
logOperationStart(operation, params)  // Start of operation
logOperationSuccess(operation, result) // Success with result
logOperationError(operation, error)    // Error with context
logCacheHit(key)                       // Cache hit
logCacheMiss(key)                      // Cache miss
logNetworkRequest(endpoint, method)    // Network call start
logNetworkResponse(endpoint, status, duration) // Network response
logDatabaseQuery(query, params)        // Database query
logTransformation(from, to)            // Data transformation
```

### UseCaseLogger
```kotlin
logStart(params)                       // Start of use case
logSuccess(result)                     // Success with result
logFailure(error, params)              // Failure with context
logValidationError(field, reason)      // Validation failed
logBusinessRuleViolation(rule, details) // Business rule violation
logExecutionTime(durationMs)           // Execution timing
```

### ViewModelLogger
```kotlin
logInit(params)                        // ViewModel init
logCleanup()                           // ViewModel cleanup
logUserAction(action, params)          // User action
logStateChange(stateName, value)       // State change
logUiEvent(event)                      // UI event emission
logError(operation, error, params)     // Error with context
logNavigation(destination)             // Navigation event
```

## Usage Examples

### Repository Layer

```kotlin
class MessageRepositoryImpl : MessageRepository {
    private val logger = repositoryLogger("MessageRepository", LogTag.Data.MessageRepository)

    override suspend fun getMessages(roomId: String): Result<List<Message>> = 
        repositoryErrorBoundarySuspend(logger, "getMessages") {
            logger.logOperationStart("getMessages", mapOf("roomId" to roomId))
            logger.logCacheMiss("messages:$roomId")
            logger.logDatabaseQuery("SELECT * FROM messages WHERE roomId = ?")
            logger.logNetworkRequest("/rooms/$roomId/messages", "GET")
            logger.logNetworkResponse("/rooms/$roomId/messages", 200, 150)
            logger.logTransformation("NetworkResponse", "DomainMessage")
            messages
        }
}
```

### UseCase Layer

```kotlin
class SendMessageUseCase(
    private val messageRepository: MessageRepository
) {
    private val logger = useCaseLogger("SendMessageUseCase", LogTag.UseCase.SendMessage)

    suspend operator fun invoke(roomId: String, content: String): Result<Message> {
        logger.logStart(mapOf("roomId" to roomId, "contentLength" to content.length))
        
        if (content.isBlank()) {
            logger.logValidationError("content", "Message cannot be empty")
            return Result.failure(DomainError.ValidationError(...))
        }
        
        val message = messageRepository.sendMessage(roomId, content).getOrThrow()
        logger.logSuccess("Message ${message.id} sent")
        return Result.success(message)
    }
}
```

### ViewModel Layer

```kotlin
class ChatViewModel : BaseViewModel() {
    private val logger = viewModelLogger("ChatViewModel", LogTag.ViewModel.Chat)

    init {
        logger.logInit(mapOf("roomId" to roomId))
    }

    fun sendMessage(content: String) {
        logger.logUserAction("sendMessage", mapOf("contentLength" to content.length))
        
        viewModelScope.launch {
            sendMessageUseCase(roomId, content)
                .onSuccess {
                    logger.logStateChange("messageSent", it.id)
                    logger.logUiEvent("messageSent")
                }
                .onFailure { error ->
                    logger.logError("sendMessage", error)
                }
        }
    }

    override fun onCleared() {
        logger.logCleanup()
    }
}
```

## Log Categories

### UI Category
- `UI.Navigation` - Navigation events
- `UI.Chat` - Chat screen events
- `UI.Profile` - Profile screen events
- `UI.Settings` - Settings screen events

### ViewModel Category
- `VM.Chat` - Chat ViewModel events
- `VM.Home` - Home ViewModel events
- `VM.Profile` - Profile ViewModel events

### UseCase Category
- `UseCase.Auth.Login` - Login use case
- `UseCase.Message.Send` - Send message use case
- `UseCase.Room.Create` - Create room use case

### Data Category
- `Data.Repository.Message` - Message repository
- `Data.Repository.Room` - Room repository
- `Data.Database` - Database operations
- `Data.Cache` - Cache operations

### Network Category
- `Network.Matrix` - Matrix protocol
- `Network.Matrix.Sync` - Matrix sync
- `Network.HTTP` - HTTP requests

### Platform Category
- `Platform.Android` - Android-specific
- `Platform.Notification` - Push notifications
- `Platform.Biometric` - Biometric auth

## Error Types

```kotlin
sealed class DomainError : Exception {
    class NetworkError(statusCode: Int?)    // Network failures
    class AuthError(authType: String?)       // Auth failures
    class ValidationError(field: String?)    // Validation failures
    class NotFoundError(resourceType: String?, resourceId: String?)  // 404
    class StorageError(operation: String?)   // Database/storage failures
    class SecurityError(securityType: String?) // Security failures
    class UnknownError                        // Catch-all
}
```

## Best Practices

### 1. Log at Boundaries

Only log at the entry/exit points of each layer:

```kotlin
// ✅ Good - Log at boundary
class SendMessageUseCase {
    suspend operator fun invoke(...) {
        logger.logStart() // Good
        validate() // Don't log inside validate()
        transform() // Don't log inside transform()
        send() // Don't log inside send()
        logger.logSuccess() // Good
    }
}
```

### 2. Include Relevant Context

```kotlin
// ✅ Good - Includes relevant context
logger.logError("sendMessage", error, mapOf(
    "roomId" to roomId,
    "messageLength" to content.length,
    "retryCount" to retryCount
))
```

### 3. Mask Sensitive Data

```kotlin
// ✅ Good - Passwords/tokens are masked
logger.logStart(mapOf(
    "username" to username,
    "password" to PasswordMasked(password)  // Will be logged as "***"
))
```

### 4. Use Appropriate Log Levels

| Level | Usage |
|-------|-------|
| DEBUG | Detailed operation flow, cache hits/misses |
| INFO | Important state changes, user actions |
| WARNING | Recoverable errors, validation failures |
| ERROR | Failures that affect user experience |

## Log Example Output

```
[UseCase/Auth/Login] Executing {"username":"john","password":"***"}
[Data/Repository/Auth] Starting: login
[Data/Repository/Auth] Network request POST /login
[Data/Repository/Auth] Network response 200 (342ms)
[Data/Repository/Auth] Transform: NetworkResponse -> DomainUser
[Data/Repository/Auth] Success: login
[UseCase/Auth/Login] Completed: User logged in
[VM/Auth/Login] State: isLoggedIn = true
[VM/Auth/Login] Navigate to: home
```

## Files Reference

| File | Purpose |
|------|---------|
| `LogTag.kt` | Tag definitions for all categories |
| `LayerLogger.kt` | Layer-specific logger implementations |
| `ErrorBoundary.kt` | Error handling and wrapping utilities |
| `AppLogger.kt` | Core logging implementation |
| `BaseViewModel.kt` | Base ViewModel with built-in logging |

## Testing

When testing, verify that:

1. Each layer logs at its boundaries
2. Errors include proper context
3. Sensitive data is masked
4. Log levels are appropriate
5. Error chains are preserved
