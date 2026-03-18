package com.armorclaw.shared.ui.components.atom

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.ArmorClawShapes
import com.armorclaw.shared.ui.theme.DesignTokens
import com.armorclaw.shared.ui.theme.Outline
import com.armorclaw.shared.ui.theme.Primary
import com.armorclaw.shared.ui.theme.StatusError

enum class InputVariant {
    Outlined,
    Filled
}

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
    imeAction: androidx.compose.ui.text.input.ImeAction = androidx.compose.ui.text.input.ImeAction.Next,
    imeActionHandler: (() -> Unit)? = null,
    maxLength: Int? = null,
    showCharacterCount: Boolean = false,
    variant: InputVariant = InputVariant.Outlined,
    enabled: Boolean = true,
    readOnly: Boolean = false,
    singleLine: Boolean = true,
    maxLines: Int = 1,
    textStyle: TextStyle? = null,
    supportingText: String? = null,
    visualTransformation: VisualTransformation = VisualTransformation.None,
    errorIcon: ImageVector? = null
) {
    val isError = error != null

    Column(modifier = modifier) {
        when (variant) {
            InputVariant.Outlined -> OutlinedInputField(
                value = value,
                onValueChange = onValueChange,
                label = label,
                placeholder = placeholder,
                error = error,
                leadingIcon = leadingIcon,
                trailingIcon = trailingIcon,
                keyboardType = keyboardType,
                imeAction = imeAction,
                imeActionHandler = imeActionHandler,
                maxLength = maxLength,
                showCharacterCount = showCharacterCount,
                enabled = enabled,
                readOnly = readOnly,
                singleLine = singleLine,
                maxLines = maxLines,
                textStyle = textStyle,
                supportingText = supportingText,
                visualTransformation = visualTransformation,
                errorIcon = errorIcon
            )
            InputVariant.Filled -> FilledInputField(
                value = value,
                onValueChange = onValueChange,
                label = label,
                placeholder = placeholder,
                error = error,
                leadingIcon = leadingIcon,
                trailingIcon = trailingIcon,
                keyboardType = keyboardType,
                imeAction = imeAction,
                imeActionHandler = imeActionHandler,
                maxLength = maxLength,
                showCharacterCount = showCharacterCount,
                enabled = enabled,
                readOnly = readOnly,
                singleLine = singleLine,
                maxLines = maxLines,
                textStyle = textStyle,
                supportingText = supportingText,
                visualTransformation = visualTransformation,
                errorIcon = errorIcon
            )
        }
    }
}

@Composable
fun PasswordInputField(
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    label: String? = null,
    error: String? = null,
    enabled: Boolean = true,
    visibilityIcon: ImageVector? = null,
    visibilityOffIcon: ImageVector? = null
) {
    var passwordVisible by rememberSaveable { mutableStateOf(false) }

    InputField(
        value = value,
        onValueChange = onValueChange,
        modifier = modifier,
        label = label,
        error = error,
        keyboardType = KeyboardType.Password,
        imeAction = androidx.compose.ui.text.input.ImeAction.Done,
        trailingIcon = if (visibilityIcon != null && visibilityOffIcon != null) {
            {
                IconButton(onClick = { passwordVisible = !passwordVisible }) {
                    Icon(
                        imageVector = if (passwordVisible) visibilityOffIcon else visibilityIcon,
                        contentDescription = if (passwordVisible) "Hide password" else "Show password"
                    )
                }
            }
        } else null,
        visualTransformation = if (passwordVisible) VisualTransformation.None else PasswordVisualTransformation(),
        enabled = enabled
    )
}

@Composable
private fun OutlinedInputField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String?,
    placeholder: String?,
    error: String?,
    leadingIcon: ImageVector?,
    trailingIcon: (@Composable () -> Unit)?,
    keyboardType: KeyboardType,
    imeAction: androidx.compose.ui.text.input.ImeAction,
    imeActionHandler: (() -> Unit)?,
    maxLength: Int?,
    showCharacterCount: Boolean,
    enabled: Boolean,
    readOnly: Boolean,
    singleLine: Boolean,
    maxLines: Int,
    textStyle: TextStyle?,
    supportingText: String?,
    visualTransformation: VisualTransformation,
    errorIcon: ImageVector?
) {
    val isError = error != null

    OutlinedTextField(
        value = value,
        onValueChange = { newValue ->
            val filtered = if (maxLength != null) {
                newValue.take(maxLength)
            } else {
                newValue
            }
            onValueChange(filtered)
        },
        modifier = Modifier.fillMaxWidth(),
        label = label?.let { { Text(it) } },
        placeholder = placeholder?.let { { Text(it) } },
        leadingIcon = leadingIcon?.let {
            { Icon(imageVector = it, contentDescription = null) }
        },
        trailingIcon = {
            Row(verticalAlignment = Alignment.CenterVertically) {
                if (isError && errorIcon != null) {
                    Icon(
                        imageVector = errorIcon,
                        contentDescription = "Error",
                        tint = MaterialTheme.colorScheme.error,
                        modifier = Modifier.size(20.dp)
                    )
                }
                trailingIcon?.invoke()
            }
        },
        isError = isError,
        keyboardOptions = KeyboardOptions(
            keyboardType = keyboardType,
            imeAction = imeAction
        ),
        keyboardActions = KeyboardActions(
            onAny = { imeActionHandler?.invoke() }
        ),
        visualTransformation = visualTransformation,
        enabled = enabled,
        readOnly = readOnly,
        singleLine = singleLine,
        maxLines = maxLines,
        textStyle = textStyle ?: TextStyle.Default,
        shape = ArmorClawShapes.medium,
        colors = TextFieldDefaults.colors(
            focusedIndicatorColor = Primary,
            unfocusedIndicatorColor = Outline,
            errorIndicatorColor = StatusError,
            focusedContainerColor = Color.Transparent,
            unfocusedContainerColor = Color.Transparent
        )
    )

    SupportingText(
        error = error,
        supportingText = supportingText,
        showCharacterCount = showCharacterCount,
        currentLength = value.length,
        maxLength = maxLength
    )
}

@Composable
private fun FilledInputField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String?,
    placeholder: String?,
    error: String?,
    leadingIcon: ImageVector?,
    trailingIcon: (@Composable () -> Unit)?,
    keyboardType: KeyboardType,
    imeAction: androidx.compose.ui.text.input.ImeAction,
    imeActionHandler: (() -> Unit)?,
    maxLength: Int?,
    showCharacterCount: Boolean,
    enabled: Boolean,
    readOnly: Boolean,
    singleLine: Boolean,
    maxLines: Int,
    textStyle: TextStyle?,
    supportingText: String?,
    visualTransformation: VisualTransformation,
    errorIcon: ImageVector?
) {
    val isError = error != null

    OutlinedTextField(
        value = value,
        onValueChange = { newValue ->
            val filtered = if (maxLength != null) {
                newValue.take(maxLength)
            } else {
                newValue
            }
            onValueChange(filtered)
        },
        modifier = Modifier.fillMaxWidth(),
        label = label?.let { { Text(it) } },
        placeholder = placeholder?.let { { Text(it) } },
        leadingIcon = leadingIcon?.let {
            { Icon(imageVector = it, contentDescription = null) }
        },
        trailingIcon = {
            Row(verticalAlignment = Alignment.CenterVertically) {
                if (isError && errorIcon != null) {
                    Icon(
                        imageVector = errorIcon,
                        contentDescription = "Error",
                        tint = MaterialTheme.colorScheme.error,
                        modifier = Modifier.size(20.dp)
                    )
                }
                trailingIcon?.invoke()
            }
        },
        isError = isError,
        keyboardOptions = KeyboardOptions(
            keyboardType = keyboardType,
            imeAction = imeAction
        ),
        keyboardActions = KeyboardActions(
            onAny = { imeActionHandler?.invoke() }
        ),
        visualTransformation = visualTransformation,
        enabled = enabled,
        readOnly = readOnly,
        singleLine = singleLine,
        maxLines = maxLines,
        textStyle = textStyle ?: TextStyle.Default,
        colors = TextFieldDefaults.colors(
            focusedIndicatorColor = Primary,
            unfocusedIndicatorColor = Outline,
            errorIndicatorColor = StatusError
        )
    )

    SupportingText(
        error = error,
        supportingText = supportingText,
        showCharacterCount = showCharacterCount,
        currentLength = value.length,
        maxLength = maxLength
    )
}

@Composable
private fun SupportingText(
    error: String?,
    supportingText: String?,
    showCharacterCount: Boolean,
    currentLength: Int,
    maxLength: Int?
) {
    if (error != null || supportingText != null || showCharacterCount) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 4.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            if (error != null) {
                Text(
                    text = error,
                    color = MaterialTheme.colorScheme.error,
                    style = MaterialTheme.typography.bodySmall
                )
            } else if (supportingText != null) {
                Text(
                    text = supportingText,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                    style = MaterialTheme.typography.bodySmall
                )
            }

            if (showCharacterCount && maxLength != null && error == null) {
                Text(
                    text = "$currentLength / $maxLength",
                    color = if (currentLength > maxLength)
                        MaterialTheme.colorScheme.error
                    else
                        MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                    style = MaterialTheme.typography.bodySmall,
                    modifier = Modifier.fillMaxWidth()
                )
            }
        }
    }
}
