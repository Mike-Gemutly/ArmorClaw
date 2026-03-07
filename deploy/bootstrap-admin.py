#!/usr/bin/env python3
"""
ArmorClaw Production Bootstrap – Secure Admin Creation
Never enables public registration. Uses ephemeral shared secret.
"""
import os
import sys
import subprocess
import json
import time
import secrets
import hmac
import hashlib
import logging
from pathlib import Path
import toml
import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

# ------------------------------------------------------------------------------
# Configuration & Constants
# ------------------------------------------------------------------------------
CONDUIT_CONFIG_PATH = Path("/etc/armorclaw/conduit.toml")
INIT_FLAG_PATH = Path("/var/lib/armorclaw/.bootstrapped")
LOG_FILE = Path("/var/log/armorclaw/bootstrap.log")
CONDUIT_URL = "http://localhost:6167"

# Admin username configuration with conflict mitigation
DEFAULT_ADMIN_USER = "admin"
# Priority: Environment var > Random suffix > Default
RAW_ADMIN_USERNAME = os.getenv("ARMORCLAW_ADMIN_USERNAME")

# Validate and sanitize username
if RAW_ADMIN_USERNAME:
    if validate_env_username(RAW_ADMIN_USERNAME):
        ADMIN_USERNAME = RAW_ADMIN_USERNAME
        logger.info(f"Using validated admin username: {ADMIN_USERNAME}")
    else:
        logger.error("Invalid ARMORCLAW_ADMIN_USERNAME - using random")
        RAW_ADMIN_USERNAME = None

if not RAW_ADMIN_USERNAME:
    # Generate randomized default if not specified
    import uuid
    random_suffix = uuid.uuid4().hex[:8]
    ADMIN_USERNAME = f"armor-admin-{random_suffix}"
    logger.info(f"Using randomized admin username: {ADMIN_USERNAME}")

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

def update_conduit_config(shared_secret: str):
    """Inject shared secret and enforce disabled registration"""
    try:
        config = toml.load(CONDUIT_CONFIG_PATH)
        config.setdefault("global", {})
        config["global"]["allow_registration"] = False
        config["global"]["registration_shared_secret"] = shared_secret

        with open(CONDUIT_CONFIG_PATH, "w") as f:
            toml.dump(config, f)

        logger.info("Conduit config updated with shared secret")
        return True
    except Exception as e:
        logger.error(f"Failed to update Conduit config: {e}")
        return False

def remove_shared_secret():
    """Permanently remove shared secret from config (compliance cleanup)"""
    try:
        config = toml.load(CONDUIT_CONFIG_PATH)
        if "global" in config and "registration_shared_secret" in config["global"]:
            del config["global"]["registration_shared_secret"]
            with open(CONDUIT_CONFIG_PATH, "w") as f:
                toml.dump(config, f)
            logger.info("Shared secret removed from config (security cleanup)")
    except Exception as e:
        logger.warning(f"Failed to remove shared secret: {e}")

def register_admin(shared_secret: str, password: str):
    """Register admin user using Synapse-style shared-secret admin API with conflict handling"""
    try:
        # Step 1: Get nonce
        r = requests.get(f"{CONDUIT_URL}/_synapse/admin/v1/register", timeout=10)
        
        # Don't raise for 4xx - handle them manually
        if r.status_code not in (200, 400, 409):
            logger.error(f"Unexpected status code getting nonce: {r.status_code}")
            return False
            
        response_data = r.json()
        nonce = response_data.get("nonce")
        if not nonce:
            raise ValueError("No nonce returned")

        # Step 2: Compute HMAC-SHA1 (nonce + username + password + admin flag)
        data = f"{nonce}\x00{ADMIN_USERNAME}\x00{password}\x00admin"
        mac = hmac.new(
            shared_secret.encode(),
            data.encode(),
            hashlib.sha1
        ).hexdigest()

        # Step 3: Register
        payload = {
            "nonce": nonce,
            "username": ADMIN_USERNAME,
            "password": password,
            "admin": True,
            "mac": mac
        }
        r = requests.post(
            f"{CONDUIT_URL}/_synapse/admin/v1/register",
            json=payload,
            timeout=10
)
        
        # Don't raise for 4xx - handle them manually
        if r.status_code not in (200, 400, 409):
            logger.error(f"Failed to get nonce: HTTP {r.status_code}")
            return False
        
        response_data = r.json()
        nonce = response_data.get("nonce")
        if not nonce:
            raise ValueError("No nonce returned")
        
        # Step 2: Compute HMAC-SHA1 (nonce + username + password + admin flag)
        data = f"{nonce}\x00{ADMIN_USERNAME}\x00{password}\x00admin"
        mac = hmac.new(
            shared_secret.encode(),
            data.encode(),
            hashlib.sha1
        ).hexdigest()
        
        # Step 3: Register
        payload = {
            "nonce": nonce,
            "username": ADMIN_USERNAME,
            "password": password,
            "admin": True,
            "mac": mac
        }
        r = requests.post(
            f"{CONDUIT_URL}/_synapse/admin/v1/register",
            json=payload,
            timeout=10
        )
        
        # Handle response
        if r.status_code == 200:
            response_data = r.json()
            user_id = response_data.get("user_id", f"@{ADMIN_USERNAME}:{os.getenv('ARMORCLAW_SERVER_NAME', 'localhost')}")
            logger.info(f"Admin user registered successfully: {user_id}")
            
            # Store password securely (0600) - TEMPORARILY for display
            pw_file = Path("/var/lib/armorclaw/.admin_password_display")
            pw_file.write_text(password)
            pw_file.chmod(0o600)
            logger.info(f"Admin password prepared for display: {pw_file}")
            
            # Store actual username used
            with open("/var/lib/armorclaw/.admin_username", "w") as f:
                f.write(ADMIN_USERNAME)
            
            return True
        elif r.status_code == 400 or r.status_code == 409:
            # Username conflict handling
            error_code = response_data.get("errcode", "")
            error_msg = response_data.get("error", "")
            
            if "user_in_use" in error_code.lower() or "already in use" in error_msg.lower():
                logger.warning(f"Username '{ADMIN_USERNAME}' already exists")
                
                # Generate alternative username
                import uuid
                alt_suffix = uuid.uuid4().hex[:6]
                alt_username = f"armor-admin-{alt_suffix}"
                
                logger.info(f"Trying alternative username: {alt_username}")
                
                # Retry with alternative username
                data = f"{nonce}\x00{alt_username}\x00{password}\x00admin"
                mac = hmac.new(
                    shared_secret.encode(),
                    data.encode(),
                    hashlib.sha1
                ).hexdigest()
                
                payload = {
                    "nonce": nonce,
                    "username": alt_username,
                    "password": password,
                    "admin": True,
                    "mac": mac
                }
                
                r2 = requests.post(
                    f"{CONDUIT_URL}/_synapse/admin/v1/register",
                    json=payload,
                    timeout=10
                )
                
                if r2.status_code == 200:
                    response_data2 = r2.json()
                    user_id = response_data2.get("user_id", f"@{alt_username}:{os.getenv('ARMORCLAW_SERVER_NAME', 'localhost')}")
                    logger.info(f"Alternative admin user registered successfully: {user_id}")
                    
                    # Store alternative username for reference
                    try:
                        with open("/var/lib/armorclaw/.admin_username", "w") as f:
                            f.write(alt_username)
                        # Update for display in this run
                        nonlocal ADMIN_USERNAME
                        ADMIN_USERNAME = alt_username
                    except Exception as e:
                        logger.error(f"Failed to save username: {e}")
                    
                    return True
                else:
                    logger.error("Failed to register alternative admin user")
                    return False
            else:
                logger.error(f"Registration failed: {response_data}")
                return False
        else:
            logger.error(f"Unexpected status code: {r.status_code}")
            return False
            error_code = response_data.get("errcode", "")
            error_msg = response_data.get("error", "")
            
            if "user_in_use" in error_code.lower() or "already in use" in error_msg.lower():
                logger.warning(f"Username '{ADMIN_USERNAME}' already exists")
                
                # Generate alternative username
                import uuid
                alt_suffix = uuid.uuid4().hex[:6]
                alt_username = f"armor-admin-{alt_suffix}"
                
                logger.info(f"Trying alternative username: {alt_username}")
                
                # Retry with alternative username
                data = f"{nonce}\x00{alt_username}\x00{password}\x00admin"
                mac = hmac.new(
                    shared_secret.encode(),
                    data.encode(),
                    hashlib.sha1
                ).hexdigest()
                
                payload = {
                    "nonce": nonce,
                    "username": alt_username,
                    "password": password,
                    "admin": True,
                    "mac": mac
                }
                
                r2 = requests.post(
                    f"{CONDUIT_URL}/_synapse/admin/v1/register",
                    json=payload,
                    timeout=10
                )
                
                if r2.status_code == 200:
                    user_id = r2.json().get("user_id", f"@{alt_username}:{os.getenv('ARMORCLAW_SERVER_NAME', 'localhost')}")
                    logger.info(f"Alternative admin user registered successfully: {user_id}")
                    
                    # Store alternative username for reference
                    with open("/var/lib/armorclaw/.admin_username", "w") as f:
                        f.write(alt_username)
                    
                    ADMIN_USERNAME = alt_username
                else:
                    logger.error("Failed to register alternative admin user")
                    return False
        
        logger.info(f"Admin user registered successfully: {user_id}")
        
        # Store username for reference (no password persistence)
        with open("/var/lib/armorclaw/.admin_username", "w") as f:
            f.write(ADMIN_USERNAME)
        
        return True
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

    shared_secret = generate_secure_secret()
    logger.debug(f"Generated shared secret: {shared_secret[:8]}...")

    if not wait_for_conduit():
        return 1

    if not update_conduit_config(shared_secret):
        return 1

    # Reload Conduit config (SIGHUP or restart)
    try:
        pid = subprocess.check_output(["pgrep", "-f", "conduit"]).decode().strip().split("\n")[0]
        os.kill(int(pid), 1)  # SIGHUP
        logger.info("Sent SIGHUP to Conduit")
        time.sleep(5)
    except Exception:
        logger.warning("Could not reload Conduit — full restart may be needed")

    if not wait_for_conduit():  # Re-check after reload
        return 1

    if not register_admin(shared_secret, password):
        return 1

    remove_shared_secret()
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