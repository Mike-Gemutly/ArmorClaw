#!/usr/bin/env node
/**
 * ArmorClaw + OpenClaw Integration Entrypoint
 *
 * This is the main entry point for running OpenClaw inside ArmorClaw.
 * It initializes the bridge connection and starts the OpenClaw agent.
 */

import { ArmorClawBridgeClient } from './bridge-client.ts';

// Configuration from environment
const BRIDGE_SOCKET = process.env.ARMORCLAW_BRIDGE_SOCKET || '/run/armorclaw/bridge.sock';
const MATRIX_ROOM = process.env.ARMORCLAW_MATRIX_ROOM || '';
const SECRETS_PATH = process.env.ARMORCLAW_SECRETS_PATH || '/run/secrets';
const VERBOSE = process.env.ARMORCLAW_VERBOSE === 'true';

interface AgentConfig {
  apiKey?: string;
  modelProvider?: string;
  modelName?: string;
}

/**
 * Logger with ArmorClaw prefix
 */
function log(level: string, message: string, ...args: unknown[]) {
  const timestamp = new Date().toISOString();
  console.error(`[${timestamp}] [${level}] [ArmorClaw] ${message}`, ...args);
}

/**
 * Load secrets from bridge or file system
 */
async function loadSecrets(client: ArmorClawBridgeClient): Promise<AgentConfig> {
  const config: AgentConfig = {};

  // Try to get secrets from bridge first
  try {
    const secretKeys = await client.listSecrets();

    for (const key of secretKeys.keys) {
      const secret = await client.getSecret(key);
      if (secret.value) {
        switch (key.toLowerCase()) {
          case 'openai_api_key':
            config.apiKey = secret.value;
            config.modelProvider = 'openai';
            break;
          case 'anthropic_api_key':
            config.apiKey = secret.value;
            config.modelProvider = 'anthropic';
            break;
          case 'google_api_key':
          case 'gemini_api_key':
            config.apiKey = secret.value;
            config.modelProvider = 'google';
            break;
          default:
            // Store other secrets as env vars
            process.env[key.toUpperCase()] = secret.value;
        }
      }
    }

    log('info', `Loaded ${secretKeys.keys.length} secrets from bridge`);
  } catch (error) {
    log('warn', 'Could not load secrets from bridge, checking file system');

    // Fallback to file system secrets
    const fs = await import('fs');
    const path = await import('path');

    try {
      const files = fs.readdirSync(SECRETS_PATH);
      for (const file of files) {
        try {
          const content = fs.readFileSync(path.join(SECRETS_PATH, file), 'utf8').trim();
          const key = file.replace(/\.json$/, '');

          // Try to parse as JSON first
          try {
            const secrets = JSON.parse(content);
            Object.assign(process.env, secrets);
          } catch {
            // Plain text secret
            process.env[key.toUpperCase()] = content;
          }
        } catch (e) {
          log('warn', `Failed to read secret file: ${file}`);
        }
      }
    } catch (e) {
      log('error', 'No secrets available');
    }
  }

  return config;
}

/**
 * Main agent loop
 */
async function runAgent(client: ArmorClawBridgeClient, config: AgentConfig) {
  log('info', 'Starting ArmorClaw-OpenClaw agent...');
  log('info', `Model provider: ${config.modelProvider || 'not configured'}`);
  log('info', `Matrix room: ${MATRIX_ROOM || 'not configured'}`);

  let lastEventId: string | null = null;

  while (true) {
    try {
      // Poll for new messages
      const events = await client.matrixReceive(MATRIX_ROOM, 10);

      for (const event of events) {
        // Skip already processed events
        if (lastEventId && event.event_id === lastEventId) continue;

        // Only process text messages
        if (event.type === 'm.room.message' && event.content.msgtype === 'm.text') {
          const body = event.content.body || '';
          const sender = event.sender;

          log('info', `Message from ${sender}: ${body.substring(0, 100)}...`);

          // Process message and generate response
          const response = await processMessage(body, sender, config);

          if (response) {
            await client.matrixSend(event.room_id, response);
            log('info', `Sent response to ${event.room_id}`);
          }
        }

        lastEventId = event.event_id;
      }

      // Wait before next poll
      await new Promise(resolve => setTimeout(resolve, 1000));

    } catch (error) {
      log('error', `Agent loop error: ${error}`);
      await new Promise(resolve => setTimeout(resolve, 5000));
    }
  }
}

/**
 * Process an incoming message and generate a response
 */
async function processMessage(
  body: string,
  sender: string,
  config: AgentConfig
): Promise<string | null> {
  // Handle special commands
  if (body.startsWith('/')) {
    const [command, ...args] = body.slice(1).split(/\s+/);

    switch (command.toLowerCase()) {
      case 'status':
        return `✅ ArmorClaw-OpenClaw agent running\nProvider: ${config.modelProvider || 'none'}\nSender: ${sender}`;

      case 'help':
        return `🦞 ArmorClaw-OpenClaw Agent Commands:
/status - Show agent status
/help - Show this help message
/model - Show current model
Any other message will be processed by the AI model`;

      case 'model':
        return `Current model: ${config.modelName || 'default'} (${config.modelProvider || 'not configured'})`;

      default:
        // Unknown command, treat as regular message
        break;
    }
  }

  // If no API key configured, return placeholder
  if (!config.apiKey) {
    return `⚠️ No AI model configured. Please set up API keys in ArmorClaw keystore.\n\nYour message was: "${body.substring(0, 100)}..."`;
  }

  // TODO: Integrate with actual OpenClaw agent here
  // For now, return a placeholder response
  return `🦞 OpenClaw (via ArmorClaw) received your message.

To enable full AI responses, integrate this with the OpenClaw gateway.

Your message: "${body.substring(0, 200)}${body.length > 200 ? '...' : ''}"`;
}

/**
 * Main entry point
 */
async function main() {
  log('info', '=== ArmorClaw-OpenClaw Integration ===');
  log('info', `Bridge socket: ${BRIDGE_SOCKET}`);
  log('info', `Node version: ${process.version}`);

  // Create bridge client
  const client = new ArmorClawBridgeClient({ socketPath: BRIDGE_SOCKET });

  // Wait for bridge to be available
  let retries = 0;
  const maxRetries = 30;

  while (retries < maxRetries) {
    try {
      const health = await client.health();
      log('info', `Bridge health: ${health.status}`);
      break;
    } catch (error) {
      retries++;
      log('warn', `Waiting for bridge... (${retries}/${maxRetries})`);
      await new Promise(resolve => setTimeout(resolve, 1000));
    }
  }

  if (retries >= maxRetries) {
    log('error', 'Bridge not available after 30 seconds');
    process.exit(1);
  }

  // Get bridge status
  try {
    const status = await client.status();
    log('info', `Bridge version: ${status.version}`);
    log('info', `Active containers: ${status.containers}`);
  } catch (error) {
    log('warn', `Could not get bridge status: ${error}`);
  }

  // Check Matrix status
  try {
    const matrixStatus = await client.matrixStatus();
    log('info', `Matrix enabled: ${matrixStatus.enabled}`);
    log('info', `Matrix connected: ${matrixStatus.connected}`);

    if (matrixStatus.user_id) {
      log('info', `Matrix user: ${matrixStatus.user_id}`);
    }
  } catch (error) {
    log('warn', `Could not get Matrix status: ${error}`);
  }

  // Load secrets and configuration
  const config = await loadSecrets(client);

  // Start the agent loop
  await runAgent(client, config);
}

// Handle shutdown gracefully
process.on('SIGTERM', () => {
  log('info', 'Received SIGTERM, shutting down...');
  process.exit(0);
});

process.on('SIGINT', () => {
  log('info', 'Received SIGINT, shutting down...');
  process.exit(0);
});

// Run main
main().catch((error) => {
  log('error', `Fatal error: ${error}`);
  process.exit(1);
});
