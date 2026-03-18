package com.armorclaw.shared.ui.components.molecule

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.BrandGreenDark
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.BrandPurpleDark
import com.armorclaw.shared.ui.theme.BrandPurpleLight
import com.armorclaw.shared.ui.theme.BrandRed
import com.armorclaw.shared.ui.theme.BrandRedDark
import com.armorclaw.shared.ui.theme.DesignTokens
import com.armorclaw.shared.ui.theme.OnBackground
import com.armorclaw.shared.ui.theme.Outline

@Composable
fun ArmorClawCard(
    modifier: Modifier = Modifier,
    onClick: (() -> Unit)? = null,
    elevation: Dp = DesignTokens.Elevation.sm,
    shape: Shape = com.armorclaw.shared.ui.theme.ArmorClawShapes.medium,
    containerColor: Color = MaterialTheme.colorScheme.surface,
    contentColor: Color = MaterialTheme.colorScheme.onSurface,
    border: BorderStroke? = null,
    borderColor: Color? = null,
    content: @Composable ColumnScope.() -> Unit
) {
    val actualBorder = border ?: borderColor?.let { BorderStroke(1.dp, it) }

    Card(
        modifier = modifier
            .then(
                if (onClick != null) {
                    Modifier.clickable { onClick() }
                } else {
                    Modifier
                }
            ),
        shape = shape,
        colors = CardDefaults.cardColors(
            containerColor = containerColor,
            contentColor = contentColor
        ),
        elevation = CardDefaults.cardElevation(
            defaultElevation = elevation
        ),
        border = actualBorder
    ) {
        Column(
            modifier = Modifier.padding(DesignTokens.Spacing.md),
            content = content
        )
    }
}

@Composable
fun OutlinedCard(
    modifier: Modifier = Modifier,
    onClick: (() -> Unit)? = null,
    shape: Shape = com.armorclaw.shared.ui.theme.ArmorClawShapes.medium,
    borderColor: Color = Outline,
    content: @Composable ColumnScope.() -> Unit
) {
    ArmorClawCard(
        modifier = modifier,
        onClick = onClick,
        elevation = 0.dp,
        shape = shape,
        containerColor = Color.Transparent,
        border = BorderStroke(1.dp, borderColor),
        content = content
    )
}

@Composable
fun ElevatedCard(
    modifier: Modifier = Modifier,
    onClick: (() -> Unit)? = null,
    elevation: Dp = DesignTokens.Elevation.md,
    content: @Composable ColumnScope.() -> Unit
) {
    ArmorClawCard(
        modifier = modifier,
        onClick = onClick,
        elevation = elevation,
        content = content
    )
}

@Composable
fun InfoCard(
    title: String,
    message: String,
    modifier: Modifier = Modifier,
    icon: ImageVector? = null
) {
    ArmorClawCard(
        modifier = modifier,
        containerColor = BrandPurpleLight.copy(alpha = 0.3f),
        borderColor = BrandPurple
    ) {
        if (icon != null) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = BrandPurpleDark,
                modifier = Modifier.padding(bottom = DesignTokens.Spacing.sm)
            )
        }
        Text(
            text = title,
            style = MaterialTheme.typography.titleMedium,
            color = BrandPurpleDark,
            modifier = Modifier.padding(bottom = DesignTokens.Spacing.xs)
        )
        Text(
            text = message,
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground
        )
    }
}

@Composable
fun SuccessCard(
    title: String,
    message: String,
    modifier: Modifier = Modifier
) {
    ArmorClawCard(
        modifier = modifier,
        containerColor = BrandGreen.copy(alpha = 0.1f),
        borderColor = BrandGreen
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.titleMedium,
            color = BrandGreenDark,
            modifier = Modifier.padding(bottom = DesignTokens.Spacing.xs)
        )
        Text(
            text = message,
            style = MaterialTheme.typography.bodyMedium
        )
    }
}

@Composable
fun ErrorCard(
    title: String,
    message: String,
    modifier: Modifier = Modifier
) {
    ArmorClawCard(
        modifier = modifier,
        containerColor = BrandRed.copy(alpha = 0.1f),
        borderColor = BrandRed
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.titleMedium,
            color = BrandRedDark,
            modifier = Modifier.padding(bottom = DesignTokens.Spacing.xs)
        )
        Text(
            text = message,
            style = MaterialTheme.typography.bodyMedium
        )
    }
}
