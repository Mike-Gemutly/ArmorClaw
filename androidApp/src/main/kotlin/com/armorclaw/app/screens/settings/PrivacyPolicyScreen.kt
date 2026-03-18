package com.armorclaw.app.screens.settings
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.ui.tooling.preview.Preview

import androidx.compose.material3.MaterialTheme

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.ArmorClawTheme

/**
 * Privacy policy screen
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PrivacyPolicyScreen(
    onNavigateBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    val scrollState = rememberScrollState()
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Privacy Policy") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.Default.ArrowBack, "Back")
                    }
                }
            )
        },
        modifier = modifier
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .verticalScroll(scrollState)
                .padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Last updated
            Text(
                text = "Last Updated: February 10, 2026",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f)
            )
            
            Divider()
            
            // Introduction
            Text(
                text = "Introduction",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold
            )
            Text(
                text = "Welcome to ArmorClaw. We respect your privacy and are committed to protecting your personal data. This privacy policy explains how we collect, use, and protect your data.",
                style = MaterialTheme.typography.bodyMedium
            )
            
            Divider()
            
            // Data Collection
            Text(
                text = "Data Collection",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold
            )
            Text(
                text = "We collect the following data:",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "• Account information (username, email)",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "• Profile information (name, avatar, status)",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "• Messages and conversations (encrypted)",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "• Files and attachments (encrypted)",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "• Usage data (app analytics)",
                style = MaterialTheme.typography.bodyMedium
            )
            
            Divider()
            
            // End-to-End Encryption
            Text(
                text = "End-to-End Encryption",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold
            )
            Text(
                text = "All messages in ArmorClaw are end-to-end encrypted using AES-256-GCM. This means that only you and the recipients can read your messages. We cannot access your messages.",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "Encryption keys are stored on your device and never shared with us or any third parties.",
                style = MaterialTheme.typography.bodyMedium
            )
            
            Divider()
            
            // Data Usage
            Text(
                text = "How We Use Your Data",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold
            )
            Text(
                text = "We use your data to:",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "• Provide and improve our service",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "• Authenticate you and prevent fraud",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "• Send you notifications (if enabled)",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "• Analyze usage patterns to improve performance",
                style = MaterialTheme.typography.bodyMedium
            )
            
            Divider()
            
            // Data Sharing
            Text(
                text = "Data Sharing",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold
            )
            Text(
                text = "We do not sell, rent, or share your personal data with third parties for marketing purposes. We only share your data:",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "• When required by law",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "• To protect our rights and property",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "• To prevent fraud or abuse",
                style = MaterialTheme.typography.bodyMedium
            )
            
            Divider()
            
            // Data Storage
            Text(
                text = "Data Storage",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold
            )
            Text(
                text = "Your messages are stored on your device in an encrypted database (SQLCipher). When you send a message, it is also stored on our servers in encrypted form.",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "You can delete your messages at any time. When you delete your account, all your data is permanently deleted within 30 days.",
                style = MaterialTheme.typography.bodyMedium
            )
            
            Divider()
            
            // Your Rights
            Text(
                text = "Your Rights",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold
            )
            Text(
                text = "You have the right to:",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "• Access your personal data",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "• Correct inaccurate data",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "• Delete your personal data",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "• Export your personal data",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "• Opt-out of data collection",
                style = MaterialTheme.typography.bodyMedium
            )
            
            Divider()
            
            // Contact
            Text(
                text = "Contact Us",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold
            )
            Text(
                text = "If you have any questions about this privacy policy or our data practices, please contact us at:",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                text = "Email: privacy@armorclaw.app",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.primary
            )
        }
    }
}

@Preview(showBackground = true)
@Composable
private fun PrivacyPolicyScreenPreview() {
    ArmorClawTheme {
        PrivacyPolicyScreen(onNavigateBack = {})
    }
}
