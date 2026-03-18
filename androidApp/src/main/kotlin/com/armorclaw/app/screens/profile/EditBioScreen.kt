package com.armorclaw.app.screens.profile
import androidx.compose.foundation.layout.Arrangement

import androidx.compose.material3.MaterialTheme

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.AccentColor
import com.armorclaw.shared.ui.theme.SurfaceColor

/**
 * Edit bio screen
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun EditBioScreen(
    currentBio: String,
    onNavigateBack: () -> Unit,
    onSaveBio: (bio: String) -> Unit,
    modifier: Modifier = Modifier
) {
    val scrollState = rememberScrollState()
    var bio by remember { mutableStateOf(currentBio) }
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Edit Bio") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.Default.ArrowBack, "Back")
                    }
                },
                actions = {
                    TextButton(
                        onClick = { onSaveBio(bio) },
                        colors = ButtonDefaults.textButtonColors(
                            contentColor = AccentColor
                        )
                    ) {
                        Text("Save", fontWeight = FontWeight.SemiBold)
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = SurfaceColor
                )
            )
        },
        modifier = modifier
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .verticalScroll(scrollState)
                .imePadding(),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            OutlinedTextField(
                value = bio,
                onValueChange = { bio = it },
                label = { Text("Bio") },
                placeholder = { Text("Tell us a bit about yourself...") },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(200.dp)
                    .padding(horizontal = 16.dp),
                maxLines = 10,
                shape = RoundedCornerShape(12.dp)
            )
            
            Text(
                text = "${bio.length}/150 characters",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                modifier = Modifier.padding(horizontal = 16.dp)
            )
        }
    }
}

@Preview(showBackground = true)
@Composable
private fun EditBioScreenPreview() {
    ArmorClawTheme {
        EditBioScreen(
            currentBio = "",
            onNavigateBack = {},
            onSaveBio = {}
        )
    }
}
