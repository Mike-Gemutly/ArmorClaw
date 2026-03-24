package app.armorclaw

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.navigation.compose.rememberNavController
import app.armorclaw.navigation.ArmorClawNavHost

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            ArmorClawTheme {
                val navController = rememberNavController()
                ArmorClawNavHost(navController = navController)
            }
        }
    }
}

@Composable
fun ArmorClawTheme(
    content: @Composable () -> Unit
) {
    val darkColorScheme = darkColorScheme()

    MaterialTheme(
        colorScheme = darkColorScheme,
        content = content
    )
}
