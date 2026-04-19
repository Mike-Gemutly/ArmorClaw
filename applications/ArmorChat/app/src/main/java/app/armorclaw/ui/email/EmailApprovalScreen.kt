package app.armorclaw.ui.email

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Cancel
import androidx.compose.material.icons.filled.Email
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import app.armorclaw.ui.components.EmailApprovalCard
import app.armorclaw.viewmodel.HitlViewModel

enum class ApprovalResult { PENDING, APPROVED, DENIED, NOT_FOUND }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun EmailApprovalScreen(
    approvalId: String,
    onBack: () -> Unit = {},
    hitlViewModel: HitlViewModel = viewModel(factory = HitlViewModel.Factory())
) {
    val pendingEmails by hitlViewModel.pendingEmails.collectAsState()
    val isLoading by hitlViewModel.isLoading.collectAsState()
    val error by hitlViewModel.error.collectAsState()
    var result by remember { mutableStateOf(ApprovalResult.PENDING) }

    val emailEvent = pendingEmails.find { it.approvalId == approvalId }

    LaunchedEffect(Unit) {
        hitlViewModel.loadAllPending()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Email Approval") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                }
            )
        }
    ) { innerPadding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(16.dp)
        ) {
            when {
                result == ApprovalResult.APPROVED -> {
                    ResultView(
                        icon = { Icon(Icons.Default.CheckCircle, contentDescription = null, tint = Color(0xFF388E3C), modifier = Modifier.size(64.dp)) },
                        title = "Approved",
                        subtitle = "The email has been approved for sending."
                    )
                }
                result == ApprovalResult.DENIED -> {
                    ResultView(
                        icon = { Icon(Icons.Default.Cancel, contentDescription = null, tint = Color(0xFFD32F2F), modifier = Modifier.size(64.dp)) },
                        title = "Denied",
                        subtitle = "The email send request has been rejected."
                    )
                }
                isLoading -> {
                    CircularProgressIndicator(
                        modifier = Modifier.align(Alignment.Center)
                    )
                }
                error != null -> {
                    Column(
                        modifier = Modifier.align(Alignment.Center),
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text(
                            text = error ?: "Unknown error",
                            style = MaterialTheme.typography.bodyLarge,
                            color = MaterialTheme.colorScheme.error,
                            textAlign = TextAlign.Center
                        )
                    }
                }
                emailEvent == null && !isLoading -> {
                    ResultView(
                        icon = { Icon(Icons.Default.Email, contentDescription = null, tint = Color(0xFF9E9E9E), modifier = Modifier.size(64.dp)) },
                        title = "Not Found",
                        subtitle = "No pending email approval with ID: ${approvalId.take(8)}…"
                    )
                }
                emailEvent != null -> {
                    Column(
                        modifier = Modifier.fillMaxWidth(),
                        verticalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        Text(
                            text = "Review the email below before approving.",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )

                        EmailApprovalCard(
                            approvalId = emailEvent.approvalId,
                            emailId = emailEvent.emailId,
                            to = emailEvent.to,
                            piiFieldCount = emailEvent.piiFields,
                            timeoutSeconds = emailEvent.timeoutS,
                            onApprove = { id ->
                                hitlViewModel.approveEmail(id)
                                result = ApprovalResult.APPROVED
                            },
                            onDeny = { id, _ ->
                                hitlViewModel.denyEmail(id)
                                result = ApprovalResult.DENIED
                            }
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun ResultView(
    icon: @Composable () -> Unit,
    title: String,
    subtitle: String
) {
    Column(
        modifier = Modifier.fillMaxSize(),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        icon()
        Spacer(modifier = Modifier.height(16.dp))
        Text(
            text = title,
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.Bold
        )
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            text = subtitle,
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center
        )
    }
}
