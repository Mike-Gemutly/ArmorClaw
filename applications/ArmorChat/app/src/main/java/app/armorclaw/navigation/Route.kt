package app.armorclaw.navigation

sealed class Route(val route: String) {
    object Bonding : Route("bonding")
    object SecurityConfig : Route("security_config")
    object KeyBackup : Route("key_backup")
    object KeyRecovery : Route("key_recovery")
    object HardeningPassword : Route("hardening_password")
    object HardeningDevice : Route("hardening_device")
    object HardeningBiometrics : Route("hardening_biometrics")
    object Home : Route("home")
    object Workflow : Route("workflow")
    object Approvals : Route("approvals")
    data class Room(val roomId: String) : Route("room/$roomId")
    data class EmailApproval(val approvalId: String) : Route("email/approve/$approvalId")
}
