package com.armorclaw.app.screens.settings

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.armorclaw.app.util.ExternalLinkHandler

/**
 * Terms of Service screen
 *
 * Displays the app's terms of service.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TermsOfServiceScreen(
    onNavigateBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    val context = LocalContext.current
    val linkHandler = remember { ExternalLinkHandler(context) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Terms of Service") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = Icons.Filled.ArrowBack,
                            contentDescription = "Back"
                        )
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
                .verticalScroll(rememberScrollState())
                .padding(16.dp)
        ) {
            Text(
                text = "Terms of Service",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold
            )
            Text(
                text = "Last updated: January 2025",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            
            Spacer(modifier = Modifier.height(16.dp))
            
            TermsSection("1. Acceptance of Terms")
            TermsBody("By downloading, installing, or using ArmorClaw (\"the App\"), you agree to be bound by these Terms of Service. If you do not agree to these terms, please do not use the App.")
            
            TermsSection("2. Description of Service")
            TermsBody("ArmorClaw is a secure, privacy-focused Matrix client that enables end-to-end encrypted messaging, voice/video calls, and file sharing. The App uses the decentralized Matrix protocol for communication.")
            
            TermsSection("3. User Accounts")
            TermsBody("You are responsible for maintaining the confidentiality of your account credentials. You must not share your password or recovery keys with anyone. You are responsible for all activities that occur under your account.")
            
            TermsSection("4. Acceptable Use")
            TermsBody("You agree not to use the App to:")
            TermsList(
                items = listOf(
                    "Transmit illegal, harmful, or offensive content",
                    "Harass, abuse, or harm other users",
                    "Distribute malware or malicious code",
                    "Violate any applicable laws or regulations",
                    "Infringe on intellectual property rights",
                    "Attempt to compromise the security of the App or its users"
                )
            )
            
            TermsSection("5. Privacy")
            TermsBody("Your privacy is important to us. Our collection and use of personal information is governed by our Privacy Policy. By using the App, you consent to the collection and use of information as detailed in the Privacy Policy.")
            
            TermsSection("6. Encryption and Security")
            TermsBody("The App uses end-to-end encryption to protect your messages. While we strive to provide strong security:")
            TermsList(
                items = listOf(
                    "No encryption is 100% secure",
                    "You are responsible for safeguarding your device and credentials",
                    "Lost recovery keys cannot be recovered by us",
                    "We cannot access your encrypted messages"
                )
            )
            
            TermsSection("7. Third-Party Services")
            TermsBody("The App connects to the Matrix network, which includes servers operated by third parties. We are not responsible for the content, privacy practices, or availability of third-party Matrix servers.")
            
            TermsSection("8. Intellectual Property")
            TermsBody("The App is protected by copyright, trademark, and other laws. Our trademarks may not be used without prior written consent. The App is open source and released under the Apache 2.0 license.")
            
            TermsSection("9. Disclaimers")
            TermsBody("THE APP IS PROVIDED \"AS IS\" WITHOUT WARRANTY OF ANY KIND. WE DO NOT GUARANTEE THAT THE APP WILL BE UNINTERRUPTED, SECURE, OR ERROR-FREE. WE ARE NOT LIABLE FOR ANY DAMAGES ARISING FROM YOUR USE OF THE APP.")
            
            TermsSection("10. Limitation of Liability")
            TermsBody("TO THE MAXIMUM EXTENT PERMITTED BY LAW, WE SHALL NOT BE LIABLE FOR ANY INDIRECT, INCIDENTAL, SPECIAL, CONSEQUENTIAL, OR PUNITIVE DAMAGES, INCLUDING LOSS OF DATA, PROFITS, OR USE.")
            
            TermsSection("11. Changes to Terms")
            TermsBody("We may update these Terms from time to time. We will notify you of significant changes via the App or other means. Continued use after changes constitutes acceptance of the updated Terms.")
            
            TermsSection("12. Termination")
            TermsBody("We may suspend or terminate your access to the App at any time, without notice, for conduct that we believe violates these Terms or is harmful to other users, us, or the App.")
            
            TermsSection("13. Governing Law")
            TermsBody("These Terms shall be governed by the laws of the jurisdiction in which you reside, without regard to conflict of law principles.")
            
            TermsSection("14. Contact Us")
            TermsBody("If you have questions about these Terms, please contact us at:")
            
            Spacer(modifier = Modifier.height(8.dp))
            
            TextButton(
                onClick = { linkHandler.openEmail("legal@armorclaw.app", "Terms of Service Question") }
            ) {
                Text("legal@armorclaw.app")
            }
            
            Spacer(modifier = Modifier.height(24.dp))
            
            Divider()
            
            Spacer(modifier = Modifier.height(16.dp))
            
            Text(
                text = "By using ArmorClaw, you acknowledge that you have read, understood, and agree to be bound by these Terms of Service.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}

@Composable
private fun TermsSection(title: String) {
    Spacer(modifier = Modifier.height(16.dp))
    Text(
        text = title,
        style = MaterialTheme.typography.titleMedium,
        fontWeight = FontWeight.Bold
    )
    Spacer(modifier = Modifier.height(8.dp))
}

@Composable
private fun TermsBody(text: String) {
    Text(
        text = text,
        style = MaterialTheme.typography.bodyMedium
    )
}

@Composable
private fun TermsList(items: List<String>) {
    Column(
        modifier = Modifier.padding(start = 16.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        items.forEach { item ->
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                Text(
                    text = "•",
                    style = MaterialTheme.typography.bodyMedium
                )
                Text(
                    text = item,
                    style = MaterialTheme.typography.bodyMedium
                )
            }
        }
    }
}
