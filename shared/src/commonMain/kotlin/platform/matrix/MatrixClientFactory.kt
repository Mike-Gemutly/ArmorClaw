package com.armorclaw.shared.platform.matrix

import com.armorclaw.shared.platform.matrix.MatrixClient
import com.armorclaw.shared.platform.matrix.MatrixClientConfig
import com.armorclaw.shared.platform.matrix.MatrixSession

/**
 * Factory for creating MatrixClient instances
 *
 * This factory provides platform-specific Matrix client creation.
 * On Android, it uses the Matrix Rust SDK via FFI.
 *
 * ## Usage
 * ```kotlin
 * val client = MatrixClientFactory.create()
 * client.login(homeserver, username, password)
 * ```
 */
expect object MatrixClientFactory {
    /**
     * Create a new Matrix client instance
     *
     * @param config Client configuration options
     * @return Platform-specific MatrixClient implementation
     */
    fun create(config: MatrixClientConfig = MatrixClientConfig()): MatrixClient

    /**
     * Create a Matrix client from a stored session
     *
     * @param session The session to restore
     * @param config Client configuration options
     * @return MatrixClient with restored session
     */
    fun createFromSession(
        session: MatrixSession,
        config: MatrixClientConfig = MatrixClientConfig()
    ): MatrixClient
}
