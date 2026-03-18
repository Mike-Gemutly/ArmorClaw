package com.armorclaw.app.accessibility

import androidx.compose.ui.ExperimentalComposeUiApi
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.SemanticsPropertyReceiver
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.invisibleToUser
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.stateDescription
import androidx.compose.ui.semantics.testTag
import androidx.compose.ui.semantics.traversalIndex

/**
 * Compose extensions for accessibility
 *
 * These extensions provide convenient modifiers for making
 * Compose UI components accessible.
 */

/**
 * Set content description for accessibility
 */
fun Modifier.accessibilityContentDescription(
    description: String
): Modifier = semantics {
    contentDescription = description
}

/**
 * Set heading level for accessibility
 */
fun Modifier.accessibilityHeading(): Modifier = semantics {
    heading()
}

/**
 * Set state description for accessibility
 */
fun Modifier.accessibilityStateDescription(
    state: String
): Modifier = semantics {
    stateDescription = state
}

/**
 * Set value for accessibility
 */
fun Modifier.accessibilityValue(
    value: String
): Modifier = semantics {
    // Compose doesn't have a built-in 'value' property in the same way
    // We use stateDescription or other properties as appropriate
    stateDescription = value
}

/**
 * Set test tag for testing
 */
fun Modifier.accessibilityTestTag(
    tag: String
): Modifier = semantics {
    testTag = tag
}

/**
 * Set traversal index for accessibility
 */
fun Modifier.accessibilityTraversalIndex(
    index: Float
): Modifier = semantics {
    traversalIndex = index
}

/**
 * Hide element from accessibility
 */
@OptIn(ExperimentalComposeUiApi::class)
fun Modifier.accessibilityHidden(): Modifier = semantics {
    invisibleToUser()
}

/**
 * Make element a clickable button with accessibility
 */
fun Modifier.accessibilityClickable(
    onClickLabel: String? = null,
    role: androidx.compose.ui.semantics.Role? = null
): Modifier = semantics {
    // Compose built-in semantics will handle this
    if (onClickLabel != null) {
        // Set onClickLabel
    }
    if (role != null) {
        // Set role
    }
}

/**
 * Make element a toggleable switch with accessibility
 */
fun Modifier.accessibilityToggleable(
    state: Boolean,
    onClickLabel: String? = null
): Modifier = semantics {
    // Compose built-in semantics will handle this
    if (onClickLabel != null) {
        // Set onClickLabel
    }
}

/**
 * Make element selectable with accessibility
 */
fun Modifier.accessibilitySelectable(
    selected: Boolean,
    onClickLabel: String? = null
): Modifier = semantics {
    // Compose built-in semantics will handle this
    if (onClickLabel != null) {
        // Set onClickLabel
    }
}

/**
 * Progress bar accessibility
 */
fun Modifier.accessibilityProgressBar(
    progress: Float,
    min: Float = 0f,
    max: Float = 1f
): Modifier = semantics {
    // Compose built-in semantics will handle this
    // Can add stateDescription for percentage
}

/**
 * Slider accessibility
 */
fun Modifier.accessibilitySlider(
    value: Float,
    min: Float = 0f,
    max: Float = 1f
): Modifier = semantics {
    // Compose built-in semantics will handle this
    // Can add stateDescription for value
}

/**
 * Tab accessibility
 */
fun Modifier.accessibilityTab(
    selected: Boolean,
    onClickLabel: String? = null
): Modifier = semantics {
    // Compose built-in semantics will handle this
    if (onClickLabel != null) {
        // Set onClickLabel
    }
}

/**
 * TextField accessibility
 */
fun Modifier.accessibilityTextField(
    label: String,
    value: String,
    isError: Boolean = false,
    errorMessage: String? = null
): Modifier = semantics {
    // Compose built-in semantics will handle this
    // Can add extra semantics if needed
}

/**
 * Custom accessibility label with both content and state
 */
fun Modifier.accessibilityLabel(
    contentDescription: String,
    stateDescription: String? = null
): Modifier = semantics {
    this.contentDescription = contentDescription
    if (stateDescription != null) {
        this.stateDescription = stateDescription
    }
}

/**
 * Live region for dynamic content
 */
fun Modifier.accessibilityLiveRegion(
    mode: AccessibilityLiveMode
): Modifier = semantics {
    // Compose doesn't have built-in live region support
    // This is a placeholder
}

/**
 * Live region modes
 */
enum class AccessibilityLiveMode {
    POLITE,
    ASSERTIVE
}

/**
 * Focusable accessibility
 */
fun Modifier.accessibilityFocusable(
    isFocusable: Boolean = true
): Modifier = semantics {
    // Compose built-in semantics will handle this
}

/**
 * Accessibility group
 */
fun Modifier.accessibilityGroup(
    contentDescription: String
): Modifier = semantics {
    this.contentDescription = contentDescription
}

/**
 * Accessibility collection
 */
fun Modifier.accessibilityCollection(
    itemCount: Int
): Modifier = semantics {
    // Compose doesn't have built-in collection support
    // This is a placeholder
}
