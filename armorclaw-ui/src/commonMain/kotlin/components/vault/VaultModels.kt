package components.vault

import androidx.compose.runtime.Immutable

/**
 * Vault Key Model for UI
 *
 * Represents a PII key stored in the Cold Vault.
 * This is a UI-specific model that mirrors the domain model.
 */
@Immutable
data class VaultKeyUi(
    val id: String,
    val fieldName: String,
    val displayName: String,
    val category: VaultKeyCategory,
    val sensitivity: VaultKeySensitivity,
    val lastAccessed: Long? = null,
    val accessCount: Int = 0
)

/**
 * Vault Key Category
 */
enum class VaultKeyCategory {
    PERSONAL,
    FINANCIAL,
    CONTACT,
    AUTHENTICATION,
    MEDICAL,
    OTHER
}

/**
 * Vault Key Sensitivity
 */
enum class VaultKeySensitivity {
    LOW,
    MEDIUM,
    HIGH,
    CRITICAL
}

/**
 * Vault State for UI
 */
@Immutable
data class VaultStateUi(
    val isInitialized: Boolean = false,
    val status: VaultStatus = VaultStatus.LOCKED,
    val storedKeys: List<VaultKeyUi> = emptyList(),
    val requiredKeys: Set<String> = emptySet(),
    val error: String? = null
)

/**
 * Vault Status
 */
enum class VaultStatus {
    LOCKED,
    SECURED,
    ACTIVE,
    ERROR
}
