package com.armorclaw.app.screens.studio

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.clickable
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.runtime.*
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.armorclaw.app.studio.AgentBlockRegistry
import com.armorclaw.shared.ui.theme.DesignTokens
import kotlinx.coroutines.launch
import com.armorclaw.app.studio.BlocklyWebView

/**
 * Agent Studio Wizard Screen
 *
 * A 4-step wizard for creating and configuring AI agents with horizontal pager navigation.
 * Provides a user-friendly flow for agent creation with progressive disclosure of complexity.
 *
 * ## Architecture
 * ```
 * AgentStudioScreen
 *      ├── Scaffold (TopAppBar + Content)
 *      │       └── HorizontalPager (4 steps)
 *      │           ├── Step 0: RoleDefinitionScreen
 *      │           ├── Step 1: SkillSelectionScreen
 *      │           ├── Step 2: WorkflowBuilderScreen
 *      │           └── Step 3: PermissionsScreen
 *      └── Navigation Controls (Next/Back buttons, progress indicator)
 * ```
 *
 * ## Features
 * - Horizontal swipe navigation between steps
 * - Step progress indicator showing current position
 * - Next/Back navigation buttons with appropriate state management
 * - Material 3 design system compliance
 * - Dark mode support
 * - Accessibility with proper content descriptions
 *
 * ## State Management
 * - pagerState: Tracks current step (0-3)
 * - step data: Stores wizard data across steps
 * - navigation: Handles Next/Back button states
 *
 * ## Usage
 * ```kotlin
 * AgentStudioScreen(
 *     onAgentCreated = { agent -> /* Handle created agent */ }
 * )
 * ```
 */
@OptIn(ExperimentalFoundationApi::class, ExperimentalMaterial3Api::class)
@Composable
fun AgentStudioScreen(
    onAgentCreated: (AgentWizardData) -> Unit = {}
) {
    // Wizard state management
    val wizardData = remember { AgentWizardData() }
    val pagerState = rememberPagerState(initialPage = 0) { 4 }
    val coroutineScope = rememberCoroutineScope()
    
    // Navigation state
    val canGoBack = pagerState.currentPage > 0
    val canGoNext = pagerState.currentPage < 3 // Last step is permissions, no next button
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        text = when (pagerState.currentPage) {
                            0 -> "Step 1: Define Agent"
                            1 -> "Step 2: Select Skills"
                            2 -> "Step 3: Build Workflow"
                            3 -> "Step 4: Set Permissions"
                            else -> "Create Agent"
                        },
                        style = MaterialTheme.typography.titleLarge,
                        fontWeight = FontWeight.SemiBold
                    )
                },
                navigationIcon = {
                    if (canGoBack) {
                        IconButton(onClick = {
                            coroutineScope.launch {
                                pagerState.animateScrollToPage(pagerState.currentPage - 1)
                            }
                        }) {
                            Icon(
                                imageVector = Icons.Default.ArrowBack,
                                contentDescription = "Go to previous step"
                            )
                        }
                    }
                }
            )
        }
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
        ) {
            // Progress indicator
            ProgressIndicator(
                currentPage = pagerState.currentPage,
                totalPages = 4,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = DesignTokens.Spacing.lg, vertical = DesignTokens.Spacing.md)
            )
            
            // Horizontal pager for wizard steps
            HorizontalPager(
                state = pagerState,
                modifier = Modifier
                    .weight(1f)
                    .padding(horizontal = DesignTokens.Spacing.lg)
            ) { page ->
                when (page) {
                    0 -> RoleDefinitionScreen(
                        wizardData = wizardData,
                        modifier = Modifier.fillMaxSize()
                    )
                    1 -> SkillSelectionScreen(
                        wizardData = wizardData,
                        modifier = Modifier.fillMaxSize()
                    )
                    2 -> WorkflowBuilderScreen(
                        wizardData = wizardData,
                        modifier = Modifier.fillMaxSize()
                    )
                    3 -> PermissionsScreen(
                        wizardData = wizardData,
                        modifier = Modifier.fillMaxSize()
                    )
                }
            }
            
            // Navigation buttons
            NavigationButtons(
                canGoBack = canGoBack,
                canGoNext = canGoNext,
                onBack = {
                    coroutineScope.launch {
                        pagerState.animateScrollToPage(pagerState.currentPage - 1)
                    }
                },
                onNext = {
                    if (pagerState.currentPage < 3) {
                        coroutineScope.launch {
                            pagerState.animateScrollToPage(pagerState.currentPage + 1)
                        }
                    } else {
                        // Final step - create agent
                        onAgentCreated(wizardData)
                    }
                },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = DesignTokens.Spacing.lg, vertical = DesignTokens.Spacing.md)
            )
        }
    }
}

/**
 * Role Definition Screen - Step 1
 *
 * Allows users to define the basic properties of their agent:
 * - Agent name
 * - Agent type/category
 * - Description
 * - Avatar upload
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun RoleDefinitionScreen(
    wizardData: AgentWizardData,
    modifier: Modifier = Modifier
) {
    var agentName by remember { mutableStateOf(wizardData.agentName) }
    var agentType by remember { mutableStateOf(wizardData.agentType) }
    var description by remember { mutableStateOf(wizardData.description) }
    
    Column(modifier = modifier) {
        // Agent Name Input
        OutlinedTextField(
            value = agentName,
            onValueChange = {
                agentName = it
                wizardData.agentName = it
            },
            label = { Text("Agent Name") },
            modifier = Modifier
                .fillMaxWidth()
                .padding(vertical = DesignTokens.Spacing.md),
            singleLine = true
        )
        
        // Agent Type Selection
        ExposedDropdownMenuBox(
            expanded = wizardData.isAgentTypeExpanded,
            onExpandedChange = { wizardData.isAgentTypeExpanded = !wizardData.isAgentTypeExpanded }
        ) {
            OutlinedTextField(
                value = agentType,
                onValueChange = { /* handled by menu */ },
                label = { Text("Agent Type") },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = DesignTokens.Spacing.md),
                readOnly = true,
                trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = wizardData.isAgentTypeExpanded) }
            )
            
            ExposedDropdownMenu(
                expanded = wizardData.isAgentTypeExpanded,
                onDismissRequest = { wizardData.isAgentTypeExpanded = false }
            ) {
                listOf("Assistant", "Guardian", "Analyzer", "Scheduler").forEach { type ->
                    DropdownMenuItem(
                        text = { Text(type) },
                        onClick = {
                            agentType = type
                            wizardData.agentType = type
                            wizardData.isAgentTypeExpanded = false
                        }
                    )
                }
            }
        }
        
        // Description Input
        OutlinedTextField(
            value = description,
            onValueChange = {
                description = it
                wizardData.description = it
            },
            label = { Text("Description") },
            modifier = Modifier
                .fillMaxWidth()
                .padding(vertical = DesignTokens.Spacing.md),
            maxLines = 3
        )
        
// Avatar Upload Section
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(vertical = DesignTokens.Spacing.md)
        ) {
            Text(
                text = "Avatar",
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.Medium
            )
            
            Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))
            
            Surface(
                modifier = Modifier
                    .size(80.dp), // Larger size for better visibility and user experience
                shape = CircleShape,
                color = MaterialTheme.colorScheme.surfaceVariant
            ) {
                Box(
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        imageVector = Icons.Default.Person,
                        contentDescription = "Agent avatar placeholder",
                        modifier = Modifier.size(48.dp),
                        tint = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.5f)
                    )
                }
            }
            
            Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))
            
            TextButton(
                onClick = { /* Handle avatar upload */ },
                modifier = Modifier.align(Alignment.CenterHorizontally)
            ) {
                Text("Upload Avatar")
            }
        }
    }
}

/**
 * Skill Selection Screen - Step 2
 *
 * Allows users to select skills for their agent using dynamic SDUI forms
 * from the Bridge API. Shows available skill categories and individual skills.
 */
@Composable
private fun SkillSelectionScreen(
    wizardData: AgentWizardData,
    modifier: Modifier = Modifier
) {
    // Defensive programming: Safely handle potential null/empty AgentBlockRegistry to prevent crashes
    val allBlocks = AgentBlockRegistry.allBlocks.takeIf { it.isNotEmpty() } ?: emptyList()
    val skillCategories = remember(allBlocks) { allBlocks.groupBy { it.category } }
    
    Column(modifier = modifier) {
        Text(
            text = "Select Skills for your Agent",
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.SemiBold,
            modifier = Modifier.padding(bottom = DesignTokens.Spacing.md)
        )
        
        // Skill categories
        skillCategories.forEach { (category, blocks) ->
            Text(
                text = category.displayName,
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium,
                color = MaterialTheme.colorScheme.primary,
                modifier = Modifier.padding(vertical = DesignTokens.Spacing.md)
            )
            
            // Skill items
            blocks.forEach { block ->
                Surface(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = DesignTokens.Spacing.sm)
                        .clickable {
                            // Toggle skill selection
                            if (wizardData.selectedSkills.contains(block.type)) {
                                wizardData.selectedSkills.remove(block.type)
                            } else {
                                wizardData.selectedSkills.add(block.type)
                            }
                        },
                    shape = RoundedCornerShape(4.dp),
                    tonalElevation = DesignTokens.Elevation.xs
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(DesignTokens.Spacing.md),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            text = block.message0 ?: block.type,
                            style = MaterialTheme.typography.bodyMedium
                        )
                        
                        Checkbox(
                            checked = wizardData.selectedSkills.contains(block.type),
                            onCheckedChange = { checked ->
                                if (checked) {
                                    wizardData.selectedSkills.add(block.type)
                                } else {
                                    wizardData.selectedSkills.remove(block.type)
                                }
                            }
                        )
                    }
                }
            }
        }
    }
}

/**
 * Workflow Builder Screen - Step 3
 *
 * Embeds BlocklyWebView for visual workflow creation using drag-and-drop blocks.
 * Allows users to create automation workflows for their agent.
 */
@Composable
private fun WorkflowBuilderScreen(
    wizardData: AgentWizardData,
    modifier: Modifier = Modifier
) {
    Column(modifier = modifier) {
        Text(
            text = "Build Agent Workflow",
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.SemiBold,
            modifier = Modifier.padding(bottom = DesignTokens.Spacing.md)
        )
        
        Text(
            text = "Create visual workflows by dragging and dropping blocks",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.7f),
            modifier = Modifier.padding(bottom = DesignTokens.Spacing.lg)
        )
        
        // Embed BlocklyWebView with defensive programming for initial blocks to prevent crashes
        Box(
            modifier = Modifier
                .fillMaxSize()
                .weight(1f)
        ) {
            BlocklyWebView(
                onWorkspaceChanged = { xml ->
                    wizardData.workflowXml = xml
                },
                initialBlocks = Json.encodeToString(AgentBlockRegistry.allBlocks.takeIf { it.isNotEmpty() } ?: emptyList()),
                modifier = Modifier.fillMaxSize()
            )
        }
    }
}

/**
 * Permissions Screen - Step 4
 *
 * Configures PII access controls and sensitivity badges for the agent.
 * Allows users to set privacy and security parameters.
 */
@Composable
private fun PermissionsScreen(
    wizardData: AgentWizardData,
    modifier: Modifier = Modifier
) {
    Column(modifier = modifier) {
        Text(
            text = "Set Permissions & Privacy",
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.SemiBold,
            modifier = Modifier.padding(bottom = DesignTokens.Spacing.md)
        )
        
        Text(
            text = "Configure how your agent handles sensitive information",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.7f),
            modifier = Modifier.padding(bottom = DesignTokens.Spacing.lg)
        )
        
        // PII Access Controls
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(vertical = DesignTokens.Spacing.md)
        ) {
            Text(
                text = "PII Access Controls",
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium
            )
            
            Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))
            
            // Email access toggle
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = DesignTokens.Spacing.sm),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text("Access Email")
                Switch(
                    checked = wizardData.emailAccess,
                    onCheckedChange = { wizardData.emailAccess = it }
                )
            }
            
            // Contacts access toggle
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = DesignTokens.Spacing.sm),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text("Access Contacts")
                Switch(
                    checked = wizardData.contactsAccess,
                    onCheckedChange = { wizardData.contactsAccess = it }
                )
            }
            
            // Calendar access toggle
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = DesignTokens.Spacing.sm),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text("Access Calendar")
                Switch(
                    checked = wizardData.calendarAccess,
                    onCheckedChange = { wizardData.calendarAccess = it }
                )
            }
        }
        
        // Sensitivity Badges
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(vertical = DesignTokens.Spacing.md)
        ) {
            Text(
                text = "Sensitivity Level",
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium
            )
            
            Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))
            
            // Sensitivity options
            listOf("Low", "Medium", "High", "Critical").forEach { level ->
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = DesignTokens.Spacing.sm),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(level)
                    RadioButton(
                        selected = wizardData.sensitivityLevel == level,
                        onClick = { wizardData.sensitivityLevel = level }
                    )
                }
            }
        }
    }
}

/**
 * Progress Indicator
 *
 * Shows the current step in the wizard with numbered circles.
 */
@Composable
private fun ProgressIndicator(
    currentPage: Int,
    totalPages: Int,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        (0 until totalPages).forEach { index ->
            Column(
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                Surface(
                    modifier = Modifier.size(32.dp),
                    shape = CircleShape,
                    color = if (index == currentPage) {
                        MaterialTheme.colorScheme.primary
                    } else {
                        MaterialTheme.colorScheme.surfaceVariant
                    }
                ) {
                    Box(
                        contentAlignment = Alignment.Center
                    ) {
                        Text(
                            text = "${index + 1}",
                            style = MaterialTheme.typography.bodyMedium,
                            color = if (index == currentPage) {
                                MaterialTheme.colorScheme.onPrimary
                            } else {
                                MaterialTheme.colorScheme.onBackground.copy(alpha = 0.5f)
                            }
                        )
                    }
                }

                if (index < totalPages - 1) {
                    Spacer(modifier = Modifier.height(8.dp))
                    Divider(
                        modifier = Modifier
                            .width(2.dp)
                            .height(16.dp),
                        color = if (index < currentPage) {
                            MaterialTheme.colorScheme.primary
                        } else {
                            MaterialTheme.colorScheme.surfaceVariant
                        }
                    )
                }
            }
        }
    }
}

/**
 * Navigation Buttons
 *
 * Provides Next/Back buttons with appropriate states and actions.
 */
@Composable
private fun NavigationButtons(
    canGoBack: Boolean,
    canGoNext: Boolean,
    onBack: () -> Unit,
    onNext: () -> Unit,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        // Back button
        TextButton(
            onClick = onBack,
            enabled = canGoBack
        ) {
            Text("Back")
        }
        
        // Next button
        Button(
            onClick = onNext,
            enabled = canGoNext
        ) {
            Text(if (canGoNext) "Next" else "Create Agent")
        }
    }
}

/**
 * Wizard data model to store state across steps
 */
data class AgentWizardData(
    var agentName: String = "",
    var agentType: String = "",
    var description: String = "",
    var isAgentTypeExpanded: Boolean = false,
    var selectedSkills: MutableSet<String> = mutableSetOf(),
    var workflowXml: String = "",
    var emailAccess: Boolean = false,
    var contactsAccess: Boolean = false,
    var calendarAccess: Boolean = false,
    var sensitivityLevel: String = "Medium"
)

// ========================================
// Preview Composables
// ========================================

/**
 * Preview of AgentStudioScreen in light mode
 */
@Preview
@Composable
fun AgentStudioScreenPreviewLight() {
    MaterialTheme {
        Surface {
            AgentStudioScreen()
        }
    }
}

/**
 * Preview of AgentStudioScreen in dark mode
 */
@Preview(uiMode = android.content.res.Configuration.UI_MODE_NIGHT_YES)
@Composable
fun AgentStudioScreenPreviewDark() {
    MaterialTheme {
        Surface {
            AgentStudioScreen()
        }
    }
}

/**
 * Preview of RoleDefinitionScreen in light mode
 */
@Preview
@Composable
fun RoleDefinitionScreenPreviewLight() {
    MaterialTheme {
        Surface {
            RoleDefinitionScreen(AgentWizardData())
        }
    }
}

@Preview(uiMode = android.content.res.Configuration.UI_MODE_NIGHT_YES)
@Composable
fun RoleDefinitionScreenPreviewDark() {
    MaterialTheme {
        Surface {
            RoleDefinitionScreen(AgentWizardData())
        }
    }
}

@Preview
@Composable
fun SkillSelectionScreenPreviewLight() {
    MaterialTheme {
        Surface {
            SkillSelectionScreen(AgentWizardData())
        }
    }
}

@Preview(uiMode = android.content.res.Configuration.UI_MODE_NIGHT_YES)
@Composable
fun SkillSelectionScreenPreviewDark() {
    MaterialTheme {
        Surface {
            SkillSelectionScreen(AgentWizardData())
        }
    }
}

@Preview
@Composable
fun WorkflowBuilderScreenPreviewLight() {
    MaterialTheme {
        Surface {
            WorkflowBuilderScreen(AgentWizardData())
        }
    }
}

@Preview(uiMode = android.content.res.Configuration.UI_MODE_NIGHT_YES)
@Composable
fun WorkflowBuilderScreenPreviewDark() {
    MaterialTheme {
        Surface {
            WorkflowBuilderScreen(AgentWizardData())
        }
    }
}

@Preview
@Composable
fun PermissionsScreenPreviewLight() {
    MaterialTheme {
        Surface {
            PermissionsScreen(AgentWizardData())
        }
    }
}

@Preview(uiMode = android.content.res.Configuration.UI_MODE_NIGHT_YES)
@Composable
fun PermissionsScreenPreviewDark() {
    MaterialTheme {
        Surface {
            PermissionsScreen(AgentWizardData())
        }
    }
}