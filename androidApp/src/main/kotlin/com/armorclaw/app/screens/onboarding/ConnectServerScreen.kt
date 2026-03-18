package com.armorclaw.app.screens.onboarding

import android.Manifest
import android.content.pm.PackageManager
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.camera.core.*
import androidx.camera.lifecycle.ProcessCameraProvider
import androidx.camera.view.PreviewView
import androidx.compose.animation.*
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.content.ContextCompat
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.armorclaw.app.BuildConfig
import com.armorclaw.app.viewmodels.SetupViewModel
import com.armorclaw.app.viewmodels.SetupStep
import com.armorclaw.app.viewmodels.SetupUiState
import com.armorclaw.shared.platform.bridge.*
import com.armorclaw.app.viewmodels.BridgeHealthStatus
import com.armorclaw.app.viewmodels.BridgeHealthDetails
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.BrandRed
import com.armorclaw.shared.ui.theme.DesignTokens
import com.google.mlkit.vision.barcode.BarcodeScanning
import com.google.mlkit.vision.barcode.common.Barcode
import com.google.mlkit.vision.common.InputImage
import kotlinx.coroutines.launch
import org.koin.androidx.compose.koinViewModel
import java.util.concurrent.Executors

data class ConnectionInfo(
    val homeserver: String,
    val userId: String,
    val supportsE2EE: Boolean,
    val version: String,
    val isAdmin: Boolean = false
)

/**
 * QR-First Connect Screen
 *
 * Primary setup path using QR provisioning.
 * Manual Matrix login is hidden behind "Advanced Mode".
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ConnectServerScreen(
    onConnected: (ConnectionInfo) -> Unit,
    onBack: () -> Unit,
    viewModel: SetupViewModel = koinViewModel()
) {
    val context = LocalContext.current
    val lifecycleOwner = LocalLifecycleOwner.current
    val scope = rememberCoroutineScope()

    // UI State
    var showAdvancedMode by remember { mutableStateOf(false) }
    var showPasteDialog by remember { mutableStateOf(false) }

    // Camera permission state
    var hasCameraPermission by remember {
        mutableStateOf(
            ContextCompat.checkSelfPermission(
                context,
                Manifest.permission.CAMERA
            ) == PackageManager.PERMISSION_GRANTED
        )
    }

    // Scan state
    var isScanning by remember { mutableStateOf(true) }
    var scanError by remember { mutableStateOf<String?>(null) }
    var scannedValue by remember { mutableStateOf<String?>(null) }
    var flashEnabled by remember { mutableStateOf(false) }

    // ViewModel state
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val warnings by viewModel.securityWarnings.collectAsStateWithLifecycle()

    // Permission launcher
    val permissionLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestPermission()
    ) { granted ->
        hasCameraPermission = granted
        if (!granted) {
            scanError = "Camera permission required to scan QR codes"
        }
    }

    // Request permission on first composition
    LaunchedEffect(Unit) {
        if (!hasCameraPermission) {
            permissionLauncher.launch(Manifest.permission.CAMERA)
        }
    }

    // Handle QR scan result
    LaunchedEffect(scannedValue) {
        scannedValue?.let { qrValue ->
            println("📱 QR SCAN: qrValue=$qrValue")

            if (isValidArmorClawQR(qrValue)) {
                println("✅ QR is valid ArmorClaw format")

                // Check if QR contains auth credentials (auto-login)
                val hasAuthCredentials = qrValue.contains("username") || qrValue.contains("token") ||
                                      qrValue.contains("access_token")

                println("🔑 Has auth credentials: $hasAuthCredentials")

                if (hasAuthCredentials) {
                    // Try auto-login first
                    println("🔄 Trying auto-login...")
                    viewModel.handleQrProvisionWithAuth(qrValue, autoLogin = true)
                } else {
                    // Standard QR - require credential entry
                    println("🔄 Standard QR - calling handleQrProvision...")
                    viewModel.handleQrProvision(qrValue)
                }
            } else {
                println("❌ Invalid QR format")
                scanError = "Invalid QR code format. Please scan an ArmorClaw invite QR."
                isScanning = true
                scannedValue = null
            }
        }
    }

    // Auto-proceed when completed
    LaunchedEffect(uiState.isCompleted, uiState.completedInfo) {
        if (uiState.isCompleted && uiState.completedInfo != null) {
            val info = uiState.completedInfo!!
            onConnected(
                ConnectionInfo(
                    homeserver = uiState.serverInfo?.homeserver ?: "",
                    userId = info.userId,
                    supportsE2EE = true,
                    version = "1.6.2",
                    isAdmin = info.isAdmin
                )
            )
        }
    }

    // Camera preview state
    val cameraProviderFuture = remember { ProcessCameraProvider.getInstance(context) }
    val executor = remember { Executors.newSingleThreadExecutor() }
    val barcodeScanner = remember { BarcodeScanning.getClient() }

    // Clean up executor and barcode scanner on composition exit
    DisposableEffect(Unit) {
        onDispose {
            executor.shutdown()
            barcodeScanner.close()
        }
    }

    // Track pending credentials for two-phase connect:
    // Phase 1: startSetup (discovery + health check)
    // Phase 2: connectWithCredentials (only after canProceed becomes true)
    var pendingUsername by remember { mutableStateOf<String?>(null) }
    var pendingPassword by remember { mutableStateOf<String?>(null) }

    // Auto-proceed to credential phase when health check passes
    LaunchedEffect(uiState.canProceed, pendingUsername, pendingPassword) {
        if (uiState.canProceed && pendingUsername != null && pendingPassword != null) {
            viewModel.connectWithCredentials(pendingUsername!!, pendingPassword!!)
            pendingUsername = null
            pendingPassword = null
        }
    }

    // Track if we're showing credential form after QR scan
    val showCredentialForm = uiState.setupStep == SetupStep.READY &&
                          uiState.canProceed &&
                          uiState.serverInfo != null &&
                          !showAdvancedMode &&
                          !showPasteDialog

    println("📊 SCREEN STATE:")
    println("  - setupStep: ${uiState.setupStep}")
    println("  - canProceed: ${uiState.canProceed}")
    println("  - serverInfo: ${uiState.serverInfo}")
    println("  - isConnecting: ${uiState.isConnecting}")
    println("  - errorMessage: ${uiState.errorMessage}")
    println("  - showAdvancedMode: $showAdvancedMode")
    println("  - showPasteDialog: $showPasteDialog")
    println("  - showCredentialForm: $showCredentialForm")

    if (showCredentialForm) {
        println("✅ SHOWING CREDENTIAL FORM")
        // Show credential entry form after successful QR scan
        CredentialFormAfterQr(
            uiState = uiState,
            warnings = warnings,
            onConnect = { username, password ->
                viewModel.connectWithCredentials(username, password)
            },
            onBack = onBack,
            onDismissWarning = { viewModel.dismissWarning(it) },
            onRetryHealthCheck = { viewModel.retryHealthCheck() }
        )
    } else if (showAdvancedMode) {
        println("⚙️ SHOWING ADVANCED SETUP")
        // Show manual login form (advanced)
        AdvancedSetupScreen(
            uiState = uiState,
            warnings = warnings,
            onBack = { showAdvancedMode = false },
            onConnect = { homeserver, username, password, bridgeUrl ->
                // Store credentials for phase 2, then start phase 1
                pendingUsername = username
                pendingPassword = password
                viewModel.startSetup(homeserver, bridgeUrl.ifBlank { null })
            },
            onUseDemo = { viewModel.useDemoServer() },
            onDismissWarning = { viewModel.dismissWarning(it) },
            onRetry = { viewModel.retry() },
            onRetryHealthCheck = { viewModel.retryHealthCheck() },
            onReset = { viewModel.resetSetup() }
        )
    } else if (showPasteDialog) {
        println("📋 SHOWING PASTE DIALOG")
        // Show paste link dialog
        PasteLinkDialog(
            onDismiss = { showPasteDialog = false },
            onSubmit = { link ->
                showPasteDialog = false
                if (isValidArmorClawQR(link)) {
                    viewModel.handleQrProvision(link)
                } else {
                    scanError = "Invalid link format"
                }
            }
        )
    } else {
        println("📷 SHOWING QR SCANNER")
        // Main QR-First UI
        Scaffold(
            topBar = {
                TopAppBar(
                    title = { Text("Connect to ArmorClaw") },
                    navigationIcon = {
                        IconButton(
                            onClick = onBack,
                            enabled = !uiState.isConnecting
                        ) {
                            Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                        }
                    },
                    colors = TopAppBarDefaults.topAppBarColors(
                        containerColor = Color.Black.copy(alpha = 0.8f),
                        titleContentColor = Color.White,
                        navigationIconContentColor = Color.White
                    )
                )
            }
        ) { paddingValues ->
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(paddingValues)
            ) {
                // Camera preview (when permission granted and scanning)
                if (hasCameraPermission && isScanning && !uiState.isConnecting) {
                    AndroidView(
                        factory = { ctx ->
                            val previewView = PreviewView(ctx)

                            cameraProviderFuture.addListener({
                                val cameraProvider = cameraProviderFuture.get()

                                val preview = Preview.Builder()
                                    .build()
                                    .also {
                                        it.setSurfaceProvider(previewView.surfaceProvider)
                                    }

                                val imageAnalyzer = ImageAnalysis.Builder()
                                    .setBackpressureStrategy(ImageAnalysis.STRATEGY_KEEP_ONLY_LATEST)
                                    .build()
                                    .also { analysis ->
                                        analysis.setAnalyzer(executor) { imageProxy ->
                                            processImageForBarcode(
                                                imageProxy = imageProxy,
                                                barcodeScanner = barcodeScanner,
                                                onBarcodeDetected = { barcode ->
                                                    if (isScanning && scannedValue == null) {
                                                        scannedValue = barcode.rawValue
                                                        isScanning = false
                                                        if (!isValidArmorClawQR(barcode.rawValue)) {
                                                            scanError = "Invalid QR code format. Please scan an ArmorClaw invite QR."
                                                            isScanning = true
                                                            scannedValue = null
                                                        }
                                                    }
                                                }
                                            )
                                        }
                                    }

                                val cameraSelector = CameraSelector.DEFAULT_BACK_CAMERA

                                try {
                                    cameraProvider.unbindAll()
                                    val camera = cameraProvider.bindToLifecycle(
                                        lifecycleOwner,
                                        cameraSelector,
                                        preview,
                                        imageAnalyzer
                                    )

                                    if (camera.cameraInfo.hasFlashUnit()) {
                                        scope.launch {
                                            camera.cameraControl.enableTorch(flashEnabled)
                                        }
                                    }
                                } catch (e: Exception) {
                                    scanError = "Failed to start camera: ${e.message}"
                                }
                            }, ContextCompat.getMainExecutor(ctx))

                            previewView
                        },
                        modifier = Modifier.fillMaxSize()
                    )

                    // Scan overlay
                    QrScanOverlay(
                        flashEnabled = flashEnabled,
                        onFlashToggle = { flashEnabled = !flashEnabled },
                        error = scanError,
                        onErrorDismiss = {
                            scanError = null
                            isScanning = true
                            scannedValue = null
                        }
                    )
                } else if (!hasCameraPermission) {
                    // No permission state
                    NoPermissionState(
                        onRequestPermission = { permissionLauncher.launch(Manifest.permission.CAMERA) },
                        onManualEntry = { showPasteDialog = true }
                    )
                }

                // Processing overlay
                if (uiState.isConnecting) {
                    ProcessingOverlay(
                        step = uiState.setupStep,
                        progress = uiState.setupStep.progress,
                        error = uiState.errorMessage,
                        onRetry = { viewModel.retry() }
                    )
                }

                // Bottom actions (always visible)
                Column(
                    modifier = Modifier
                        .align(Alignment.BottomCenter)
                        .padding(DesignTokens.Spacing.lg)
                        .fillMaxWidth(),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
                ) {
                    // Paste link button
                    OutlinedButton(
                        onClick = { showPasteDialog = true },
                        modifier = Modifier.fillMaxWidth(),
                        enabled = !uiState.isConnecting,
                        colors = ButtonDefaults.outlinedButtonColors(
                            contentColor = Color.White
                        )
                    ) {
                        Icon(Icons.Default.ContentPaste, contentDescription = null)
                        Spacer(modifier = Modifier.width(8.dp))
                        Text("Paste Invite Link")
                    }

                    // Advanced mode link (for self-hosted servers)
                    TextButton(
                        onClick = { showAdvancedMode = true },
                        enabled = !uiState.isConnecting,
                        colors = ButtonDefaults.textButtonColors(
                            contentColor = Color.White.copy(alpha = 0.7f)
                        )
                    ) {
                        Text(
                            text = "Use Custom Server (Advanced)",
                            style = MaterialTheme.typography.bodySmall
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun NoPermissionState(
    onRequestPermission: () -> Unit,
    onManualEntry: () -> Unit
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background)
            .padding(DesignTokens.Spacing.xl),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Icon(
            imageVector = Icons.Default.CameraAlt,
            contentDescription = null,
            modifier = Modifier.size(80.dp),
            tint = BrandPurple
        )

        Spacer(modifier = Modifier.height(DesignTokens.Spacing.lg))

        Text(
            text = "Camera Permission Required",
            style = MaterialTheme.typography.titleLarge,
            fontWeight = FontWeight.Bold
        )

        Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))

        Text(
            text = "To scan QR codes, ArmorClaw needs access to your camera. " +
                    "Your camera is only used for scanning and is never recorded.",
            style = MaterialTheme.typography.bodyMedium,
            textAlign = TextAlign.Center,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
        )

        Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))

        Button(
            onClick = onRequestPermission,
            modifier = Modifier.fillMaxWidth()
        ) {
            Icon(Icons.Default.Camera, contentDescription = null)
            Spacer(modifier = Modifier.width(8.dp))
            Text("Grant Camera Permission")
        }

        Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))

        OutlinedButton(
            onClick = onManualEntry,
            modifier = Modifier.fillMaxWidth()
        ) {
            Icon(Icons.Default.Link, contentDescription = null)
            Spacer(modifier = Modifier.width(8.dp))
            Text("Enter Link Manually")
        }
    }
}

@Composable
private fun QrScanOverlay(
    flashEnabled: Boolean,
    onFlashToggle: () -> Unit,
    error: String?,
    onErrorDismiss: () -> Unit
) {
    Box(modifier = Modifier.fillMaxSize()) {
        // Scan frame
        Box(
            modifier = Modifier
                .align(Alignment.Center)
                .size(250.dp)
                .clip(RoundedCornerShape(16.dp))
        ) {
            ScanCorners(
                modifier = Modifier
                    .align(Alignment.Center)
                    .size(250.dp)
            )
        }

        // Instructions
        Text(
            text = "Point camera at invite QR code",
            style = MaterialTheme.typography.bodyLarge,
            color = Color.White,
            modifier = Modifier
                .align(Alignment.TopCenter)
                .padding(top = 120.dp)
        )

        // Flash toggle
        IconButton(
            onClick = onFlashToggle,
            modifier = Modifier
                .align(Alignment.TopEnd)
                .padding(DesignTokens.Spacing.md)
        ) {
            Icon(
                imageVector = if (flashEnabled) Icons.Default.FlashOn else Icons.Default.FlashOff,
                contentDescription = if (flashEnabled) "Turn off flash" else "Turn on flash",
                tint = Color.White
            )
        }

        // Error message
        AnimatedVisibility(
            visible = error != null,
            enter = fadeIn() + slideInVertically(),
            exit = fadeOut() + slideOutVertically(),
            modifier = Modifier
                .align(Alignment.BottomCenter)
                .padding(bottom = 200.dp)
        ) {
            error?.let { message ->
                Card(
                    colors = CardDefaults.cardColors(
                        containerColor = BrandRed.copy(alpha = 0.9f)
                    ),
                    modifier = Modifier.padding(horizontal = DesignTokens.Spacing.lg)
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(DesignTokens.Spacing.md),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Icon(
                            Icons.Default.Error,
                            contentDescription = null,
                            tint = Color.White
                        )
                        Spacer(modifier = Modifier.width(DesignTokens.Spacing.sm))
                        Text(
                            text = message,
                            style = MaterialTheme.typography.bodyMedium,
                            color = Color.White,
                            modifier = Modifier.weight(1f)
                        )
                        IconButton(onClick = onErrorDismiss) {
                            Icon(
                                Icons.Default.Close,
                                contentDescription = "Dismiss",
                                tint = Color.White
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun ScanCorners(modifier: Modifier = Modifier) {
    val cornerLength = 40.dp
    val cornerWidth = 4.dp
    val cornerColor = BrandGreen

    Box(modifier = modifier) {
        // Top-left
        Box(
            modifier = Modifier
                .align(Alignment.TopStart)
                .width(cornerLength)
                .height(cornerWidth)
                .background(cornerColor)
        )
        Box(
            modifier = Modifier
                .align(Alignment.TopStart)
                .width(cornerWidth)
                .height(cornerLength)
                .background(cornerColor)
        )

        // Top-right
        Box(
            modifier = Modifier
                .align(Alignment.TopEnd)
                .width(cornerLength)
                .height(cornerWidth)
                .background(cornerColor)
        )
        Box(
            modifier = Modifier
                .align(Alignment.TopEnd)
                .width(cornerWidth)
                .height(cornerLength)
                .background(cornerColor)
        )

        // Bottom-left
        Box(
            modifier = Modifier
                .align(Alignment.BottomStart)
                .width(cornerLength)
                .height(cornerWidth)
                .background(cornerColor)
        )
        Box(
            modifier = Modifier
                .align(Alignment.BottomStart)
                .width(cornerWidth)
                .height(cornerLength)
                .background(cornerColor)
        )

        // Bottom-right
        Box(
            modifier = Modifier
                .align(Alignment.BottomEnd)
                .width(cornerLength)
                .height(cornerWidth)
                .background(cornerColor)
        )
        Box(
            modifier = Modifier
                .align(Alignment.BottomEnd)
                .width(cornerWidth)
                .height(cornerLength)
                .background(cornerColor)
        )
    }
}

@Composable
private fun ProcessingOverlay(
    step: SetupStep,
    progress: Float,
    error: String?,
    onRetry: () -> Unit
) {
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(Color.Black.copy(alpha = 0.85f)),
        contentAlignment = Alignment.Center
    ) {
        Card(
            modifier = Modifier
                .fillMaxWidth()
                .padding(DesignTokens.Spacing.xl),
            colors = CardDefaults.cardColors(
                containerColor = MaterialTheme.colorScheme.surface
            )
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(DesignTokens.Spacing.lg),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
            ) {
                if (error != null) {
                    Icon(
                        Icons.Default.Error,
                        contentDescription = null,
                        tint = BrandRed,
                        modifier = Modifier.size(48.dp)
                    )
                    Text(
                        text = "Connection Failed",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold
                    )
                    Text(
                        text = error,
                        style = MaterialTheme.typography.bodyMedium,
                        textAlign = TextAlign.Center
                    )
                    Button(onClick = onRetry) {
                        Text("Try Again")
                    }
                } else {
                    CircularProgressIndicator(
                        progress = progress,
                        color = BrandPurple,
                        modifier = Modifier.size(48.dp)
                    )
                    Text(
                        text = step.displayText,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold
                    )
                    LinearProgressIndicator(
                        progress = progress,
                        modifier = Modifier.fillMaxWidth(),
                        color = BrandPurple
                    )
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun PasteLinkDialog(
    onDismiss: () -> Unit,
    onSubmit: (String) -> Unit
) {
    var linkText by remember { mutableStateOf("") }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Paste Invite Link") },
        text = {
            Column {
                Text(
                    text = "Paste the invite link you received from your server administrator.",
                    style = MaterialTheme.typography.bodyMedium
                )
                Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))
                OutlinedTextField(
                    value = linkText,
                    onValueChange = { linkText = it },
                    placeholder = { Text("armorclaw://config?d=...") },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri)
                )
            }
        },
        confirmButton = {
            Button(
                onClick = { onSubmit(linkText.trim()) },
                enabled = linkText.isNotBlank()
            ) {
                Text("Connect")
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text("Cancel")
            }
        }
    )
}

/**
 * Credential form shown after successful QR scan
 *
 * This is the primary QR-first flow:
 * 1. User scans QR → server discovered + bridge health check
 * 2. Bridge is healthy → show this credential form
 * 3. User enters credentials → connect
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun CredentialFormAfterQr(
    uiState: SetupUiState,
    warnings: List<SecurityWarning>,
    onConnect: (username: String, password: String) -> Unit,
    onBack: () -> Unit,
    onDismissWarning: (String) -> Unit,
    onRetryHealthCheck: () -> Unit
) {
    val context = LocalContext.current
    val serverInfo = uiState.serverInfo

    // If serverInfo is null, show error state instead of crashing
    if (serverInfo == null) {
        ErrorScreen(message = "Server information not available. Please scan QR code again.")
        return
    }

    var username by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var passwordVisible by remember { mutableStateOf(false) }

    // Derive server domain/IP for username placeholder
    val serverDomain = remember(serverInfo.homeserver) {
        deriveServerDomain(serverInfo.homeserver)
    }

    // Check if bridge health is blocking
    val isBridgeHealthBlocking = uiState.bridgeHealthStatus == BridgeHealthStatus.UNREACHABLE ||
                                  uiState.bridgeHealthStatus == BridgeHealthStatus.UNHEALTHY ||
                                  uiState.bridgeHealthStatus == BridgeHealthStatus.NOT_READY ||
                                  uiState.bridgeHealthStatus == BridgeHealthStatus.ERROR

    ArmorClawTheme {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = { Text("Sign In") },
                    navigationIcon = {
                        IconButton(onClick = onBack) {
                            Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                        }
                    }
                )
            }
        ) { paddingValues ->
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(paddingValues)
                    .padding(DesignTokens.Spacing.lg)
                    .verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
            ) {
                // Server info card
                Card(
                    colors = CardDefaults.cardColors(
                        containerColor = MaterialTheme.colorScheme.primaryContainer
                    )
                ) {
                    Row(
                        modifier = Modifier.padding(DesignTokens.Spacing.md),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        Icon(
                            Icons.Default.Dns,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.onPrimaryContainer,
                            modifier = Modifier.size(24.dp)
                        )
                        Column(modifier = Modifier.weight(1f)) {
                            Text(
                                text = serverInfo.displayName ?: "ArmorClaw Server",
                                style = MaterialTheme.typography.titleSmall,
                                fontWeight = FontWeight.Bold,
                                color = MaterialTheme.colorScheme.onPrimaryContainer
                            )
                            Text(
                                text = serverInfo.homeserver,
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onPrimaryContainer.copy(alpha = 0.7f)
                            )
                        }
                        Icon(
                            Icons.Default.CheckCircle,
                            contentDescription = "Connected",
                            tint = BrandGreen
                        )
                    }
                }

                Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))

                Text(
                    text = "Sign in with your ArmorClaw account",
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                )

                Spacer(modifier = Modifier.height(DesignTokens.Spacing.lg))

                // CRITICAL: Bridge health error banner (shown instead of form when blocked)
                if (isBridgeHealthBlocking && uiState.bridgeHealthMessage != null) {
                    BridgeHealthErrorBanner(
                        bridgeUrl = uiState.bridgeHealthDetails?.bridgeUrl ?: serverInfo.bridgeUrl,
                        message = uiState.bridgeHealthMessage ?: "Bridge unreachable",
                        suggestedActions = uiState.bridgeHealthDetails?.suggestedActions ?: emptyList(),
                        isIpOnlyServer = uiState.bridgeHealthDetails?.isIpOnlyServer ?: false,
                        onRetry = onRetryHealthCheck,
                        onAdvanced = { },
                        modifier = Modifier.fillMaxWidth()
                    )
                }

                // Warning banner
                Card(
                    colors = CardDefaults.cardColors(
                        containerColor = BrandPurple.copy(alpha = 0.1f)
                    ),
                    border = BorderStroke(1.dp, BrandPurple)
                ) {
                    Row(
                        modifier = Modifier.padding(DesignTokens.Spacing.md),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Icon(
                            Icons.Default.Security,
                            contentDescription = null,
                            tint = BrandPurple
                        )
                        Spacer(modifier = Modifier.width(DesignTokens.Spacing.sm))
                        Text(
                            text = "Your connection is end-to-end encrypted. Your password never leaves your device in plain text.",
                            style = MaterialTheme.typography.bodySmall,
                            color = BrandPurple
                        )
                    }
                }

                // Username field
                OutlinedTextField(
                    value = username,
                    onValueChange = { username = it },
                    label = { Text("Username") },
                    placeholder = { Text(
                        if (serverDomain.isNotEmpty()) "@username:$serverDomain"
                        else "@username:example.com"
                    )},
                    leadingIcon = { Icon(Icons.Default.Person, contentDescription = null) },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                    enabled = !uiState.isConnecting,
                    supportingText = if (serverDomain.isNotEmpty()) {
                        { Text("Matrix ID: @localpart:$serverDomain", style = MaterialTheme.typography.bodySmall) }
                    } else null
                )

                // Password field
                OutlinedTextField(
                    value = password,
                    onValueChange = { password = it },
                    label = { Text("Password") },
                    placeholder = { Text("•••••••") },
                    leadingIcon = { Icon(Icons.Default.Lock, contentDescription = null) },
                    trailingIcon = {
                        IconButton(onClick = { passwordVisible = !passwordVisible }) {
                            Icon(
                                if (passwordVisible) Icons.Default.VisibilityOff else Icons.Default.Visibility,
                                contentDescription = if (passwordVisible) "Hide" else "Show"
                            )
                        }
                    },
                    visualTransformation = if (passwordVisible) VisualTransformation.None else PasswordVisualTransformation(),
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                    enabled = !uiState.isConnecting
                )

                // Error message
                if (uiState.errorMessage != null) {
                    Card(
                        colors = CardDefaults.cardColors(
                            containerColor = BrandRed.copy(alpha = 0.1f)
                        ),
                        border = BorderStroke(1.dp, BrandRed)
                    ) {
                        Row(
                            modifier = Modifier.padding(DesignTokens.Spacing.md),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Icon(
                                Icons.Default.Error,
                                contentDescription = null,
                                tint = BrandRed
                            )
                            Spacer(modifier = Modifier.width(DesignTokens.Spacing.sm))
                            Text(
                                text = uiState.errorMessage!!,
                                style = MaterialTheme.typography.bodySmall,
                                color = BrandRed
                            )
                        }
                    }
                }

                Spacer(modifier = Modifier.weight(1f))

                // Connect button
                Button(
                    onClick = { onConnect(username, password) },
                    modifier = Modifier.fillMaxWidth(),
                    enabled = !uiState.isConnecting &&
                            username.isNotBlank() &&
                            password.isNotBlank()
                ) {
                    if (uiState.isConnecting) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(20.dp),
                            color = MaterialTheme.colorScheme.onPrimary
                        )
                    } else {
                        Text("Sign In")
                    }
                }

                // Back to scan link
                TextButton(
                    onClick = onBack,
                    enabled = !uiState.isConnecting,
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Text("Scan another QR code")
                }
            }
        }
    }
}

/**
 * Advanced setup screen for manual Matrix login
 * Only shown when user explicitly selects "Advanced Mode"
 *
 * CRITICAL (CTO Note 2026-02-23): Health gating is enforced.
 * If bridge health check fails, shows actionable error banner instead of credential form.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun AdvancedSetupScreen(
    uiState: SetupUiState,
    warnings: List<SecurityWarning>,
    onBack: () -> Unit,
    onConnect: (homeserver: String, username: String, password: String, bridgeUrl: String) -> Unit,
    onUseDemo: () -> Unit,
    onDismissWarning: (String) -> Unit,
    onRetry: () -> Unit,
    onRetryHealthCheck: () -> Unit = {},
    onReset: () -> Unit
) {
    var homeserver by remember { mutableStateOf("") }
    var username by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var passwordVisible by remember { mutableStateOf(false) }
    var showAdvanced by remember { mutableStateOf(false) }
    var customBridgeUrl by remember { mutableStateOf("") }

    // Derive server domain/IP for username placeholder
    val serverDomain = remember(homeserver) {
        deriveServerDomain(homeserver)
    }

    // Check if this is an IP-only server
    val isIpOnlyServer = remember(serverDomain) {
        isIpAddress(serverDomain)
    }

    // Auto-derive bridge URL for IP servers
    val derivedBridgeUrl = remember(homeserver, customBridgeUrl) {
        if (customBridgeUrl.isNotBlank()) customBridgeUrl
        else deriveBridgeUrlFromHomeserver(homeserver)
    }
    
    // Check if bridge health is blocking
    val isBridgeHealthBlocking = uiState.bridgeHealthStatus == BridgeHealthStatus.UNREACHABLE ||
                                  uiState.bridgeHealthStatus == BridgeHealthStatus.UNHEALTHY ||
                                  uiState.bridgeHealthStatus == BridgeHealthStatus.NOT_READY ||
                                  uiState.bridgeHealthStatus == BridgeHealthStatus.ERROR

    ArmorClawTheme {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = { Text("Custom Server Setup") },
                    navigationIcon = {
                        IconButton(onClick = onBack) {
                            Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                        }
                    }
                )
            }
        ) { paddingValues ->
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(paddingValues)
                    .padding(DesignTokens.Spacing.lg)
                    .verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
            ) {
                // CRITICAL: Bridge health error banner (shown instead of form when blocked)
                if (isBridgeHealthBlocking && uiState.bridgeHealthMessage != null) {
                    BridgeHealthErrorBanner(
                        bridgeUrl = uiState.bridgeHealthDetails?.bridgeUrl ?: uiState.serverInfo?.bridgeUrl ?: "",
                        message = uiState.bridgeHealthMessage ?: "Bridge unreachable",
                        suggestedActions = uiState.bridgeHealthDetails?.suggestedActions ?: emptyList(),
                        isIpOnlyServer = uiState.bridgeHealthDetails?.isIpOnlyServer ?: false,
                        onRetry = onRetryHealthCheck,
                        onAdvanced = { showAdvanced = true },
                        modifier = Modifier.fillMaxWidth()
                    )
                }

                // Warning banner
                Card(
                    colors = CardDefaults.cardColors(
                        containerColor = BrandRed.copy(alpha = 0.1f)
                    ),
                    border = BorderStroke(1.dp, BrandRed)
                ) {
                    Row(
                        modifier = Modifier.padding(DesignTokens.Spacing.md),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Icon(
                            Icons.Default.Warning,
                            contentDescription = null,
                            tint = BrandRed
                        )
                        Spacer(modifier = Modifier.width(DesignTokens.Spacing.sm))
                        Text(
                            text = "Advanced Mode: For self-hosted or custom servers only.",
                            style = MaterialTheme.typography.bodySmall,
                            color = BrandRed
                        )
                    }
                }

                // Server form
                OutlinedTextField(
                    value = homeserver,
                    onValueChange = { homeserver = it },
                    label = { Text("Homeserver URL") },
                    placeholder = { Text("https://matrix.example.com or http://192.168.1.100:8008") },
                    leadingIcon = { Icon(Icons.Default.Dns, contentDescription = null) },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri),
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                    enabled = !uiState.isConnecting,
                    supportingText = if (homeserver.isNotBlank() && isIpOnlyServer) {
                        { Text("IP-only server: Username format is @localpart:IP-address", color = BrandPurple) }
                    } else null
                )

                OutlinedTextField(
                    value = username,
                    onValueChange = { username = it },
                    label = { Text("Username (MXID)") },
                    placeholder = { Text(
                        if (serverDomain.isNotEmpty()) "@username:$serverDomain"
                        else "@username:example.com"
                    )},
                    leadingIcon = { Icon(Icons.Default.Person, contentDescription = null) },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                    enabled = !uiState.isConnecting,
                    supportingText = if (isIpOnlyServer) {
                        { Text("For IP servers, use format @yourname:192.168.1.100", style = MaterialTheme.typography.bodySmall) }
                    } else if (serverDomain.isNotEmpty()) {
                        { Text("Full Matrix ID: @localpart:$serverDomain", style = MaterialTheme.typography.bodySmall) }
                    } else null
                )

                OutlinedTextField(
                    value = password,
                    onValueChange = { password = it },
                    label = { Text("Password") },
                    placeholder = { Text("••••••••") },
                    leadingIcon = { Icon(Icons.Default.Lock, contentDescription = null) },
                    trailingIcon = {
                        IconButton(onClick = { passwordVisible = !passwordVisible }) {
                            Icon(
                                if (passwordVisible) Icons.Default.VisibilityOff else Icons.Default.Visibility,
                                contentDescription = if (passwordVisible) "Hide" else "Show"
                            )
                        }
                    },
                    visualTransformation = if (passwordVisible) VisualTransformation.None else PasswordVisualTransformation(),
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                    enabled = !uiState.isConnecting
                )

                // Advanced options - show by default for IP-only servers
                val showAdvancedByDefault = isIpOnlyServer && customBridgeUrl.isBlank()
                var showAdvancedLocal by remember(showAdvancedByDefault) { mutableStateOf(showAdvanced || showAdvancedByDefault) }

                TextButton(
                    onClick = { showAdvancedLocal = !showAdvancedLocal },
                    enabled = !uiState.isConnecting
                ) {
                    Icon(if (showAdvancedLocal) Icons.Default.ExpandLess else Icons.Default.ExpandMore, contentDescription = null)
                    Spacer(modifier = Modifier.width(4.dp))
                    Text("Bridge URL Override")
                    if (isIpOnlyServer && derivedBridgeUrl.isNotEmpty()) {
                        Spacer(modifier = Modifier.width(8.dp))
                        Text("(Auto-detected)", style = MaterialTheme.typography.labelSmall, color = BrandGreen)
                    }
                }

                AnimatedVisibility(visible = showAdvancedLocal) {
                    Column(verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm)) {
                        OutlinedTextField(
                            value = customBridgeUrl,
                            onValueChange = { customBridgeUrl = it },
                            label = { Text("Custom Bridge URL (optional)") },
                            placeholder = { Text(derivedBridgeUrl.ifEmpty { "http://192.168.1.100:8080" }) },
                            leadingIcon = { Icon(Icons.Default.Router, contentDescription = null) },
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri),
                            singleLine = true,
                            modifier = Modifier.fillMaxWidth(),
                            enabled = !uiState.isConnecting,
                            supportingText = if (isIpOnlyServer && derivedBridgeUrl.isNotEmpty() && customBridgeUrl.isBlank()) {
                                { Text("Auto-derived: $derivedBridgeUrl", color = BrandGreen, style = MaterialTheme.typography.bodySmall) }
                            } else null
                        )
                        if (isIpOnlyServer) {
                            Text(
                                text = "For IP-only servers, bridge URL is typically same IP with port 8080",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                            )
                        }
                    }
                }

                Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))

                // Demo server (only in DEBUG builds)
                if (BuildConfig.DEBUG) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm)
                    ) {
                        OutlinedButton(
                            onClick = onUseDemo,
                            modifier = Modifier.weight(1f),
                            enabled = !uiState.isConnecting
                        ) {
                            Icon(Icons.Default.Science, contentDescription = null)
                            Spacer(modifier = Modifier.width(4.dp))
                            Text("Demo Server")
                        }
                        OutlinedButton(
                            onClick = {
                                homeserver = "http://10.0.2.2:8008"
                                customBridgeUrl = "http://10.0.2.2:8080"
                            },
                            modifier = Modifier.weight(1f),
                            enabled = !uiState.isConnecting
                        ) {
                            Icon(Icons.Default.Router, contentDescription = null)
                            Spacer(modifier = Modifier.width(4.dp))
                            Text("Local Dev")
                        }
                    }
                }

                Spacer(modifier = Modifier.weight(1f))

                // Connect button
                Button(
                    onClick = { 
                        // Use derived bridge URL if custom URL is blank for IP-only servers
                        val bridgeUrlToUse = customBridgeUrl.ifBlank { derivedBridgeUrl }
                        onConnect(homeserver, username, password, bridgeUrlToUse) 
                    },
                    modifier = Modifier.fillMaxWidth(),
                    enabled = !uiState.isConnecting &&
                            homeserver.isNotBlank() &&
                            username.isNotBlank() &&
                            password.isNotBlank()
                ) {
                    if (uiState.isConnecting) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(20.dp),
                            color = MaterialTheme.colorScheme.onPrimary
                        )
                    } else {
                        Text("Connect")
                    }
                }
            }
        }
    }
}

// Helper functions

/**
 * Bridge Health Error Banner - CTO Requirement (2026-02-23)
 * 
 * Shows actionable error message when bridge is unreachable.
 * This gates credential entry until bridge health check passes.
 */
@Composable
private fun BridgeHealthErrorBanner(
    bridgeUrl: String,
    message: String,
    suggestedActions: List<String>,
    isIpOnlyServer: Boolean,
    onRetry: () -> Unit,
    onAdvanced: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier,
        colors = CardDefaults.cardColors(
            containerColor = BrandRed.copy(alpha = 0.1f)
        ),
        border = BorderStroke(1.dp, BrandRed)
    ) {
        Column(
            modifier = Modifier.padding(DesignTokens.Spacing.md),
            verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm)
        ) {
            // Header
            Row(
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    Icons.Default.Error,
                    contentDescription = null,
                    tint = BrandRed,
                    modifier = Modifier.size(24.dp)
                )
                Spacer(modifier = Modifier.width(DesignTokens.Spacing.sm))
                Text(
                    text = "Bridge Unreachable",
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    color = BrandRed
                )
            }
            
            // Bridge URL
            Text(
                text = "Could not connect to: $bridgeUrl",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.8f)
            )
            
            // Error message
            if (message.isNotBlank()) {
                Text(
                    text = message,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                )
            }
            
            // Suggested actions
            if (suggestedActions.isNotEmpty()) {
                Spacer(modifier = Modifier.height(DesignTokens.Spacing.xs))
                Text(
                    text = "Suggested actions:",
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.Bold
                )
                Column(
                    modifier = Modifier.padding(start = DesignTokens.Spacing.sm),
                    verticalArrangement = Arrangement.spacedBy(2.dp)
                ) {
                    suggestedActions.take(4).forEach { action ->
                        Row(verticalAlignment = Alignment.Top) {
                            Text(
                                text = "•",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                            )
                            Spacer(modifier = Modifier.width(DesignTokens.Spacing.xs))
                            Text(
                                text = action,
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                            )
                        }
                    }
                }
            }
            
            // Special hint for IP-only servers
            if (isIpOnlyServer) {
                Spacer(modifier = Modifier.height(DesignTokens.Spacing.xs))
                Surface(
                    color = BrandPurple.copy(alpha = 0.1f),
                    shape = RoundedCornerShape(4.dp)
                ) {
                    Row(
                        modifier = Modifier.padding(DesignTokens.Spacing.sm),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Icon(
                            Icons.Default.Info,
                            contentDescription = null,
                            tint = BrandPurple,
                            modifier = Modifier.size(16.dp)
                        )
                        Spacer(modifier = Modifier.width(DesignTokens.Spacing.xs))
                        Text(
                            text = "IP-only servers require bridge v8.1.1+ with socket fix",
                            style = MaterialTheme.typography.labelSmall,
                            color = BrandPurple
                        )
                    }
                }
            }
            
            // Action buttons
            Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm)
            ) {
                Button(
                    onClick = onRetry,
                    modifier = Modifier.weight(1f)
                ) {
                    Icon(Icons.Default.Refresh, contentDescription = null, modifier = Modifier.size(18.dp))
                    Spacer(modifier = Modifier.width(4.dp))
                    Text("Retry")
                }
                OutlinedButton(
                    onClick = onAdvanced,
                    modifier = Modifier.weight(1f)
                ) {
                    Icon(Icons.Default.Settings, contentDescription = null, modifier = Modifier.size(18.dp))
                    Spacer(modifier = Modifier.width(4.dp))
                    Text("Advanced")
                }
            }
        }
    }
}

@androidx.camera.core.ExperimentalGetImage
private fun processImageForBarcode(
    imageProxy: ImageProxy,
    barcodeScanner: com.google.mlkit.vision.barcode.BarcodeScanner,
    onBarcodeDetected: (Barcode) -> Unit
) {
    val mediaImage = imageProxy.image
    if (mediaImage != null) {
        val inputImage = InputImage.fromMediaImage(
            mediaImage,
            imageProxy.imageInfo.rotationDegrees
        )

        barcodeScanner.process(inputImage)
            .addOnSuccessListener { barcodes ->
                barcodes.firstOrNull()?.let { barcode ->
                    onBarcodeDetected(barcode)
                }
            }
            .addOnCompleteListener {
                imageProxy.close()
            }
    } else {
        imageProxy.close()
    }
}

private fun isValidArmorClawQR(value: String?): Boolean {
    if (value.isNullOrBlank()) return false

    return value.startsWith("armorclaw://") ||
            value.startsWith("https://armorclaw.app/") ||
            value.contains("matrix_homeserver") ||
            value.contains("rpc_url")
}

/**
 * Derive server domain/IP from homeserver URL for MXID placeholder
 */
private fun deriveServerDomain(homeserver: String): String {
    if (homeserver.isBlank()) return ""

    return homeserver
        .removePrefix("https://")
        .removePrefix("http://")
        .removeSuffix("/")
        .split("/").first()
        .split(":").first() // Remove port if present for display
}

/**
 * Check if the given string is an IP address (IPv4)
 * Delegates to shared NetworkUtils to avoid code duplication (RC-05).
 */
private fun isIpAddress(domain: String): Boolean = com.armorclaw.shared.platform.network.NetworkUtils.isIpAddress(domain)

/**
 * Derive bridge URL from homeserver URL
 * For IP-only servers: http://IP:8008 -> http://IP:8080
 * For domain servers: https://matrix.example.com -> https://bridge.example.com
 */
private fun deriveBridgeUrlFromHomeserver(homeserver: String): String {
    if (homeserver.isBlank()) return ""

    val url = homeserver.removeSuffix("/")
    val protocol = when {
        url.startsWith("https://") -> "https://"
        url.startsWith("http://") -> "http://"
        else -> "https://"
    }

    val hostPart = url
        .removePrefix("https://")
        .removePrefix("http://")
        .split("/").first()

    // Extract host and port
    val (host, port) = if (hostPart.contains(":")) {
        val parts = hostPart.split(":")
        parts[0] to parts[1].toIntOrNull()
    } else {
        hostPart to null
    }

    // Check if it's an IP address
    val isIp = isIpAddress(host)

    return if (isIp) {
        // For IP-only servers, use port 8080 for bridge
        val bridgePort = 8080
        "$protocol$host:$bridgePort"
    } else {
        // For domain servers, derive bridge subdomain
        val bridgeHost = when {
            host.contains("matrix.") -> host.replace("matrix.", "bridge.")
            host.contains("chat.") -> host.replace("chat.", "bridge.")
            else -> "bridge.$host"
        }
        "$protocol$bridgeHost"
    }
}

/**
 * Simple error screen component
 */
@Composable
private fun ErrorScreen(message: String) {
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background)
            .padding(DesignTokens.Spacing.xl),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
        ) {
            Icon(
                Icons.Default.Error,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.error,
                modifier = Modifier.size(64.dp)
            )
            Text(
                text = "Error",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.error
            )
            Text(
                text = message,
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f),
                textAlign = TextAlign.Center
            )
        }
    }
}
