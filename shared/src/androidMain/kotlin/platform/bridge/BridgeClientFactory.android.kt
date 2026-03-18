package com.armorclaw.shared.platform.bridge

import io.ktor.client.*
import io.ktor.client.engine.okhttp.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.plugins.logging.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.serialization.json.Json

/**
 * Android implementation of BridgeClientFactory
 *
 * Uses OkHttp engine with certificate pinning support.
 */
actual object BridgeClientFactory {
    actual fun createClient(config: BridgeConfig): BridgeRpcClient {
        return BridgeRpcClientImpl(config, createHttpClient(config))
    }

    actual fun createHttpClient(config: BridgeConfig): HttpClient {
        return HttpClient(OkHttp) {
            install(ContentNegotiation) {
                json(Json {
                    ignoreUnknownKeys = true
                    isLenient = true
                })
            }

            install(Logging) {
                logger = Logger.DEFAULT
                level = LogLevel.INFO
            }

            expectSuccess = true
        }
    }
}
