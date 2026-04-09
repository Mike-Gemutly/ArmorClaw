import { BrowserLauncher } from '../../core/browser/launcher';
import { logger } from '../utils/logger';

function waitForManualExit(): Promise<never> {
  return new Promise(() => {});
}

export async function mapCommand(options: {
  url?: string;
  output: string;
  session?: string;
}): Promise<void> {
  logger.info('🧭 Charting new navigation route...');

  if (!options.url) {
    logger.warn('No URL provided, will start with blank page');
  }

  const sessionId = options.session || `session-${Date.now()}`;
  logger.info(`Using session: ${sessionId}`);

  const launcher = new BrowserLauncher();
  
  try {
    const context = await launcher.launch(sessionId, false);

    if (options.url) {
      const page = context.pages()[0] || await context.newPage();
      logger.info(`Navigating to: ${options.url}`);
      await page.goto(options.url);
    }

    logger.success('Browser launched successfully');
    logger.info('The Helm (HUD) should be visible in the top-right corner');
    logger.info('Click elements to record them, then click Save to export Nav-Chart');

    await launcher.injectHUD(context);
    await launcher.exposeRecordingFunction(context, options.output);

    logger.info('Waiting for you to complete the charting session...');
    logger.info('Press Ctrl+C to exit (your work will be saved)');

    await waitForManualExit();

  } catch (error) {
    logger.error(`Charting session failed: ${error}`);
    process.exit(1);
  } finally {
    await launcher.close();
  }
}
