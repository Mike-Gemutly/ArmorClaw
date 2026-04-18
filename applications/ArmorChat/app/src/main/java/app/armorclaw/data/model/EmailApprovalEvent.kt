package app.armorclaw.data.model

object EmailApprovalEventType {
    const val EMAIL_APPROVAL_REQUEST = "app.armorclaw.email_approval_request"
}

data class EmailApprovalEvent(
    val approvalId: String,
    val emailId: String,
    val to: String,
    val piiFields: Int,
    val timeoutS: Int = 300
) {
    companion object {
        @Suppress("UNCHECKED_CAST")
        fun fromContentMap(content: Map<String, Any>): EmailApprovalEvent? {
            val approvalId = content["approval_id"] as? String ?: return null
            val emailId = content["email_id"] as? String ?: return null
            val to = content["to"] as? String ?: return null
            val piiFields = (content["pii_fields"] as? Number)?.toInt() ?: 0
            val timeoutS = (content["timeout_s"] as? Number)?.toInt() ?: 300
            return EmailApprovalEvent(
                approvalId = approvalId,
                emailId = emailId,
                to = to,
                piiFields = piiFields,
                timeoutS = timeoutS
            )
        }
    }

    fun toSystemAlertContent(): SystemAlertContent {
        return SystemAlertContent(
            alertType = AlertType.EMAIL_APPROVAL_REQUEST,
            severity = AlertSeverity.WARNING,
            title = "Email Approval Request",
            message = "Agent wants to send email to $to with $piiFields PII field(s)",
            metadata = mapOf(
                "approval_id" to approvalId,
                "email_id" to emailId,
                "to" to to,
                "pii_fields" to piiFields,
                "timeout_s" to timeoutS
            )
        )
    }
}
