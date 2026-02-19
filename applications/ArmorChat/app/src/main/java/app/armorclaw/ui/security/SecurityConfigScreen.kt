package app.armorclaw.ui.security

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.toggleable
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

/**
 * Data category configuration for security settings
 */
data class DataCategoryConfig(
    val id: String,
    val name: String,
    val description: String,
    val examples: List<String>,
    val riskLevel: RiskLevel,
    val permission: PermissionLevel = PermissionLevel.DENY,
    val allowedWebsites: List<String> = emptyList(),
    val requiresApproval: Boolean = true
)

enum class RiskLevel {
    HIGH, MEDIUM, LOW
}

enum class PermissionLevel {
    DENY, ALLOW, ALLOW_ALL
}

/**
 * Security configuration screen for first-boot setup
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SecurityConfigScreen(
    currentStep: Int = 3,
    totalSteps: Int = 5,
    onComplete: () -> Unit = {},
    onBack: () -> Unit = {}
) {
    var selectedCategory by remember { mutableStateOf<DataCategoryConfig?>(null) }
    var categories by remember {
        mutableStateOf(
            listOf(
                DataCategoryConfig(
                    id = "banking",
                    name = "Banking Information",
                    description = "Account numbers, routing numbers, balances",
                    examples = listOf("Account numbers", "Routing numbers", "Balances", "Credit card numbers"),
                    riskLevel = RiskLevel.HIGH
                ),
                DataCategoryConfig(
                    id = "pii",
                    name = "Personally Identifiable Information",
                    description = "Government-issued identifiers and personal documents",
                    examples = listOf("SSN", "Driver's license", "Passport", "Tax ID"),
                    riskLevel = RiskLevel.HIGH
                ),
                DataCategoryConfig(
                    id = "medical",
                    name = "Medical Information",
                    description = "Health records and medical history",
                    examples = listOf("Diagnoses", "Prescriptions", "Lab results", "Insurance info"),
                    riskLevel = RiskLevel.HIGH
                ),
                DataCategoryConfig(
                    id = "residential",
                    name = "Residential Information",
                    description = "Physical address and contact details",
                    examples = listOf("Home address", "Phone number", "Personal email"),
                    riskLevel = RiskLevel.MEDIUM
                ),
                DataCategoryConfig(
                    id = "network",
                    name = "Network Information",
                    description = "Network identifiers and infrastructure details",
                    examples = listOf("IP address", "MAC address", "Hostname", "DNS records"),
                    riskLevel = RiskLevel.MEDIUM
                ),
                DataCategoryConfig(
                    id = "identity",
                    name = "Identity Information",
                    description = "Personal identity attributes",
                    examples = listOf("Full name", "Date of birth", "Photo", "Signature"),
                    riskLevel = RiskLevel.MEDIUM
                ),
                DataCategoryConfig(
                    id = "location",
                    name = "Location Information",
                    description = "Geographic location data",
                    examples = listOf("GPS coordinates", "City", "Country", "Timezone"),
                    riskLevel = RiskLevel.LOW
                ),
                DataCategoryConfig(
                    id = "credentials",
                    name = "Credentials",
                    description = "Authentication and access credentials",
                    examples = listOf("Usernames", "Passwords", "API keys", "Tokens"),
                    riskLevel = RiskLevel.HIGH
                )
            )
        )
    }

    val configuredCount = categories.count { it.permission != PermissionLevel.DENY }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Security Configuration") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                    }
                }
            )
        },
        bottomBar = {
            BottomAppBar {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    TextButton(onClick = onBack) {
                        Icon(Icons.Default.ArrowBack, contentDescription = null)
                        Spacer(Modifier.width(8.dp))
                        Text("Back")
                    }

                    Button(onClick = onComplete) {
                        Text("Save & Continue")
                        Spacer(Modifier.width(8.dp))
                        Icon(Icons.Default.ArrowForward, contentDescription = null)
                    }
                }
            }
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .verticalScroll(rememberScrollState())
        ) {
            // Progress indicator
            LinearProgressIndicator(
                progress = { currentStep.toFloat() / totalSteps },
                modifier = Modifier.fillMaxWidth()
            )

            Spacer(Modifier.height(16.dp))

            // Header
            Text(
                text = "Step $currentStep of $totalSteps: Data Categories",
                style = MaterialTheme.typography.titleMedium,
                modifier = Modifier.padding(horizontal = 16.dp)
            )

            Spacer(Modifier.height(8.dp))

            Text(
                text = "Configure how ArmorClaw handles your sensitive information. " +
                        "Each category can be restricted to specific websites.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(horizontal = 16.dp)
            )

            Spacer(Modifier.height(16.dp))

            // Summary card
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp)
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(16.dp),
                    horizontalArrangement = Arrangement.SpaceBetween
                ) {
                    Column {
                        Text(
                            text = "Configuration Status",
                            style = MaterialTheme.typography.titleSmall
                        )
                        Text(
                            text = "$configuredCount of ${categories.size} categories configured",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                    if (configuredCount == categories.size) {
                        Icon(
                            Icons.Default.CheckCircle,
                            contentDescription = "Complete",
                            tint = MaterialTheme.colorScheme.primary,
                            modifier = Modifier.size(32.dp)
                        )
                    } else {
                        Icon(
                            Icons.Default.Pending,
                            contentDescription = "Incomplete",
                            tint = MaterialTheme.colorScheme.secondary,
                            modifier = Modifier.size(32.dp)
                        )
                    }
                }
            }

            Spacer(Modifier.height(16.dp))

            // Category list
            categories.forEach { category ->
                CategoryCard(
                    category = category,
                    isExpanded = selectedCategory?.id == category.id,
                    onToggle = {
                        selectedCategory = if (selectedCategory?.id == category.id) null else category
                    },
                    onPermissionChange = { newPermission ->
                        categories = categories.map {
                            if (it.id == category.id) it.copy(permission = newPermission)
                            else it
                        }
                    },
                    onWebsiteAdded = { website ->
                        categories = categories.map {
                            if (it.id == category.id) {
                                it.copy(allowedWebsites = it.allowedWebsites + website)
                            } else it
                        }
                    },
                    onWebsiteRemoved = { website ->
                        categories = categories.map {
                            if (it.id == category.id) {
                                it.copy(allowedWebsites = it.allowedWebsites - website)
                            } else it
                        }
                    }
                )
                Spacer(Modifier.height(8.dp))
            }

            Spacer(Modifier.height(80.dp)) // Space for bottom bar
        }
    }
}

@Composable
fun CategoryCard(
    category: DataCategoryConfig,
    isExpanded: Boolean,
    onToggle: () -> Unit,
    onPermissionChange: (PermissionLevel) -> Unit,
    onWebsiteAdded: (String) -> Unit,
    onWebsiteRemoved: (String) -> Unit
) {
    var newWebsite by remember { mutableStateOf("") }

    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp)
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .toggleable(onClick = onToggle) { }
                .padding(16.dp)
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        RiskBadge(category.riskLevel)
                        Spacer(Modifier.width(8.dp))
                        Text(
                            text = category.name,
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold
                        )
                    }
                    Spacer(Modifier.height(4.dp))
                    Text(
                        text = category.description,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }

                Column(horizontalAlignment = Alignment.End) {
                    when (category.permission) {
                        PermissionLevel.DENY -> {
                            Icon(
                                Icons.Default.Block,
                                contentDescription = "Denied",
                                tint = MaterialTheme.colorScheme.error
                            )
                            Text(
                                text = "DENIED",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.error
                            )
                        }
                        PermissionLevel.ALLOW -> {
                            Icon(
                                Icons.Default.Check,
                                contentDescription = "Allowed",
                                tint = MaterialTheme.colorScheme.primary
                            )
                            Text(
                                text = "ALLOWED",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.primary
                            )
                        }
                        PermissionLevel.ALLOW_ALL -> {
                            Icon(
                                Icons.Default.Warning,
                                contentDescription = "Allow All",
                                tint = MaterialTheme.colorScheme.tertiary
                            )
                            Text(
                                text = "ALL",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.tertiary
                            )
                        }
                    }
                }
            }

            if (isExpanded) {
                Spacer(Modifier.height(16.dp))
                HorizontalDivider()
                Spacer(Modifier.height(16.dp))

                // Permission selection
                Text(
                    text = "Overall Permission",
                    style = MaterialTheme.typography.titleSmall
                )
                Spacer(Modifier.height(8.dp))

                PermissionSelector(
                    currentPermission = category.permission,
                    onPermissionChange = onPermissionChange
                )

                if (category.permission != PermissionLevel.DENY) {
                    Spacer(Modifier.height(16.dp))

                    // Website allowlist
                    Text(
                        text = "Allowed Websites",
                        style = MaterialTheme.typography.titleSmall
                    )
                    Text(
                        text = "This data can ONLY be used on these websites:",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Spacer(Modifier.height(8.dp))

                    // Website chips
                    if (category.allowedWebsites.isNotEmpty()) {
                        WrapChipRow(
                            items = category.allowedWebsites,
                            onRemove = onWebsiteRemoved
                        )
                        Spacer(Modifier.height(8.dp))
                    }

                    // Add website
                    OutlinedTextField(
                        value = newWebsite,
                        onValueChange = { newWebsite = it },
                        label = { Text("Add website") },
                        placeholder = { Text("example.com") },
                        trailingIcon = {
                            IconButton(
                                onClick = {
                                    if (newWebsite.isNotBlank()) {
                                        onWebsiteAdded(newWebsite)
                                        newWebsite = ""
                                    }
                                }
                            ) {
                                Icon(Icons.Default.Add, contentDescription = "Add")
                            }
                        },
                        modifier = Modifier.fillMaxWidth()
                    )

                    Spacer(Modifier.height(16.dp))

                    // Approval requirement
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            text = "Require approval for this category",
                            style = MaterialTheme.typography.bodyMedium,
                            modifier = Modifier.weight(1f)
                        )
                        Switch(
                            checked = category.requiresApproval,
                            onCheckedChange = { /* TODO */ }
                        )
                    }
                }
            }
        }
    }
}

@Composable
fun RiskBadge(riskLevel: RiskLevel) {
    val (color, text) = when (riskLevel) {
        RiskLevel.HIGH -> MaterialTheme.colorScheme.error to "HIGH"
        RiskLevel.MEDIUM -> MaterialTheme.colorScheme.tertiary to "MED"
        RiskLevel.LOW -> MaterialTheme.colorScheme.primary to "LOW"
    }

    Surface(
        color = color.copy(alpha = 0.1f),
        shape = MaterialTheme.shapes.small
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.labelSmall,
            color = color,
            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
        )
    }
}

@Composable
fun PermissionSelector(
    currentPermission: PermissionLevel,
    onPermissionChange: (PermissionLevel) -> Unit
) {
    Column {
        PermissionOption(
            selected = currentPermission == PermissionLevel.DENY,
            onClick = { onPermissionChange(PermissionLevel.DENY) },
            title = "Deny All",
            description = "Never use this type of information",
            icon = Icons.Default.Block
        )
        PermissionOption(
            selected = currentPermission == PermissionLevel.ALLOW,
            onClick = { onPermissionChange(PermissionLevel.ALLOW) },
            title = "Allow with Restrictions",
            description = "Use only on specified websites (recommended)",
            icon = Icons.Default.Check,
            recommended = true
        )
        PermissionOption(
            selected = currentPermission == PermissionLevel.ALLOW_ALL,
            onClick = { onPermissionChange(PermissionLevel.ALLOW_ALL) },
            title = "Allow All",
            description = "No restrictions (not recommended)",
            icon = Icons.Default.Warning,
            isDangerous = true
        )
    }
}

@Composable
fun PermissionOption(
    selected: Boolean,
    onClick: () -> Unit,
    title: String,
    description: String,
    icon: ImageVector,
    recommended: Boolean = false,
    isDangerous: Boolean = false
) {
    Surface(
        onClick = onClick,
        color = when {
            selected && isDangerous -> MaterialTheme.colorScheme.errorContainer
            selected -> MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.5f)
            else -> MaterialTheme.colorScheme.surface
        },
        modifier = Modifier.fillMaxWidth()
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            RadioButton(
                selected = selected,
                onClick = onClick
            )
            Spacer(Modifier.width(8.dp))
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = when {
                    isDangerous -> MaterialTheme.colorScheme.error
                    selected -> MaterialTheme.colorScheme.primary
                    else -> MaterialTheme.colorScheme.onSurfaceVariant
                }
            )
            Spacer(Modifier.width(8.dp))
            Column(modifier = Modifier.weight(1f)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = title,
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = if (selected) FontWeight.Bold else FontWeight.Normal
                    )
                    if (recommended && selected) {
                        Spacer(Modifier.width(8.dp))
                        Surface(
                            color = MaterialTheme.colorScheme.primary,
                            shape = MaterialTheme.shapes.small
                        ) {
                            Text(
                                text = "RECOMMENDED",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onPrimary,
                                modifier = Modifier.padding(horizontal = 4.dp, vertical = 2.dp)
                            )
                        }
                    }
                }
                Text(
                    text = description,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    }
}

@Composable
fun WrapChipRow(
    items: List<String>,
    onRemove: (String) -> Unit
) {
    FlowRow(
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        items.forEach { item ->
            InputChip(
                selected = false,
                onClick = { },
                label = { Text(item) },
                trailingIcon = {
                    IconButton(
                        onClick = { onRemove(item) },
                        modifier = Modifier.size(18.dp)
                    ) {
                        Icon(
                            Icons.Default.Close,
                            contentDescription = "Remove",
                            modifier = Modifier.size(14.dp)
                        )
                    }
                }
            )
        }
    }
}
