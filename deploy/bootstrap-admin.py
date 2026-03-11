#!/usr/bin/env python3
"""
ArmorClaw Production Bootstrap – Secure Admin Creation
Temporarily enables open registration, creates admin user, then disables.
"""
import os
import sys
import subprocess
import json
import time
import secrets
import logging
from pathlib import Path
import toml
import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

# ------------------------------------------------------------------------------
# Configuration & Constants
# ------------------------------------------------------------------------------
CONDUIT_CONFIG_PATH = Path("/etc/conduit.toml")
INIT_FLAG_PATH = Path("/var/lib/armorclaw/.bootstrapped")
LOG_FILE = Path("/var/log/armorclaw/bootstrap.log")
CONDUIT_URL = "http://localhost:6167"

# Ensure log directory exists
LOG_FILE.parent.mkdir(parents=True, exist_ok=True)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    handlers=[
        logging.StreamHandler(sys.stdout),
        logging.FileHandler(LOG_FILE)
    ]
)
logger = logging.getLogger("bootstrap")

# Admin username configuration with conflict mitigation
DEFAULT_ADMIN_USER = "admin"
# Priority: Environment var > Random suffix > Default
RAW_ADMIN_USERNAME = os.getenv("ARMORCLAW_ADMIN_USERNAME")

# Validate and sanitize username (simple validation)
def validate_env_username(username):
    """Validate username is safe for Matrix"""
    if not username:
        return False
    # Only allow alphanumeric, dash, underscore
    return all(c.isalnum() or c in '-_' for c in username)

# Set admin username
if RAW_ADMIN_USERNAME and validate_env_username(RAW_ADMIN_USERNAME):
    ADMIN_USERNAME = RAW_ADMIN_USERNAME
    logger.info(f"Using validated admin username: {ADMIN_USERNAME}")
else:
    # Generate randomized default if not specified
    random_suffix = secrets.token_hex(4)
    ADMIN_USERNAME = f"armor-admin-{random_suffix}"
    logger.info(f"Using randomized admin username: {ADMIN_USERNAME}")

def wait_for_conduit(timeout=120, interval=2, max_attempts=60):
    """Wait for Conduit to be ready with exponential backoff"""
    session = requests.Session()
    retries = Retry(total=5, backoff_factor=1, status_forcelist=[502, 503, 504])
    session.mount("http://", HTTPAdapter(max_retries=retries))
    
    logger.info(f"Waiting for Conduit to become ready (max {max_attempts}s)...")
    start = time.time()
    attempt = 0
    backoff = 1
    
    while time.time() - start < timeout and attempt < max_attempts:
        try:
            r = session.get(f"{CONDUIT_URL}/_matrix/client/versions", timeout=5)
            if r.status_code == 200:
                logger.info(f"Conduit is ready after {attempt + 1} attempts")
                return True
            elif r.status_code == 503:
                # Service unavailable - exponential backoff
                logger.warning(f"Conduit temporarily unavailable (attempt {attempt + 1}/{max_attempts})")
                time.sleep(min(backoff, 10))
                backoff *= 2
            else:
                logger.debug(f"Conduit not ready yet (HTTP {r.status_code}, attempt {attempt + 1})")
        except Exception as e:
            logger.debug(f"Conduit not ready yet: {e}")
        
        attempt += 1
        time.sleep(min(interval, backoff))
    
    logger.error(f"Conduit failed to become ready after {timeout}s")
    return False

def generate_secure_secret(length=64):
    """Generate cryptographically secure random hex string"""
    return secrets.token_hex(length // 2)

def enable_open_registration():
    """Enable open registration in Conduit config for admin creation"""
    try:
        config = toml.load(CONDUIT_CONFIG_PATH)
        config.setdefault("global", {})
        # Enable open registration - no token required
        config["global"]["allow_registration"] = True

        with open(CONDUIT_CONFIG_PATH, "w") as f:
            toml.dump(config, f)

        logger.info("Conduit config updated: open registration enabled")
        return True
    except Exception as e:
        logger.error(f"Failed to update Conduit config: {e}")
        return False

def remove_registration_token():
    """Permanently remove registration token from config and disable registration"""
    try:
        config = toml.load(CONDUIT_CONFIG_PATH)
        if "global" in config:
            if "registration_token" in config["global"]:
                del config["global"]["registration_token"]
            # Disable registration after admin creation for security
            config["global"]["allow_registration"] = False
            with open(CONDUIT_CONFIG_PATH, "w") as f:
                toml.dump(config, f)
            logger.info("Registration token removed from config (security cleanup)")
    except Exception as e:
        logger.warning(f"Failed to remove registration token: {e}")

def register_admin(password: str):
    """Register admin user using Conduit's two-step open registration API"""
    global ADMIN_USERNAME
    
    def _do_registration(username: str, session: str = None):
        """Execute registration request, handling two-step flow"""
        if session:
            # Second step: complete registration with session and auth
            payload = {
                "username": username,
                "password": password,
                "session": session,
                "auth": {"type": "m.login.dummy"}
            }
        else:
            # First step: initial registration request
            payload = {
                "username": username,
                "password": password
            }
        
        r = requests.post(
            f"{CONDUIT_URL}/_matrix/client/v3/register",
            json=payload,
            timeout=10
        )
        
        logger.info(f"Registration response: HTTP {r.status_code}, body: {r.text[:500]}")
        return r
    
    def _handle_registration_response(r, username: str):
        """Process registration response, handling 401 challenge"""
        if r.status_code == 200:
            response_data = r.json()
            user_id = response_data.get("user_id", f"@{username}:{os.getenv('ARMORCLAW_SERVER_NAME', 'localhost')}")
            logger.info(f"Admin user registered successfully: {user_id}")
            
            # Store password securely (0600) - TEMPORARILY for display
            pw_file = Path("/var/lib/armorclaw/.admin_password_display")
            pw_file.write_text(password)
            pw_file.chmod(0o600)
            logger.info(f"Admin password prepared for display: {pw_file}")
            
            # Store actual username used
            with open("/var/lib/armorclaw/.admin_username", "w") as f:
                f.write(username)
            
            return True, None
        
        elif r.status_code == 401:
            # Two-step registration: extract session and complete flow
            try:
                response_data = r.json()
                session = response_data.get("session")
                flows = response_data.get("flows", [])
                
                if session and flows:
                    logger.info(f"Two-step registration: got session={session}, flows={flows}")
                    # Complete registration with second request
                    r2 = _do_registration(username, session)
                    return _handle_registration_response(r2, username)
                else:
                    logger.error(f"401 response missing session/flows: {response_data}")
                    return False, response_data
            except json.JSONDecodeError as e:
                logger.error(f"Failed to parse 401 response: {e}")
                return False, None
        
        elif r.status_code == 400 or r.status_code == 409:
            # Username conflict handling
            response_data = None
            try:
                response_data = r.json()
                error_code = response_data.get("errcode", "")
                error_msg = response_data.get("error", "")
            except:
                error_code = ""
                error_msg = ""
            
            if "user_in_use" in error_code.lower() or "already in use" in error_msg.lower() or r.status_code == 409:
                return False, "username_conflict"
            else:
                logger.error(f"Registration failed: HTTP {r.status_code}, {response_data}")
                return False, response_data
        else:
            logger.error(f"Unexpected status code: {r.status_code}")
            return False, None
    
    try:
        # First registration attempt
        r = _do_registration(ADMIN_USERNAME)
        success, error = _handle_registration_response(r, ADMIN_USERNAME)
        
        if success:
            return True
        
        # Handle username conflict - try alternative
        if error == "username_conflict":
            logger.warning(f"Username '{ADMIN_USERNAME}' already exists, trying alternative")
            
            alt_suffix = secrets.token_hex(4)
            alt_username = f"armor-admin-{alt_suffix}"
            
            logger.info(f"Trying alternative username: {alt_username}")
            
            r2 = _do_registration(alt_username)
            success2, error2 = _handle_registration_response(r2, alt_username)
            
            if success2:
                ADMIN_USERNAME = alt_username
                return True
            elif error2 == "username_conflict":
                logger.error(f"Alternative username also taken, giving up")
                return False
            else:
                logger.error(f"Failed to register alternative admin user: {error2}")
                return False
        
        logger.error(f"Registration failed: {error}")
        return False
        
    except Exception as e:
        logger.error(f"Admin registration failed: {e}")
        return False

def main():
    """Main bootstrap logic with comprehensive error handling"""
    if INIT_FLAG_PATH.exists():
        logger.info("Bootstrap already completed → skipping")
        return 0

    logger.info("Starting secure first-run bootstrap...")

    # Generate or use provided password (never log the actual password)
    password = os.getenv("ARMORCLAW_ADMIN_PASSWORD") or secrets.token_urlsafe(16)
    password_len = len(password)
    logger.info(f"Admin password generated (length: {password_len})")

    if not wait_for_conduit():
        return 1

    # Enable open registration (no token required)
    if not enable_open_registration():
        return 1

    # Restart Conduit container to pick up new config (read-only mount requires restart)
    try:
        subprocess.run(["docker", "restart", "armorclaw-conduit"], check=True)
        logger.info("Restarted Conduit container to pick up new config")
        time.sleep(5)  # Wait for Conduit to restart
    except Exception as e:
        logger.warning(f"Could not restart Conduit container: {e}")

    if not wait_for_conduit():  # Re-check after reload
        return 1

    # Register admin user without a token (open registration)
    if not register_admin(password):
        return 1

    remove_registration_token()
    INIT_FLAG_PATH.touch()
    logger.info("Bootstrap complete. Admin user ready.")

    # Display final success banner (password shown only once)
    server_name = os.getenv('ARMORCLAW_SERVER_NAME', 'localhost')
    print("\n" + "=" * 60)
    print("ArmorClaw Production Bootstrap – SUCCESS")
    print("=" * 60)
    print(f"Admin Username: @{ADMIN_USERNAME}:{server_name}")
    print(f"Admin Password: {password}")
    print("")
    print("⚠️  SAVE CREDENTIALS NOW - They will not be shown again!")
    print("")
    print("Next: Connect via Element X or ArmorChat using http://<your-ip>:6167")
    print("=" * 60 + "\n")

    return 0

if __name__ == "__main__":
    sys.exit(main())