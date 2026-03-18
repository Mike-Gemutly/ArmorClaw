package com.armorclaw.app.screens.settings

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.app.util.ExternalLinkHandler

/**
 * Open source licenses screen
 *
 * Displays a list of all open source libraries used in the app.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OpenSourceLicensesScreen(
    onNavigateBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    val context = LocalContext.current
    val linkHandler = remember { ExternalLinkHandler(context) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Open Source Licenses") },
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
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues),
            contentPadding = PaddingValues(vertical = 8.dp)
        ) {
            // Header
            item {
                Column(
                    modifier = Modifier.padding(16.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Code,
                        contentDescription = null,
                        modifier = Modifier.size(48.dp),
                        tint = MaterialTheme.colorScheme.primary
                    )
                    Spacer(modifier = Modifier.height(16.dp))
                    Text(
                        text = "This app uses the following open source libraries",
                        style = MaterialTheme.typography.bodyLarge,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                Divider()
            }

            // License items
            items(OssLicenses.list) { license ->
                OssLicenseItem(
                    license = license,
                    onClick = {
                        license.url?.let { linkHandler.openInCustomTab(it) }
                    }
                )
            }

            // Footer
            item {
                Spacer(modifier = Modifier.height(16.dp))
                Divider()
                Column(
                    modifier = Modifier.padding(16.dp)
                ) {
                    Text(
                        text = "Thank you to all the open source contributors!",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    TextButton(onClick = { linkHandler.openInCustomTab("https://opensource.org") }) {
                        Icon(
                            imageVector = Icons.Default.OpenInNew,
                            contentDescription = null,
                            modifier = Modifier.size(18.dp)
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                        Text("Learn about open source")
                    }
                }
            }
        }
    }
}

@Composable
private fun OssLicenseItem(
    license: OssLicense,
    onClick: () -> Unit
) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        onClick = onClick,
        enabled = license.url != null
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp)
        ) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = license.name,
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.Bold,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis
                    )
                    Text(
                        text = license.license,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                if (license.url != null) {
                    Icon(
                        imageVector = Icons.Default.OpenInNew,
                        contentDescription = null,
                        modifier = Modifier.size(18.dp),
                        tint = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }
        }
    }
}

/**
 * Open source license data
 */
data class OssLicense(
    val name: String,
    val license: String,
    val url: String? = null,
    val copyright: String? = null
)

/**
 * List of open source licenses used in the app
 */
object OssLicenses {
    val list = listOf(
        // Android/Compose
        OssLicense(
            name = "AndroidX AppCompat",
            license = "Apache 2.0",
            url = "https://developer.android.com/jetpack/androidx"
        ),
        OssLicense(
            name = "Jetpack Compose",
            license = "Apache 2.0",
            url = "https://developer.android.com/jetpack/compose"
        ),
        OssLicense(
            name = "Material Design Components",
            license = "Apache 2.0",
            url = "https://material.io/develop/android"
        ),
        
        // Kotlin
        OssLicense(
            name = "Kotlin",
            license = "Apache 2.0",
            url = "https://kotlinlang.org"
        ),
        OssLicense(
            name = "KotlinX Coroutines",
            license = "Apache 2.0",
            url = "https://github.com/Kotlin/kotlinx.coroutines"
        ),
        OssLicense(
            name = "KotlinX Serialization",
            license = "Apache 2.0",
            url = "https://github.com/Kotlin/kotlinx.serialization"
        ),
        OssLicense(
            name = "KotlinX DateTime",
            license = "Apache 2.0",
            url = "https://github.com/Kotlin/kotlinx-datetime"
        ),
        
        // Networking
        OssLicense(
            name = "Ktor Client",
            license = "Apache 2.0",
            url = "https://ktor.io"
        ),
        OssLicense(
            name = "OkHttp",
            license = "Apache 2.0",
            url = "https://square.github.io/okhttp"
        ),
        
        // Image Loading
        OssLicense(
            name = "Coil",
            license = "Apache 2.0",
            url = "https://coil-kt.github.io/coil"
        ),
        
        // DI
        OssLicense(
            name = "Koin",
            license = "Apache 2.0",
            url = "https://insert-koin.io"
        ),
        
        // Security
        OssLicense(
            name = "Bouncy Castle",
            license = "MIT",
            url = "https://www.bouncycastle.org"
        ),
        
        // Matrix
        OssLicense(
            name = "Matrix SDK",
            license = "Apache 2.0",
            url = "https://matrix.org/docs/sdk"
        ),
        OssLicense(
            name = "LibOlm",
            license = "Apache 2.0",
            url = "https://gitlab.matrix.org/matrix-org/olm"
        ),
        
        // UI Components
        OssLicense(
            name = "Accompanist",
            license = "Apache 2.0",
            url = "https://google.github.io/accompanist"
        ),
        
        // Testing
        OssLicense(
            name = "JUnit",
            license = "Eclipse Public License 1.0",
            url = "https://junit.org"
        ),
        OssLicense(
            name = "MockK",
            license = "Apache 2.0",
            url = "https://mockk.io"
        ),
        OssLicense(
            name = "Turbine",
            license = "Apache 2.0",
            url = "https://github.com/cashapp/turbine"
        )
    )
}
