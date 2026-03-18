package com.armorclaw.shared.ui.components.atom

import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonColors
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowForward
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.armorclaw.shared.ui.theme.DesignTokens
import com.armorclaw.shared.ui.theme.OnPrimary
import com.armorclaw.shared.ui.theme.OnSecondary
import com.armorclaw.shared.ui.theme.Primary
import com.armorclaw.shared.ui.theme.Secondary

enum class ButtonVariant {
    Primary,
    Secondary,
    Outline,
    Text,
    Ghost
}

enum class ButtonSize {
    Small,
    Medium,
    Large
}

@Composable
fun ArmorClawButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    variant: ButtonVariant = ButtonVariant.Primary,
    size: ButtonSize = ButtonSize.Medium,
    enabled: Boolean = true,
    loading: Boolean = false,
    leadingIcon: ImageVector? = null,
    trailingIcon: ImageVector? = null,
    shape: Shape? = null
) {
    val colors = getButtonColors(variant)
    val buttonSize = getButtonSize(size)

    Button(
        onClick = onClick,
        modifier = modifier.height(buttonSize.minHeight),
        enabled = enabled && !loading,
        colors = colors,
        shape = shape ?: com.armorclaw.shared.ui.theme.ArmorClawShapes.medium,
        contentPadding = buttonSize.contentPadding
    ) {
        ButtonContent(
            text = text,
            loading = loading,
            leadingIcon = leadingIcon,
            trailingIcon = trailingIcon,
            iconSize = buttonSize.iconSize,
            textSize = buttonSize.textSize
        )
    }
}

@Composable
fun ArmorClawTextButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    loading: Boolean = false,
    leadingIcon: ImageVector? = null,
    trailingIcon: ImageVector? = null
) {
    TextButton(
        onClick = onClick,
        modifier = modifier,
        enabled = enabled && !loading,
        contentPadding = PaddingValues(
            horizontal = DesignTokens.Spacing.md,
            vertical = DesignTokens.Spacing.sm
        )
    ) {
        ButtonContent(
            text = text,
            loading = loading,
            leadingIcon = leadingIcon,
            trailingIcon = trailingIcon,
            iconSize = DesignTokens.Button.iconSize,
            textSize = 14.sp
        )
    }
}

@Composable
fun PrimaryButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    loading: Boolean = false
) {
    ArmorClawButton(
        text = text,
        onClick = onClick,
        modifier = modifier,
        variant = ButtonVariant.Primary,
        enabled = enabled,
        loading = loading
    )
}

@Composable
fun SecondaryButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    loading: Boolean = false
) {
    ArmorClawButton(
        text = text,
        onClick = onClick,
        modifier = modifier,
        variant = ButtonVariant.Secondary,
        enabled = enabled,
        loading = loading
    )
}

@Composable
fun OutlinedButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    loading: Boolean = false
) {
    ArmorClawButton(
        text = text,
        onClick = onClick,
        modifier = modifier,
        variant = ButtonVariant.Outline,
        enabled = enabled,
        loading = loading
    )
}

@Composable
fun GhostButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    loading: Boolean = false
) {
    ArmorClawButton(
        text = text,
        onClick = onClick,
        modifier = modifier,
        variant = ButtonVariant.Ghost,
        enabled = enabled,
        loading = loading
    )
}

@Composable
private fun ButtonContent(
    text: String,
    loading: Boolean,
    leadingIcon: ImageVector?,
    trailingIcon: ImageVector?,
    iconSize: androidx.compose.ui.unit.Dp,
    textSize: androidx.compose.ui.unit.TextUnit
) {
    if (loading) {
        CircularProgressIndicator(
            modifier = Modifier.size(iconSize),
            strokeWidth = 2.dp,
            color = MaterialTheme.colorScheme.onPrimary
        )
    } else {
        if (leadingIcon != null) {
            Icon(
                imageVector = leadingIcon,
                contentDescription = null,
                modifier = Modifier.size(iconSize)
            )
            Spacer(modifier = Modifier.width(DesignTokens.Spacing.sm))
        }

        Text(
            text = text,
            style = MaterialTheme.typography.labelLarge.copy(
                fontSize = textSize
            )
        )

        if (trailingIcon != null) {
            Spacer(modifier = Modifier.width(DesignTokens.Spacing.sm))
            Icon(
                imageVector = trailingIcon,
                contentDescription = null,
                modifier = Modifier.size(iconSize)
            )
        }
    }
}

@Composable
private fun getButtonColors(variant: ButtonVariant): ButtonColors {
    return when (variant) {
        ButtonVariant.Primary -> ButtonDefaults.buttonColors(
            containerColor = Primary,
            contentColor = OnPrimary
        )
        ButtonVariant.Secondary -> ButtonDefaults.buttonColors(
            containerColor = Secondary,
            contentColor = OnSecondary
        )
        ButtonVariant.Outline -> ButtonDefaults.outlinedButtonColors(
            containerColor = androidx.compose.ui.graphics.Color.Transparent,
            contentColor = Primary
        )
        ButtonVariant.Text -> ButtonDefaults.textButtonColors(
            contentColor = Primary
        )
        ButtonVariant.Ghost -> ButtonDefaults.textButtonColors(
            contentColor = Primary
        )
    }
}

private data class ButtonSizeConfig(
    val minHeight: androidx.compose.ui.unit.Dp,
    val contentPadding: PaddingValues,
    val iconSize: androidx.compose.ui.unit.Dp,
    val textSize: androidx.compose.ui.unit.TextUnit
)

@Composable
private fun getButtonSize(size: ButtonSize): ButtonSizeConfig {
    return when (size) {
        ButtonSize.Small -> ButtonSizeConfig(
            minHeight = DesignTokens.Button.smallHeight,
            contentPadding = PaddingValues(
                horizontal = DesignTokens.Spacing.md,
                vertical = DesignTokens.Spacing.xs
            ),
            iconSize = DesignTokens.Button.smallIconSize,
            textSize = 12.sp
        )
        ButtonSize.Medium -> ButtonSizeConfig(
            minHeight = DesignTokens.Button.minHeight,
            contentPadding = PaddingValues(
                horizontal = DesignTokens.Spacing.lg,
                vertical = DesignTokens.Spacing.sm
            ),
            iconSize = DesignTokens.Button.iconSize,
            textSize = 14.sp
        )
        ButtonSize.Large -> ButtonSizeConfig(
            minHeight = DesignTokens.Button.largeHeight,
            contentPadding = PaddingValues(
                horizontal = DesignTokens.Spacing.xl,
                vertical = DesignTokens.Spacing.md
            ),
            iconSize = DesignTokens.Button.largeIconSize,
            textSize = 16.sp
        )
    }
}
