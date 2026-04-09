import fs from 'fs';
import path from 'path';
import { ConfigManager } from '../utils/config';
import { logger } from '../utils/logger';

export async function initCommand(options: { dir: string }): Promise<void> {
  const dir = options.dir;
  const configManager = new ConfigManager(path.join(dir, 'jetski.config.json'));

  logger.info('Initializing Jetski Chartmaker project...');

  const structure = [
    { type: 'dir', path: path.join(dir, 'charts'), desc: 'Nav-Charts directory' },
    { type: 'dir', path: path.join(dir, 'sessions'), desc: 'Browser session storage' },
  ];

  for (const item of structure) {
    if (item.type === 'dir') {
      fs.mkdirSync(item.path, { recursive: true });
      logger.success(`Created directory: ${path.relative(dir, item.path)}`);
    }
  }

  const initialConfig = {
    version: 1,
    defaultTimeout: 5000,
    screenshotOnFailure: true,
  };

  configManager.saveConfig(initialConfig);
  logger.success(`Created configuration file: jetski.config.json`);

  console.log('\n🎉 Jetski Chartmaker initialized!');
  logger.info('Next: Run `jetski-chartmaker map --url https://example.com`');
}
