"""
SSL Setup Agent Instructions

These instructions are loaded into the agent to help users
set up secure external access.
"""

SSL_SETUP_INSTRUCTIONS = """
# SSL and External Access Setup

You can help users set up secure external access to this ArmorClaw instance.
When a user mentions external access, SSL, HTTPS, or wants to connect from
outside their network, offer these options:

## Available Options

### 1. Cloudflare Tunnel (RECOMMENDED - Free & Permanent)
Best for permanent access with trusted SSL.

To set up:
```
Ask user: "Would you like me to set up a Cloudflare tunnel? This gives you a permanent URL with trusted SSL, no domain required."

If yes:
1. Run: cloudflared tunnel --url http://localhost:6167
2. Wait for URL like: https://random-name.trycloudflare.com
3. Tell user: "Your secure URL is: https://xxx.trycloudflare.com"
4. Instruct: "Use this URL in ArmorChat as your homeserver"
```

### 2. ngrok (Quick & Easy - Free but Temporary)
Best for quick testing, URL changes on restart.

To set up:
```
Ask user: "Would you like me to set up ngrok? This is quick but the URL changes each time you restart."

If yes:
1. Check if ngrok installed: which ngrok
2. If not, install it
3. Run: ngrok http 6167
4. Get URL from: curl http://localhost:4040/api/tunnels
5. Tell user the https://xxx.ngrok.io URL
```

### 3. Self-Signed Certificate (Already Set Up)
Default for local testing. Browsers will show warnings.

If user asks about the security warning:
```
Explain: "The security warning appears because the certificate is self-signed.
For trusted SSL without warnings, I can set up Cloudflare or ngrok - both are free."
```

## Conversation Flow

When user mentions connectivity issues or SSL:

1. Check current status:
   ```python
   from openclaw.skills.ssl_tunnel_setup import get_ssl_status
   status = get_ssl_status()
   ```

2. Offer solutions based on needs:
   - Permanent + trusted → Cloudflare
   - Quick test → ngrok
   - Local only → Self-signed (already done)

3. Guide through setup step by step

4. Update configuration if needed

## Example Prompts to Respond To

- "How do I access this from my phone?"
- "I get a security warning in my browser"
- "Can I use HTTPS?"
- "How do I share this with someone?"
- "What's my public URL?"

## Response Template

When user wants external access:

"I can help you set up secure external access. Here are your options:

1. **Cloudflare Tunnel** (Recommended)
   - Free permanent URL like: https://your-name.trycloudflare.com
   - Trusted SSL (no browser warnings)
   - No domain or credit card required

2. **ngrok** (Quick Test)
   - Instant setup, URL like: https://abc123.ngrok.io
   - Free but URL changes on restart

3. **Self-Signed** (Current)
   - Already configured for local testing
   - Browsers show security warnings

Which would you like to set up?"
"""

# Register this as a default skill
DEFAULT_SKILLS = {
    "ssl_setup": {
        "module": "openclaw.skills.ssl_tunnel_setup",
        "instructions": SSL_SETUP_INSTRUCTIONS,
        "triggers": ["ssl", "https", "external access", "public url", "security warning", "tunnel", "ngrok", "cloudflare"],
        "priority": 10
    }
}
