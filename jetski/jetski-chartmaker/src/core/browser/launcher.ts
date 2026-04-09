import { chromium, BrowserContext } from 'playwright';
import path from 'path';
import fs from 'fs';
import { logger } from '../../cli/utils/logger';
import { readFileSync } from 'fs';
import { StateCompiler, RecordedAction } from '../recorder/state-compiler';

export class BrowserLauncher {
  private context: BrowserContext | null = null;
  private recordedActions: RecordedAction[] = [];
  private outputFile?: string;

  async launch(sessionId: string, headless: boolean = false): Promise<BrowserContext> {
    const sessionDir = path.join('sessions', sessionId);
    fs.mkdirSync(sessionDir, { recursive: true });

    logger.info(`Launching browser with session: ${sessionId}`);
    logger.info(`Session data stored in: ${sessionDir}`);

    this.context = await chromium.launchPersistentContext(
      path.join(sessionDir, 'browser-data'),
      {
        headless,
        viewport: { width: 1280, height: 720 },
        args: [
          '--disable-blink-features=AutomationControlled',
        ],
        ignoreHTTPSErrors: true,
      }
    );

    logger.success('Browser launched successfully');
    return this.context;
  }

  async injectHUD(context: BrowserContext): Promise<void> {
    const hudScript = this.getHUDScript();

    await context.addInitScript({
      content: hudScript
    });

    logger.success('The Helm (HUD) injected into all frames');
  }

  private getHUDScript(): string {
    const hudHtml = readFileSync(path.join(__dirname, '../../../src/injectables/helm/hud.html'), 'utf-8');
    const hudCss = readFileSync(path.join(__dirname, '../../../src/injectables/helm/hud.css'), 'utf-8');
    const hudJsContent = readFileSync(path.join(__dirname, '../../../src/injectables/helm/hud.js'), 'utf-8');

    const hudCssEscaped = JSON.stringify(hudCss);
    const hudHtmlEscaped = JSON.stringify(hudHtml);

    return `
      (function() {
        'use strict';

        const container = document.createElement('div');
        container.id = 'jetski-helm-container';
        const shadow = container.attachShadow({ mode: 'closed' });

        shadow.innerHTML = \`
          <style>
            \${${hudCssEscaped}}
          </style>
          \${${hudHtmlEscaped}}
        \`;

        document.body.appendChild(container);

        ${hudJsContent}

        console.log('🧭 Jetski Chartmaker: The Helm is ready for navigation');
      })();
    `;
  }

  async exposeRecordingFunction(context: BrowserContext, outputFile: string): Promise<void> {
    this.outputFile = outputFile;

    await context.exposeFunction('jetskiRecord', (action: RecordedAction) => {
      this.recordedActions.push(action);
      logger.info(`⚓ Recorded action: ${action.action_type} on ${action.selector?.primary_css || 'unknown'}`);
    });

    await context.exposeFunction('jetskiRecordFromFrame', (action: RecordedAction) => {
      this.recordedActions.push(action);
      logger.info(`⚓ Recorded action from iframe: ${action.action_type} on ${action.selector?.primary_css || 'unknown'}`);
    });

    await context.exposeFunction('jetskiSave', async () => {
      if (this.recordedActions.length === 0) {
        logger.warn('No actions recorded, nothing to save');
        return;
      }

      await this.saveNavChart();
      logger.success(`✓ Nav-Chart saved to ${this.outputFile}`);
    });

    await context.exposeFunction('jetskiClear', () => {
      const count = this.recordedActions.length;
      this.recordedActions = [];
      logger.info(`⚓ Cleared ${count} recorded actions`);
    });

    await context.exposeFunction('jetskiError', (message: string) => {
      logger.error(`🧭 Jetski HUD Error: ${message}`);
    });

    logger.info('🧭 RPC bridge exposed: jetskiRecord, jetskiRecordFromFrame, jetskiSave, jetskiClear, jetskiError');
  }

  async setupCrossOriginBridge(context: BrowserContext): Promise<void> {
    await context.exposeFunction('jetskiRecordFromFrame', (action: RecordedAction) => {
      this.recordedActions.push(action);
      logger.info(`⚓ Cross-origin action recorded: ${action.action_type} on ${action.selector?.primary_css || 'unknown'}`);
    });

    await context.addInitScript({
      content: `
        window.addEventListener('message', (event) => {
          if (event.data?.type === 'JETSKI_RECORD' && event.data?.action) {
            if (window.jetskiRecordFromFrame) {
              window.jetskiRecordFromFrame(event.data.action);
            }
          }
        });
      `
    });

    logger.info('🧭 Cross-origin postMessage bridge established');
  }

  private async saveNavChart(): Promise<void> {
    if (!this.outputFile) {
      throw new Error('Output file not configured');
    }

    const compiler = new StateCompiler();
    const sessionId = path.basename(this.outputFile, path.extname(this.outputFile));

    const navChart = compiler.compileEvents(this.recordedActions, {
      sessionId,
      addPostActionWaits: true
    });

    await StateCompiler.saveToFile(navChart, this.outputFile);
  }

  getRecordedActions(): RecordedAction[] {
    return [...this.recordedActions];
  }

  async close(): Promise<void> {
    if (this.context) {
      await this.context.close();
      this.context = null;
      logger.info('Browser context closed');
    }
  }

  getContext(): BrowserContext | null {
    return this.context;
  }
}
