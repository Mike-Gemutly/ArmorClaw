package com.armorclaw.shared.secretary

enum class SecretaryMode {
    MEETING,
    FOCUS,
    SLEEP,
    NORMAL
}

data class PolicyContext(
    val mode: SecretaryMode,
    val whitelist: List<String> = emptyList()
)

data class PolicyDecision(
    val shouldSuppress: Boolean,
    val suppressionReason: String? = null
)

class SecretaryPolicyEngine {

    fun evaluateCard(card: ProactiveCard, context: PolicyContext): PolicyDecision {
        if (card.priority == SecretaryPriority.CRITICAL) {
            return PolicyDecision(
                shouldSuppress = false,
                suppressionReason = null
            )
        }

        return when (context.mode) {
            SecretaryMode.NORMAL -> {
                PolicyDecision(
                    shouldSuppress = false,
                    suppressionReason = null
                )
            }
            SecretaryMode.MEETING -> {
                PolicyDecision(
                    shouldSuppress = true,
                    suppressionReason = "Meeting in progress"
                )
            }
            SecretaryMode.FOCUS -> {
                if (card.id in context.whitelist) {
                    PolicyDecision(
                        shouldSuppress = false,
                        suppressionReason = null
                    )
                } else {
                    PolicyDecision(
                        shouldSuppress = true,
                        suppressionReason = "Focus mode active - not whitelisted"
                    )
                }
            }
            SecretaryMode.SLEEP -> {
                PolicyDecision(
                    shouldSuppress = true,
                    suppressionReason = "Batching for later review"
                )
            }
        }
    }
}
