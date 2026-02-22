"""
SSL Setup Agent Instructions

These instructions are loaded into the agent to help users
set up secure external access.

SECURITY NOTE: The agent NEVER accesses user email or credentials.
For services requiring authentication, the user handles login in their
browser, then provides tokens to the agent.
"""

SSL_SETUP_INSTRUCTIONS = """
# SSL and External Access Setup

You can help users set up secure external access to this ArmorClaw instance.
When a user mentions external access, SSL, HTTPS, or wants to connect from
outside their network, offer these options.

## SECURITY RULES (CRITICAL)

1. NEVER ask for or attempt to access user email
2. NEVER ask for passwords to external services
3. NEVER attempt to automate browser login
4. For authenticated services, user provides tokens AFTER they log in
5. Always prefer "quick tunnel" options that don't require login

## Available Options

### 1. Cloudflare Quick Tunnel (RECOMMENDED - No Login)
Best option - free, trusted SSL, no account required.

To set up:
```
Say: "I'll set up a Cloudflare quick tunnel. This gives you a trusted HTTPS URL with no login required."

1. Run: cloudflared tunnel --url http://localhost:6167
2. Wait for output containing: xxx.trycloudflare.com
3. Tell user: "Your secure URL is: https://xxx.trycloudflare.com"
4. Note: "This URL changes when you restart. For a permanent URL, you'd need to create a Cloudflare account (done in your browser, not here)."
```

### 2. Cloudflare Permanent Tunnel (Requires User Login)
Only use if user specifically wants a permanent URL.

Flow:
```
Say: "For a permanent Cloudflare tunnel, you'll need to log in first:

1. Open your browser and go to: https://dash.cloudflare.com/sign-up
2. Create a free account
3. In your terminal, run: cloudflared tunnel login
4. Create a tunnel: cloudflared tunnel create armorclaw
5. Give me the tunnel name or token, and I'll configure it"

DO NOT: Ask for email, password, or attempt to automate this.
```

### 3. ngrok (No Login for Basic Use)
Quick option, URL changes on restart.

To set up:
```
Say: "I'll set up ngrok for you - it's quick but the URL changes on restart."

1. Check if ngrok installed: which ngrok
2. If not, provide installation instructions
3. Run: ngrok http 6167
4. Get URL from: curl http://localhost:4040/api/tunnels
5. Tell user the https://xxx.ngrok-free.app URL

Note: For permanent ngrok URLs, user needs to sign up at ngrok.com
and provide their auth token.
```

### 4. Self-Signed Certificate (Already Set Up)
Default for local testing. Browsers will show warnings.

If user asks about the security warning:
```
Explain: "The security warning appears because the certificate is self-signed.
Your connection IS encrypted, just not verified by a certificate authority.

For trusted SSL without warnings:
- I can set up a Cloudflare quick tunnel (free, no login)
- Or an ngrok tunnel (free, no login)

Both give you a trusted HTTPS URL. Would you like me to set one up?"
```

## Conversation Flow

When user mentions connectivity issues or SSL:

1. Check current status:
   ```python
   from openclaw.skills.ssl_tunnel_setup import get_ssl_status
   status = get_ssl_status()
   ```

2. Offer solutions (prioritize no-login options):
   - First: Cloudflare quick tunnel (no login, trusted SSL)
   - Second: ngrok (no login for basic use)
   - Third: Self-signed (already done, has warnings)

3. If user wants permanent/custom:
   - Provide step-by-step instructions for browser login
   - User returns with token
   - Agent configures with provided token

## Example Prompts to Respond To

- "How do I access this from my phone?"
- "I get a security warning in my browser"
- "Can I use HTTPS?"
- "How do I share this with someone?"
- "What's my public URL?"
- "Set up cloudflare" / "Set up ngrok"

## Response Template

When user wants external access:

"I can help you get a secure public URL. The fastest option is a Cloudflare quick tunnel:

**Option 1: Cloudflare Quick Tunnel** (Recommended)
- Free, no login or email required
- Trusted HTTPS URL like: https://your-name.trycloudflare.com
- Takes about 10 seconds to set up

**Option 2: ngrok**
- Free basic use, no login required
- URL changes on restart

**Option 3: Keep current setup**
- Self-signed SSL (already configured)
- Works but shows browser warnings

Would you like me to set up the Cloudflare quick tunnel?"
"""

# Register this as a default skill
DEFAULT_SKILLS = {
    "ssl_setup": {
        "module": "openclaw.skills.ssl_tunnel_setup",
        "instructions": SSL_SETUP_INSTRUCTIONS,
        "triggers": ["ssl", "https", "external access", "public url", "security warning", "tunnel", "ngrok", "cloudflare"],
        "priority": 10,
        "security_rules": [
            "Never access user email",
            "Never ask for passwords",
            "Never automate browser login",
            "Prefer no-login tunnel options",
            "User handles authentication externally"
        ]
    }
}

