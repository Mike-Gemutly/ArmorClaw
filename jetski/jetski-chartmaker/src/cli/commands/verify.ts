import fs from 'fs';
import { isValidNavChart, NavChart } from '../../core/validator/schema';
import { BrowserLauncher } from '../../core/browser/launcher';
import { logger } from '../utils/logger';

export async function verifyCommand(filePath: string, options: {
  verbose?: boolean;
  headless?: boolean;
}): Promise<void> {
  logger.info(`🔍 Running local sea trial for: ${filePath}`);

  if (!fs.existsSync(filePath)) {
    logger.error(`Nav-Chart file not found: ${filePath}`);
    process.exit(1);
  }

  try {
    const content = fs.readFileSync(filePath, 'utf-8');
    const navChart = JSON.parse(content);

    logger.info('Validating Nav-Chart structure...');
    if (!isValidNavChart(navChart)) {
      logger.error('Nav-Chart validation failed');
      process.exit(1);
    }

    logger.success('Nav-Chart structure is valid');

    if (options.verbose) {
      logger.info(`Target Domain: ${navChart.target_domain}`);
      logger.info(`Version: ${navChart.version}`);
      logger.info(`Actions: ${Object.keys(navChart.action_map || {}).length}`);
    }

    if (options.headless) {
      await runDryRun(navChart, options.verbose || false);
    }

    logger.success('Local sea trial completed successfully');

  } catch (error) {
    logger.error(`Verification failed: ${error}`);
    process.exit(1);
  }
}

async function runDryRun(navChart: NavChart, verbose: boolean): Promise<void> {
  logger.info('Running dry-run execution...');

  const launcher = new BrowserLauncher();
  try {
    const context = await launcher.launch('dry-run-session', true);
    const page = context.pages()[0] || await context.newPage();

    await page.goto(navChart.target_domain);
    logger.info(`Navigated to ${navChart.target_domain}`);

    for (const [actionName, action] of Object.entries(navChart.action_map || {})) {
      if (verbose) {
        logger.info(`Executing action: ${actionName}`);
      }

      const actionData = action as NavChart['action_map'][string];
      switch (actionData.action_type) {
        case 'navigate':
          if (actionData.url) {
            await page.goto(actionData.url);
          }
          break;
        case 'click':
          if (actionData.selector?.primary_css) {
            await page.click(actionData.selector.primary_css);
          }
          break;
        case 'input':
          if (actionData.selector?.primary_css && actionData.value) {
            await page.fill(actionData.selector.primary_css, actionData.value);
          }
          break;
      }
    }

    logger.success('Dry-run execution completed');
  } finally {
    await launcher.close();
  }
}
