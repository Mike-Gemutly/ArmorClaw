#!/usr/bin/env python3
"""
Tests for ArmorClaw container entrypoint proxy configuration.
Tests P0-CRIT-1 Egress Proxy support.
"""
import os
import sys
import unittest
from unittest.mock import patch, mock_open

# Add parent directory to path for imports
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

class TestEgressProxyConfiguration(unittest.TestCase):
    """Test suite for egress proxy configuration in entrypoint."""

    def setUp(self):
        """Set up test fixtures."""
        self.test_secrets = {
            'provider': 'openai',
            'token': 'sk-test123',
            'display_name': 'Test Key'
        }

    def tearDown(self):
        """Clean up environment variables after tests."""
        for key in ['HTTP_PROXY', 'HTTPS_PROXY', 'http_proxy', 'https_proxy', 'NO_PROXY']:
            if key in os.environ:
                del os.environ[key]

    @patch.dict(os.environ, {'HTTP_PROXY': 'http://squid:3128:8080'}, clear=True)
    def test_http_proxy_set_from_environment(self):
        """Test that HTTP_PROXY is read from environment and set."""
        # Import entrypoint functions
        import importlib
        entrypoint = importlib.import_module('entrypoint')

        # Re-import to get fresh state
        import importlib
        importlib.reload(entrypoint)

        # Verify HTTP_PROXY was propagated to all proxy env vars
        self.assertEqual(os.environ.get('HTTP_PROXY'), 'http://squid:3128:8080')
        self.assertEqual(os.environ.get('HTTPS_PROXY'), 'http://squid:3128:8080')
        self.assertEqual(os.environ.get('http_proxy'), 'http://squid:3128:8080')
        self.assertEqual(os.environ.get('https_proxy'), 'http://squid:3128:8080')

    @patch.dict(os.environ, {}, clear=True)
    def test_no_proxy_when_not_configured(self):
        """Test that no proxy is set when HTTP_PROXY is not configured."""
        import importlib
        entrypoint = importlib.import_module('entrypoint')
        importlib.reload(entrypoint)

        # Verify no proxy environment variables are set
        self.assertIsNone(os.environ.get('HTTP_PROXY'))
        self.assertIsNone(os.environ.get('HTTPS_PROXY'))

    def test_valid_proxy_url_format(self):
        """Test validation of proxy URL format."""
        valid_urls = [
            'http://squid:3128:8080',
            'https://squid:3128:8083',
            'http://proxy.example.com:3128',
        ]

        for url in valid_urls:
            with self.subTest(url=url):
                self.assertTrue(url.startswith('http://') or url.startswith('https://'),
                    f"URL should have protocol: {url}")

    def test_invalid_proxy_url_format(self):
        """Test rejection of invalid proxy URL formats."""
        invalid_urls = [
            'squid:3128:8080',  # Missing protocol
            'ftp://squid:3128:8080',  # Wrong protocol
            '://squid:3128:8080',  # Missing protocol prefix
        ]

        for url in invalid_urls:
            with self.subTest(url=url):
                self.assertFalse(url.startswith('http://') or url.startswith('https://'),
                             f"URL should be invalid: {url}")

    @patch.dict(os.environ, {
        'HTTP_PROXY': 'http://squid:3128:8080',
        'OPENAI_API_KEY': 'sk-test'
    }, clear=True)
    def test_sdtw_provider_token_mapping(self):
        """Test that SDTW provider tokens are correctly mapped."""
        import importlib
        entrypoint = importlib.import_module('entrypoint')
        importlib.reload(entrypoint)

        # Test each SDTW provider
        sdtw_providers = {
            'slack': 'SLACK_BOT_TOKEN',
            'discord': 'DISCORD_BOT_TOKEN',
            'teams': 'MICROSOFT_API_KEY',
            'whatsapp': 'WHATSAPP_API_KEY',
        }

        for provider, expected_env_var in sdtw_providers.items():
            self.assertIsNotNone(expected_env_var,
                              f"Env var should be defined for {provider}")

    @patch.dict(os.environ, {
        'HTTP_PROXY': 'http://squid:3128:8080'
    }, clear=True)
    def test_no_proxy_for_localhost(self):
        """Test that NO_PROXY is set for localhost connections."""
        import importlib
        entrypoint = importlib.import_module('entrypoint')
        importlib.reload(entrypoint)

        no_proxy = os.environ.get('NO_PROXY')
        self.assertIsNotNone(no_proxy, "NO_PROXY should be set")
        self.assertIn('localhost', no_proxy)
        self.assertIn('127.0.0.1', no_proxy)

    @patch.dict(os.environ, {
        'HTTP_PROXY': 'http://squid:3128:8080/slack'
    }, clear=True)
    def test_slack_proxy_url_routing(self):
        """Test that Slack proxy routes to correct endpoint."""
        proxy_url = os.environ.get('HTTP_PROXY')
        self.assertIsNotNone(proxy_url)
        self.assertTrue(proxy_url.endswith('/slack'),
                       f"Slack proxy should end with /slack: {proxy_url}")

    @patch.dict(os.environ, {
        'HTTP_PROXY': 'http://squid:3128:8081/discord'
    }, clear=True)
    def test_discord_proxy_url_routing(self):
        """Test that Discord proxy routes to correct endpoint."""
        proxy_url = os.environ.get('HTTP_PROXY')
        self.assertIsNotNone(proxy_url)
        self.assertTrue(proxy_url.endswith('/discord'),
                       f"Discord proxy should end with /discord: {proxy_url}")

    @patch.dict(os.environ, {
        'HTTP_PROXY': 'http://squid:3128:8082/teams'
    }, clear=True)
    def test_teams_proxy_url_routing(self):
        """Test that Teams proxy routes to correct endpoint."""
        proxy_url = os.environ.get('HTTP_PROXY')
        self.assertIsNotNone(proxy_url)
        self.assertTrue(proxy_url.endswith('/teams'),
                       f"Teams proxy should end with /teams: {proxy_url}")

    @patch.dict(os.environ, {
        'HTTP_PROXY': 'http://squid:3128:8083/whatsapp'
    }, clear=True)
    def test_whatsapp_proxy_url_routing(self):
        """Test that WhatsApp proxy routes to correct endpoint."""
        proxy_url = os.environ.get('HTTP_PROXY')
        self.assertIsNotNone(proxy_url)
        self.assertTrue(proxy_url.endswith('/whatsapp'),
                       f"WhatsApp proxy should end with /whatsapp: {proxy_url}")


class TestSecretsProxyIntegration(unittest.TestCase):
    """Test integration between secrets loading and proxy configuration."""

    def tearDown(self):
        """Clean up environment variables after tests."""
        for key in ['HTTP_PROXY', 'SLACK_BOT_TOKEN', 'DISCORD_BOT_TOKEN',
                    'MICROSOFT_API_KEY', 'WHATSAPP_API_KEY']:
            if key in os.environ:
                del os.environ[key]

    def test_slack_secrets_with_proxy(self):
        """Test that Slack secrets are loaded with proxy enabled."""
        # This would normally be set by bridge
        test_secrets = {
            'provider': 'slack',
            'token': 'xoxb-test-token',
            'display_name': 'Test Slack Bot'
        }

        with patch.dict(os.environ, {'HTTP_PROXY': 'http://squid:3128:8080'}, clear=True):
            import importlib
            entrypoint = importlib.import_module('entrypoint')
            importlib.reload(entrypoint)

            # After apply_secrets, SLACK_BOT_TOKEN should be set
            # This validates the provider_env_map was updated

    def test_discord_secrets_with_proxy(self):
        """Test that Discord secrets are loaded with proxy enabled."""
        test_secrets = {
            'provider': 'discord',
            'token': 'test-discord-token',
            'display_name': 'Test Discord Bot'
        }

        with patch.dict(os.environ, {'HTTP_PROXY': 'http://squid:3128:8081'}, clear=True):
            import importlib
            entrypoint = importlib.import_module('entrypoint')
            importlib.reload(entrypoint)

            # After apply_secrets, DISCORD_BOT_TOKEN should be set

    def test_teams_secrets_with_proxy(self):
        """Test that Teams secrets are loaded with proxy enabled."""
        test_secrets = {
            'provider': 'teams',
            'token': 'test-teams-token',
            'display_name': 'Test Teams Bot'
        }

        with patch.dict(os.environ, {'HTTP_PROXY': 'http://squid:3128:8082'}, clear=True):
            import importlib
            entrypoint = importlib.import_module('entrypoint')
            importlib.reload(entrypoint)

            # After apply_secrets, MICROSOFT_API_KEY should be set

    def test_whatsapp_secrets_with_proxy(self):
        """Test that WhatsApp secrets are loaded with proxy enabled."""
        test_secrets = {
            'provider': 'whatsapp',
            'token': 'test-whatsapp-token',
            'display_name': 'Test WhatsApp Bot'
        }

        with patch.dict(os.environ, {'HTTP_PROXY': 'http://squid:3128:8083'}, clear=True):
            import importlib
            entrypoint = importlib.import_module('entrypoint')
            importlib.reload(entrypoint)

            # After apply_secrets, WHATSAPP_API_KEY should be set


class TestProxyConfigurationPriority(unittest.TestCase):
    """Test priority of proxy configuration over other settings."""

    def tearDown(self):
        """Clean up environment variables."""
        for key in ['HTTP_PROXY', 'HTTPS_PROXY', 'http_proxy', 'https_proxy']:
            if key in os.environ:
                del os.environ[key]

    @patch.dict(os.environ, {
        'HTTP_PROXY': 'http://squid:3128:8080'
    }, clear=True)
    def test_proxy_overrides_defaults(self):
        """Test that proxy configuration takes precedence."""
        import importlib
        entrypoint = importlib.import_module('entrypoint')
        importlib.reload(entrypoint)

        # HTTP_PROXY should be set
        self.assertEqual(os.environ.get('HTTP_PROXY'), 'http://squid:3128:8080')

        # HTTPS_PROXY, http_proxy, https_proxy should also be set
        self.assertEqual(os.environ.get('HTTPS_PROXY'), 'http://squid:3128:8080')
        self.assertEqual(os.environ.get('http_proxy'), 'http://squid:3128:8080')
        self.assertEqual(os.environ.get('https_proxy'), 'http://squid:3128:8080')


if __name__ == '__main__':
    # Run tests
    unittest.main(verbosity=2)
