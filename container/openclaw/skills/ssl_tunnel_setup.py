"""
SSL Tunnel Setup Skills for ArmorClaw

Provides skills for setting up secure external access:
- ngrok quick tunnel (free, temporary)
- Cloudflare tunnel (free, permanent)
- Self-signed certificate generation

These skills are invoked by the agent when user wants external access.
"""

import json
import subprocess
import os
from typing import Optional, Dict, Any


class SSLTunnelSkill:
    """Base class for SSL tunnel skills"""

    name = "ssl_tunnel"
    description = "Set up secure external access to ArmorClaw"

    def check_available(self) -> bool:
        """Check if this tunnel method is available"""
        raise NotImplementedError

    def setup(self, **kwargs) -> Dict[str, Any]:
        """Set up the tunnel"""
        raise NotImplementedError

    def get_url(self) -> Optional[str]:
        """Get the public URL"""
        raise NotImplementedError

    def teardown(self) -> bool:
        """Remove the tunnel"""
        raise NotImplementedError


class NgrokTunnelSkill(SSLTunnelSkill):
    """ngrok quick tunnel - free, temporary SSL"""

    name = "ngrok_tunnel"
    description = "Set up ngrok tunnel for temporary SSL access"

    def check_available(self) -> bool:
        """Check if ngrok is installed"""
        result = subprocess.run(
            ["which", "ngrok"],
            capture_output=True,
            text=True
        )
        return result.returncode == 0

    def install(self) -> Dict[str, Any]:
        """Install ngrok"""
        commands = [
            "curl -s https://ngrok-agent.s3.amazonaws.com/ngrok.asc | sudo tee /etc/apt/trusted.gpg.d/ngrok.asc >/dev/null",
            'echo "deb https://ngrok-agent.s3.amazonaws.com buster main" | sudo tee /etc/apt/sources.list.d/ngrok.list',
            "sudo apt update && sudo apt install -y ngrok"
        ]

        for cmd in commands:
            result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
            if result.returncode != 0:
                return {"success": False, "error": result.stderr}

        return {"success": True, "message": "ngrok installed successfully"}

    def setup(self, port: int = 6167, auth_token: Optional[str] = None) -> Dict[str, Any]:
        """Start ngrok tunnel"""
        # Install if not available
        if not self.check_available():
            install_result = self.install()
            if not install_result.get("success"):
                return install_result

        # Set auth token if provided
        if auth_token:
            subprocess.run(
                ["ngrok", "config", "add-authtoken", auth_token],
                capture_output=True
            )

        # Start tunnel
        try:
            # Kill any existing ngrok
            subprocess.run(["pkill", "ngrok"], capture_output=True)

            # Start new tunnel in background
            subprocess.Popen(
                ["ngrok", "http", str(port), "--log=stdout"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL
            )

            # Wait for tunnel to start
            import time
            time.sleep(3)

            # Get public URL from ngrok API
            result = subprocess.run(
                ["curl", "-s", "http://localhost:4040/api/tunnels"],
                capture_output=True,
                text=True
            )

            if result.returncode == 0:
                data = json.loads(result.stdout)
                if data.get("tunnels"):
                    public_url = data["tunnels"][0]["public_url"]
                    return {
                        "success": True,
                        "url": public_url,
                        "message": f"ngrok tunnel active: {public_url}"
                    }

            return {"success": False, "error": "Failed to get tunnel URL"}

        except Exception as e:
            return {"success": False, "error": str(e)}

    def get_url(self) -> Optional[str]:
        """Get current ngrok URL"""
        try:
            result = subprocess.run(
                ["curl", "-s", "http://localhost:4040/api/tunnels"],
                capture_output=True,
                text=True
            )
            if result.returncode == 0:
                data = json.loads(result.stdout)
                if data.get("tunnels"):
                    return data["tunnels"][0]["public_url"]
        except:
            pass
        return None

    def teardown(self) -> bool:
        """Stop ngrok tunnel"""
        result = subprocess.run(["pkill", "ngrok"], capture_output=True)
        return result.returncode == 0


class CloudflareTunnelSkill(SSLTunnelSkill):
    """Cloudflare tunnel - free, permanent SSL"""

    name = "cloudflare_tunnel"
    description = "Set up Cloudflare tunnel for permanent SSL access"

    def check_available(self) -> bool:
        """Check if cloudflared is installed"""
        result = subprocess.run(
            ["which", "cloudflared"],
            capture_output=True,
            text=True
        )
        return result.returncode == 0

    def install(self) -> Dict[str, Any]:
        """Install cloudflared"""
        commands = [
            "curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 -o /tmp/cloudflared",
            "chmod +x /tmp/cloudflared",
            "sudo mv /tmp/cloudflared /usr/local/bin/cloudflared"
        ]

        for cmd in commands:
            result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
            if result.returncode != 0:
                return {"success": False, "error": result.stderr}

        return {"success": True, "message": "cloudflared installed successfully"}

    def setup(self, port: int = 6167, quick: bool = True) -> Dict[str, Any]:
        """
        Start Cloudflare tunnel

        Args:
            port: Local port to tunnel
            quick: Use quick tunnel (no account needed, temporary URL)
        """
        # Install if not available
        if not self.check_available():
            install_result = self.install()
            if not install_result.get("success"):
                return install_result

        try:
            # Kill any existing cloudflared
            subprocess.run(["pkill", "cloudflared"], capture_output=True)

            if quick:
                # Quick tunnel - no account needed
                process = subprocess.Popen(
                    ["cloudflared", "tunnel", "--url", f"http://localhost:{port}"],
                    stdout=subprocess.PIPE,
                    stderr=subprocess.STDOUT,
                    text=True
                )

                # Wait for URL in output
                import time
                for _ in range(30):  # 30 second timeout
                    line = process.stdout.readline()
                    if "trycloudflare.com" in line:
                        # Extract URL
                        import re
                        match = re.search(r'https://[^\s]+\.trycloudflare\.com', line)
                        if match:
                            return {
                                "success": True,
                                "url": match.group(0),
                                "message": f"Cloudflare tunnel active: {match.group(0)}",
                                "note": "This URL is temporary. For permanent URL, use: cloudflared tunnel login"
                            }
                    time.sleep(1)

                return {"success": False, "error": "Timeout waiting for tunnel URL"}

            else:
                # Named tunnel (requires account)
                return {
                    "success": False,
                    "error": "Named tunnels require Cloudflare account. Run: cloudflared tunnel login"
                }

        except Exception as e:
            return {"success": False, "error": str(e)}

    def get_url(self) -> Optional[str]:
        """Get current tunnel URL (for quick tunnels, need to check logs)"""
        # Quick tunnels don't have an API, URL is in stdout
        return None

    def teardown(self) -> bool:
        """Stop cloudflared tunnel"""
        result = subprocess.run(["pkill", "cloudflared"], capture_output=True)
        return result.returncode == 0


class SelfSignedCertSkill(SSLTunnelSkill):
    """Generate self-signed certificate for local testing"""

    name = "self_signed_cert"
    description = "Generate self-signed SSL certificate"

    def check_available(self) -> bool:
        """Check if openssl is installed"""
        result = subprocess.run(
            ["which", "openssl"],
            capture_output=True,
            text=True
        )
        return result.returncode == 0

    def setup(
        self,
        ip_address: str,
        output_dir: str = "/etc/armorclaw/ssl",
        days: int = 365
    ) -> Dict[str, Any]:
        """
        Generate self-signed certificate

        Args:
            ip_address: Server IP address
            output_dir: Directory to save certificates
            days: Certificate validity in days
        """
        os.makedirs(output_dir, exist_ok=True)

        key_path = os.path.join(output_dir, "key.pem")
        cert_path = os.path.join(output_dir, "cert.pem")

        cmd = [
            "openssl", "req", "-x509", "-nodes",
            f"-days={days}",
            "-newkey", "rsa:2048",
            f"-keyout={key_path}",
            f"-out={cert_path}",
            f"-subj=/CN={ip_address}",
            f"-addext=subjectAltName=IP:{ip_address}"
        ]

        result = subprocess.run(cmd, capture_output=True, text=True)

        if result.returncode == 0:
            return {
                "success": True,
                "key_path": key_path,
                "cert_path": cert_path,
                "message": f"Self-signed certificate generated for {ip_address}",
                "warning": "Browsers will show security warnings. Consider using ngrok or Cloudflare for trusted SSL."
            }

        return {"success": False, "error": result.stderr}

    def get_url(self) -> Optional[str]:
        return None

    def teardown(self) -> bool:
        return True


# Skill registry
SSL_SKILLS = {
    "ngrok": NgrokTunnelSkill(),
    "cloudflare": CloudflareTunnelSkill(),
    "self_signed": SelfSignedCertSkill()
}


def list_ssl_skills() -> Dict[str, str]:
    """List available SSL skills with descriptions"""
    return {name: skill.description for name, skill in SSL_SKILLS.items()}


def setup_ssl_tunnel(method: str, **kwargs) -> Dict[str, Any]:
    """
    Set up SSL tunnel using specified method

    Args:
        method: One of 'ngrok', 'cloudflare', 'self_signed'
        **kwargs: Method-specific options

    Returns:
        Dict with success status and URL (if applicable)
    """
    skill = SSL_SKILLS.get(method)
    if not skill:
        return {"success": False, "error": f"Unknown method: {method}"}

    return skill.setup(**kwargs)


def get_ssl_status() -> Dict[str, Any]:
    """Get current SSL/tunnel status"""
    status = {}

    # Check ngrok
    ngrok = SSL_SKILLS["ngrok"]
    if ngrok.check_available():
        url = ngrok.get_url()
        status["ngrok"] = {
            "installed": True,
            "active": url is not None,
            "url": url
        }
    else:
        status["ngrok"] = {"installed": False, "active": False}

    # Check cloudflare
    cf = SSL_SKILLS["cloudflare"]
    status["cloudflare"] = {
        "installed": cf.check_available(),
        "active": False  # Would need to check process
    }

    # Check self-signed cert
    cert_path = "/etc/armorclaw/ssl/cert.pem"
    status["self_signed"] = {
        "exists": os.path.exists(cert_path),
        "path": cert_path if os.path.exists(cert_path) else None
    }

    return status
