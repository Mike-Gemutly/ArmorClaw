package com.armorclaw.app.screens.onboarding

import android.Manifest
import android.content.pm.PackageManager
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.camera.core.*
import androidx.camera.lifecycle.ProcessCameraProvider
import androidx.camera.view.PreviewView
import androidx.compose.animation.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.content.ContextCompat
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.BrandRed
import com.armorclaw.shared.ui.theme.DesignTokens
import com.google.mlkit.vision.barcode.BarcodeScanning
import com.google.mlkit.vision.barcode.common.Barcode
import com.google.mlkit.vision.common.InputImage
import kotlinx.coroutines.launch
import java.util.concurrent.Executors

/**
 * QR Scanner screen for QR-first provisioning
 *
 * Used in onboarding flow to scan QR codes containing server configuration.
 * Supports:
 * - armorclaw://config?d=<base64> deep links
 * - https://armorclaw.app/config?d=<base64> web links
 * - Invite codes
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun QRScanScreen(
    onQRScanned: (String) -> Unit,
    onPasteLink: () -> Unit,
    onBack: () -> Unit,
    onAdvancedMode: () -> Unit,
    modifier: Modifier = Modifier
) {
    val context = LocalContext.current
    val lifecycleOwner = LocalLifecycleOwner.current
    val scope = rememberCoroutineScope()

    // Permission state
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

    // Camera preview state
    val cameraProviderFuture = remember { ProcessCameraProvider.getInstance(context) }
    val executor = remember { Executors.newSingleThreadExecutor() }

    // Barcode scanner
    val barcodeScanner = remember { BarcodeScanning.getClient() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Scan QR Code") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = Color.Black.copy(alpha = 0.7f),
                    titleContentColor = Color.White,
                    navigationIconContentColor = Color.White
                )
            )
        },
        modifier = modifier
    ) { paddingValues ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
        ) {
            when {
                // No permission
                !hasCameraPermission -> {
                    NoPermissionContent(
                        onRequestPermission = {
                            permissionLauncher.launch(Manifest.permission.CAMERA)
                        },
                        onManualEntry = onPasteLink
                    )
                }

                // Camera active
                isScanning -> {
                    // Camera preview
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
                                                        // Validate it's our format
                                                        val rawValue = barcode.rawValue
                                                        if (rawValue != null && isValidArmorClawQR(rawValue)) {
                                                            onQRScanned(rawValue)
                                                        } else {
                                                            scanError = "Invalid QR code format"
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

                                    // Flash control
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
                    ScanOverlay(
                        flashEnabled = flashEnabled,
                        onFlashToggle = { flashEnabled = !flashEnabled },
                        error = scanError,
                        onErrorDismiss = { scanError = null; isScanning = true; scannedValue = null }
                    )
                }
            }

            // Bottom actions
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
                    onClick = onPasteLink,
                    modifier = Modifier.fillMaxWidth(),
                    colors = ButtonDefaults.outlinedButtonColors(
                        contentColor = Color.White
                    )
                ) {
                    Icon(Icons.Default.ContentPaste, contentDescription = null)
                    Spacer(modifier = Modifier.width(8.dp))
                    Text("Paste Invite Link")
                }

                // Advanced mode link
                TextButton(
                    onClick = onAdvancedMode,
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

@Composable
private fun NoPermissionContent(
    onRequestPermission: () -> Unit,
    onManualEntry: () -> Unit
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(DesignTokens.Spacing.xl),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Icon(
            imageVector = Icons.Default.CameraAlt,
            contentDescription = null,
            modifier = Modifier.size(80.dp),
            tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f)
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
                    "Your camera is only used for scanning QR codes and is never recorded.",
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
private fun ScanOverlay(
    flashEnabled: Boolean,
    onFlashToggle: () -> Unit,
    error: String?,
    onErrorDismiss: () -> Unit
) {
    Box(modifier = Modifier.fillMaxSize()) {
        // Scan frame overlay
        Box(
            modifier = Modifier
                .align(Alignment.Center)
                .size(250.dp)
                .clip(RoundedCornerShape(16.dp))
                .background(Color.Transparent)
        ) {
            // Corner brackets
            ScanCorners(
                modifier = Modifier
                    .align(Alignment.Center)
                    .size(250.dp)
            )
        }

        // Instructions
        Text(
            text = "Point camera at QR code",
            style = MaterialTheme.typography.bodyLarge,
            color = Color.White,
            modifier = Modifier
                .align(Alignment.TopCenter)
                .padding(top = 120.dp)
        )

        // Flash toggle button
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
        // Top-left corner
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

        // Top-right corner
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

        // Bottom-left corner
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

        // Bottom-right corner
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

/**
 * Process camera image for barcode detection
 */
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

/**
 * Check if QR code contains valid ArmorClaw configuration
 */
private fun isValidArmorClawQR(value: String?): Boolean {
    if (value.isNullOrBlank()) return false

    return value.startsWith("armorclaw://") ||
            value.startsWith("https://armorclaw.app/") ||
            value.contains("matrix_homeserver") ||
            value.contains("rpc_url")
}
