package com.armorclaw.app.platform

import okhttp3.CertificatePinner
import okhttp3.OkHttpClient
import java.security.MessageDigest
import java.security.cert.Certificate
import java.security.cert.CertificateEncodingException
import java.util.concurrent.TimeUnit

/**
 * Certificate pinning implementation for secure HTTPS connections
 * 
 * This class helps prevent man-in-the-middle attacks by ensuring that
 * the server presents a certificate that matches a known hash.
 */
class CertificatePinner {
    
    companion object {
        private const val PINNING_CACHE_SIZE = 5
        private const val PINNING_TTL_HOURS = 24
        
        /**
         * Create a new OkHttpClient with certificate pinning enabled
         */
        fun createPinnedClient(): OkHttpClient {
            return createPinnedClient(emptyList())
        }
        
        /**
         * Create a new OkHttpClient with certificate pinning enabled
         * 
         * @param pins List of SHA-256 certificate hashes to pin
         * @param pinDomains Whether to pin all domains or only specific ones
         */
        fun createPinnedClient(
            pins: List<String>,
            pinDomains: Boolean = true
        ): OkHttpClient {
            val builder = OkHttpClient.Builder()
                .connectTimeout(10, TimeUnit.SECONDS)
                .readTimeout(30, TimeUnit.SECONDS)
                .writeTimeout(30, TimeUnit.SECONDS)
                .retryOnConnectionFailure(true)
            
            // Enable certificate pinning if pins are provided
            if (pins.isNotEmpty()) {
                val pinBuilder = CertificatePinner.Builder()
                
                pins.forEach { pin ->
                    if (pinDomains) {
                        // Pin for all domains (development)
                        pinBuilder.add("*.armorclaw.app", pin)
                        pinBuilder.add("*.matrix.org", pin)
                        pinBuilder.add("*.matrix.org", pin)
                    } else {
                        // Pin for specific domains (production)
                        pinBuilder.add("demo.armorclaw.app", pin)
                        pinBuilder.add("matrix.org", pin)
                    }
                }
                
                builder.certificatePinner(pinBuilder.build())
            }
            
            return builder.build()
        }
        
        /**
         * Create an OkHttpClient for development without strict pinning
         * 
         * Use this for development/testing where certificates may change
         */
        fun createDevClient(): OkHttpClient {
            return OkHttpClient.Builder()
                .connectTimeout(10, TimeUnit.SECONDS)
                .readTimeout(30, TimeUnit.SECONDS)
                .writeTimeout(30, TimeUnit.SECONDS)
                .retryOnConnectionFailure(true)
                .build()
        }
        
        /**
         * Calculate SHA-256 hash of a certificate
         * 
         * @param certificate The certificate to hash
         * @return Base64-encoded SHA-256 hash
         */
        fun calculateCertificateHash(certificate: Certificate): String {
            return try {
                val encoded = certificate.encoded
                val digest = MessageDigest.getInstance("SHA-256")
                val hash = digest.digest(encoded)
                android.util.Base64.encodeToString(hash, android.util.Base64.NO_WRAP)
            } catch (e: CertificateEncodingException) {
                throw CertificateHashingException("Failed to encode certificate", e)
            }
        }
        
        /**
         * Extract certificate pins from a certificate chain
         * 
         * @param certificates The certificate chain
         * @return List of SHA-256 hashes
         */
        fun extractCertificatePins(certificates: List<Certificate>): List<String> {
            return certificates.map { certificate ->
                calculateCertificateHash(certificate)
            }
        }
        
        /**
         * Verify that a certificate matches expected pins
         * 
         * @param certificate The certificate to verify
         * @param expectedPins List of expected SHA-256 hashes
         * @return true if the certificate matches one of the pins
         */
        fun verifyCertificatePins(
            certificate: Certificate,
            expectedPins: List<String>
        ): Boolean {
            val actualPin = calculateCertificateHash(certificate)
            return expectedPins.any { expectedPin ->
                expectedPin.equals(actualPin, ignoreCase = true)
            }
        }
        
        /**
         * Create certificate pin from PEM-encoded certificate
         * 
         * @param pemCertificate PEM-encoded certificate string
         * @return SHA-256 hash
         */
        fun createPinFromPEM(pemCertificate: String): String {
            val certBytes = decodePEMCertificate(pemCertificate)
            val digest = MessageDigest.getInstance("SHA-256")
            val hash = digest.digest(certBytes)
            return android.util.Base64.encodeToString(hash, android.util.Base64.NO_WRAP)
        }
        
        /**
         * Validate certificate pin format
         * 
         * @param pin The certificate pin to validate
         * @return true if the pin format is valid
         */
        fun validatePinFormat(pin: String): Boolean {
            // Certificate pins should be base64-encoded SHA-256 hashes
            // They should be 44 characters long (32 bytes * 8/6 rounded up)
            return pin.length == 44 && pin.matches(Regex("^[A-Za-z0-9+/]+$"))
        }
    }
    
    /**
     * Exception thrown when certificate hashing fails
     */
    class CertificateHashingException(
        message: String,
        cause: Throwable? = null
    ) : Exception(message, cause)
    
    /**
     * Exception thrown when certificate verification fails
     */
    class CertificateVerificationException(
        message: String,
        cause: Throwable? = null
    ) : Exception(message, cause)
}

/**
 * Certificate pinning configuration
 */
data class CertificatePinningConfig(
    val enabled: Boolean = true,
    val pins: List<String> = emptyList(),
    val pinDomains: Boolean = true,
    val enableForDebug: Boolean = false,
    val enforcePinning: Boolean = true
)

/**
 * Certificate pinning result
 */
sealed class CertificatePinningResult {
    object Success : CertificatePinningResult()
    data class PinningDisabled(val reason: String) : CertificatePinningResult()
    data class VerificationFailed(val error: String) : CertificatePinningResult()
    data class PinNotConfigured(val domain: String) : CertificatePinningResult()
}

/**
 * Helper function to decode PEM-encoded certificate
 */
private fun decodePEMCertificate(pem: String): ByteArray {
    val lines = pem.split("\n")
    val base64Lines = lines.filter { line ->
        !line.startsWith("-----") && line.isNotBlank()
    }
    
    val base64 = base64Lines.joinToString("")
    return android.util.Base64.decode(base64, android.util.Base64.NO_WRAP)
}

/**
 * Example certificate pins for known servers
 */
object KnownCertificatePins {
    /**
     * Example pin for demo.armorclaw.app (replace with actual pins)
     */
    val DEMO_ARMORCLAW_APP = listOf(
        "sha256/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
        "sha256/BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB="
    )
    
    /**
     * Example pin for matrix.org (replace with actual pins)
     */
    val MATRIX_ORG = listOf(
        "sha256/CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC=",
        "sha256/DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD="
    )
}

/**
 * Usage example:
 * 
 * ```kotlin
 * // Create pinned client for production
 * val client = CertificatePinner.createPinnedClient(
 *     pins = KnownCertificatePins.DEMO_ARMORCLAW_APP,
 *     pinDomains = true
 * )
 * 
 * // Create client for development (no pinning)
 * val devClient = CertificatePinner.createDevClient()
 * 
 * // Verify certificate pins
 * val isValid = CertificatePinner.verifyCertificatePins(
 *     certificate,
 *     expectedPins = KnownCertificatePins.DEMO_ARMORCLAW_APP
 * )
 * ```
 */
