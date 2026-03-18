package com.armorclaw.app.studio

import android.annotation.SuppressLint
import android.content.Context
import android.webkit.JavascriptInterface
import android.webkit.WebChromeClient
import android.webkit.WebResourceError
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.MutableState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.DesignTokens
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable

/**
 * WebView-based Blockly integration for Agent Studio visual workflow builder.
 *
 * Features:
 * - Loads Blockly from local assets or configurable URL
 * - Bidirectional JavaScript bridge for workspace management
 * - Lifecycle-aware WebView management
 * - Memory leak prevention
 * - Error handling with graceful fallbacks
 *
 * @param onWorkspaceChanged Callback when workspace content changes
 * @param initialBlocks Optional initial blocks to inject on load
 * @param toolboxXml Optional custom toolbox XML configuration
 * @param blocklyUrl URL to load Blockly from (defaults to empty for local assets)
 * @param modifier Compose modifier
 */
@SuppressLint("SetJavaScriptEnabled")
@Composable
fun BlocklyWebView(
    onWorkspaceChanged: (String) -> Unit = {},
    initialBlocks: String? = null,
    toolboxXml: String? = null,
    blocklyUrl: String = "",
    modifier: Modifier = Modifier
) {
    val context = LocalContext.current
    val lifecycleOwner = LocalLifecycleOwner.current
    val scope = rememberCoroutineScope()
    
    var isLoading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    val webViewRef = remember { mutableStateOf<WebView?>(null) }
    
    // Create JavaScript bridge
    val jsBridge = remember {
        BlocklyJavaScriptBridge(
            context = context,
            onWorkspaceChanged = { xml ->
                isLoading = false
                errorMessage = null
                onWorkspaceChanged(xml)
            },
            onError = { error ->
                isLoading = false
                errorMessage = error
            },
            webView = webViewRef
        )
    }
    
    // Lifecycle management for WebView
    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            when (event) {
                Lifecycle.Event.ON_PAUSE -> {
                    webViewRef.value?.onPause()
                }
                Lifecycle.Event.ON_RESUME -> {
                    webViewRef.value?.onResume()
                }
                Lifecycle.Event.ON_DESTROY -> {
                    webViewRef.value?.destroy()
                    webViewRef.value = null
                }
                else -> {}
            }
        }
        
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose {
            lifecycleOwner.lifecycle.removeObserver(observer)
            webViewRef.value?.apply {
                onPause()
                clearHistory()
                clearCache(true)
                clearFormData()
                removeAllViews()
                destroy()
            }
        }
    }
    
    // Initialize WebView and load Blockly
    DisposableEffect(Unit) {
        scope.launch(Dispatchers.Main) {
            webViewRef.value?.loadUrl(blocklyUrl.ifBlank { "file:///android_asset/blockly/index.html" })
            isLoading = true
        }
        onDispose {}
    }
    
    Box(modifier = modifier) {
        AndroidView(
            factory = { ctx ->
                WebView(ctx).apply {
                    // WebView configuration
                    settings.apply {
                        javaScriptEnabled = true
                        domStorageEnabled = true
                        databaseEnabled = true
                        setSupportZoom(true)
                        builtInZoomControls = false
                        displayZoomControls = false
                        mediaPlaybackRequiresUserGesture = false
                    }
                    
                    // Add JavaScript interface for bidirectional communication
                    addJavascriptInterface(jsBridge, "AndroidBridge")
                    
                    // WebView client for lifecycle and error handling
                    webViewClient = object : WebViewClient() {
                        override fun onPageFinished(view: WebView?, url: String?) {
                            isLoading = false
                            
                            // Inject initial blocks and toolbox if provided
                            scope.launch(Dispatchers.Main) {
                                initialBlocks?.let {
                                    jsBridge.injectCustomBlocks(it)
                                }
                                toolboxXml?.let {
                                    jsBridge.setToolbox(it)
                                }
                            }
                        }
                        
                        override fun onReceivedError(
                            view: WebView?,
                            request: WebResourceRequest?,
                            error: WebResourceError?
                        ) {
                            isLoading = false
                            errorMessage = error?.description?.toString() ?: "Failed to load Blockly"
                        }
                    }
                    
                    // Chrome client for progress and title updates
                    webChromeClient = object : WebChromeClient() {
                        override fun onProgressChanged(view: WebView?, newProgress: Int) {
                            if (newProgress == 100) {
                                isLoading = false
                            }
                        }
                    }
                }.also { webViewRef.value = it }
            },
            update = { webView ->
                // Update WebView if URL changes
                if (webView.url != blocklyUrl && blocklyUrl.isNotBlank()) {
                    webView.loadUrl(blocklyUrl)
                }
            },
            modifier = Modifier.fillMaxSize()
        )
        
        // Loading indicator
        if (isLoading) {
            Surface(
                modifier = Modifier.fillMaxSize(),
                color = androidx.compose.ui.graphics.Color.White.copy(alpha = 0.9f)
            ) {
                Box(
                    contentAlignment = Alignment.Center
                ) {
                    CircularProgressIndicator(
                        color = BrandGreen
                    )
                }
            }
        }
        
        // Error message overlay
        errorMessage?.let { error ->
            Surface(
                modifier = Modifier.fillMaxSize(),
                color = androidx.compose.ui.graphics.Color.White.copy(alpha = 0.95f)
            ) {
                Box(
                    contentAlignment = Alignment.Center,
                    modifier = Modifier.fillMaxSize()
                ) {
                    androidx.compose.material3.Text(
                        text = error,
                        style = androidx.compose.material3.MaterialTheme.typography.bodyMedium,
                        color = androidx.compose.material3.MaterialTheme.colorScheme.error
                    )
                }
            }
        }
    }
}

/**
 * JavaScript bridge for bidirectional communication between Kotlin and JavaScript.
 *
 * Exposes methods to JavaScript via @JavascriptInterface annotation:
 * - onWorkspaceChanged(xml): Called when Blockly workspace changes
 * - onError(message): Called when JavaScript encounters an error
 *
 * Provides methods to call from Kotlin:
 * - saveWorkspace(filename): Save current workspace
 * - loadWorkspace(filename): Load workspace from storage
 * - injectCustomBlocks(json): Inject custom block definitions
 * - setToolbox(xml): Set custom toolbox configuration
 */
class BlocklyJavaScriptBridge(
    private val context: Context,
    private val onWorkspaceChanged: (String) -> Unit,
    private val onError: (String) -> Unit,
    val webView: MutableState<WebView?>
) {
    companion object {
        private const val TAG = "BlocklyBridge"
    }
    
    /**
     * Called from JavaScript when workspace content changes.
     *
     * @param xml XML representation of the workspace
     */
    @JavascriptInterface
    fun onWorkspaceChanged(xml: String) {
        onWorkspaceChanged(xml)
    }
    
    /**
     * Called from JavaScript when an error occurs.
     *
     * @param message Error message
     */
    @JavascriptInterface
    fun onError(message: String) {
        onError(message)
    }
    
    /**
     * Save current workspace to local storage.
     *
     * @param filename Name for the saved workspace
     */
    fun saveWorkspace(filename: String) {
        val jsCode = """
            if (typeof Blockly !== 'undefined' && Blockly.mainWorkspace) {
                try {
                    const xml = Blockly.Xml.workspaceToDom(Blockly.mainWorkspace);
                    const xmlText = Blockly.Xml.domToText(xml);
                    localStorage.setItem('blockly_workspace_$filename', xmlText);
                    AndroidBridge.onWorkspaceChanged(xmlText);
                    return true;
                } catch (e) {
                    AndroidBridge.onError('Failed to save workspace: ' + e.message);
                    return false;
                }
            }
            AndroidBridge.onError('Blockly workspace not initialized');
            return false;
        """.trimIndent()
        
        executeJavaScript(jsCode)
    }
    
    /**
     * Load workspace from local storage.
     *
     * @param filename Name of the saved workspace
     */
    fun loadWorkspace(filename: String) {
        val jsCode = """
            if (typeof Blockly !== 'undefined' && Blockly.mainWorkspace) {
                try {
                    const xmlText = localStorage.getItem('blockly_workspace_$filename');
                    if (xmlText) {
                        const parser = new DOMParser();
                        const xml = parser.parseFromString(xmlText, 'text/xml');
                        Blockly.mainWorkspace.clear();
                        Blockly.Xml.domToWorkspace(xml, Blockly.mainWorkspace);
                        AndroidBridge.onWorkspaceChanged(xmlText);
                        return true;
                    } else {
                        AndroidBridge.onError('No saved workspace found: $filename');
                        return false;
                    }
                } catch (e) {
                    AndroidBridge.onError('Failed to load workspace: ' + e.message);
                    return false;
                }
            }
            AndroidBridge.onError('Blockly workspace not initialized');
            return false;
        """.trimIndent()
        
        executeJavaScript(jsCode)
    }
    
    /**
     * Inject custom block definitions into Blockly.
     *
     * @param json JSON string containing block definitions
     */
    fun injectCustomBlocks(json: String) {
        val jsCode = """
            if (typeof Blockly !== 'undefined') {
                try {
                    const blocks = JSON.parse('$json');
                    Object.keys(blocks).forEach(blockName => {
                        if (!Blockly.Blocks[blockName]) {
                            Blockly.Blocks[blockName] = blocks[blockName];
                        }
                    });
                    // Refresh workspace to apply new blocks
                    if (Blockly.mainWorkspace) {
                        Blockly.mainWorkspace.refresh();
                    }
                } catch (e) {
                    AndroidBridge.onError('Failed to inject blocks: ' + e.message);
                }
            }
        """.trimIndent()
        
        executeJavaScript(jsCode)
    }
    
    /**
     * Set custom toolbox XML configuration.
     *
     * @param xml Toolbox XML configuration
     */
    fun setToolbox(xml: String) {
        val jsCode = """
            if (typeof Blockly !== 'undefined' && Blockly.mainWorkspace) {
                try {
                    const parser = new DOMParser();
                    const toolboxXml = parser.parseFromString('$xml', 'text/xml');
                    Blockly.mainWorkspace.updateToolbox(toolboxXml);
                } catch (e) {
                    AndroidBridge.onError('Failed to set toolbox: ' + e.message);
                }
            }
        """.trimIndent()
        
        executeJavaScript(jsCode)
    }
    
    /**
     * Get current workspace XML.
     *
     * @return XML string of the workspace, or null if not available
     */
    fun getWorkspaceXml(): String? {
        var result: String? = null
        val jsCode = """
            if (typeof Blockly !== 'undefined' && Blockly.mainWorkspace) {
                try {
                    const xml = Blockly.Xml.workspaceToDom(Blockly.mainWorkspace);
                    return Blockly.Xml.domToText(xml);
                } catch (e) {
                    AndroidBridge.onError('Failed to get workspace: ' + e.message);
                }
            }
            return null;
        """.trimIndent()
        
        return result
    }
    
    /**
     * Clear the current workspace.
     */
    fun clearWorkspace() {
        val jsCode = """
            if (typeof Blockly !== 'undefined' && Blockly.mainWorkspace) {
                try {
                    Blockly.mainWorkspace.clear();
                    AndroidBridge.onWorkspaceChanged('<xml></xml>');
                } catch (e) {
                    AndroidBridge.onError('Failed to clear workspace: ' + e.message);
                }
            }
        """.trimIndent()
        
        executeJavaScript(jsCode)
    }
    
    /**
     * Execute JavaScript code in the WebView.
     *
     * @param jsCode JavaScript code to execute
     */
    private fun executeJavaScript(jsCode: String) {
        webView.value?.evaluateJavascript("javascript:$jsCode", null)
    }
}

// BlockDefinition, BlockArgument, ToolboxCategory, ToolboxConfig are defined in shared module
// See: shared/src/androidMain/kotlin/com/armorclaw/app/studio/AgentBlocks.kt

// ============== Preview Functions ==============

/**
 * Preview of BlocklyWebView with light theme.
 */
@Composable
fun BlocklyWebViewPreviewLight() {
    ArmorClawTheme(darkTheme = false) {
        Surface {
            BlocklyWebView(
                onWorkspaceChanged = { xml ->
                    // Handle workspace changes
                },
                initialBlocks = null,
                toolboxXml = null,
                modifier = Modifier.fillMaxSize()
            )
        }
    }
}

/**
 * Preview of BlocklyWebView with dark theme.
 */
@Composable
fun BlocklyWebViewPreviewDark() {
    ArmorClawTheme(darkTheme = true) {
        Surface {
            BlocklyWebView(
                onWorkspaceChanged = { xml ->
                    // Handle workspace changes
                },
                initialBlocks = null,
                toolboxXml = null,
                modifier = Modifier.fillMaxSize()
            )
        }
    }
}

/**
 * Preview of BlocklyWebView with error state.
 */
@Composable
fun BlocklyWebViewPreviewError() {
    ArmorClawTheme(darkTheme = true) {
        Surface {
            BlocklyWebView(
                onWorkspaceChanged = { xml ->
                    // Handle workspace changes
                },
                blocklyUrl = "invalid://url",
                modifier = Modifier.fillMaxSize()
            )
        }
    }
}
