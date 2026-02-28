/**
 * @fileoverview Browser Service HTTP Server
 * Express-based HTTP API for headless browser automation
 */

import express, { Request, Response, NextFunction } from 'express';
import { BrowserClient, getBrowser, initializeBrowser, closeBrowser } from './browser';
import type {
  NavigateCommand,
  FillCommand,
  ClickCommand,
  WaitCommand,
  ExtractCommand,
  ScreenshotCommand,
  BrowserResponse,
  BrowserSession,
} from './types';

//=============================================================================
// Server Configuration
//=============================================================================

const PORT = process.env.PORT || 3000;
const LOG_LEVEL = process.env.LOG_LEVEL || 'info';

const app = express();

// Middleware
app.use(express.json());

//=============================================================================
// Logging
//=============================================================================

function log(level: string, message: string, data?: Record<string, unknown>): void {
  if (level === 'error' || LOG_LEVEL === 'debug') {
    const timestamp = new Date().toISOString();
    console.log(JSON.stringify({ timestamp, level, message, ...(data || {}) }));
  }
}

//=============================================================================
// Request Validation
//=============================================================================

function validateNavigate(req: Request): NavigateCommand {
  if (!req.body.url) {
    throw new Error('url is required');
  }
  return {
    url: req.body.url,
    waitUntil: req.body.waitUntil,
    timeout: req.body.timeout,
  };
}

function validateFill(req: Request): FillCommand {
  if (!req.body.fields || !Array.isArray(req.body.fields)) {
    throw new Error('fields array is required');
  }
  return {
    fields: req.body.fields,
    auto_submit: req.body.auto_submit,
    submit_delay: req.body.submit_delay,
  };
}

function validateClick(req: Request): ClickCommand {
  if (!req.body.selector) {
    throw new Error('selector is required');
  }
  return {
    selector: req.body.selector,
    waitFor: req.body.waitFor,
    timeout: req.body.timeout,
  };
}

function validateWait(req: Request): WaitCommand {
  if (!req.body.condition || !req.body.value) {
    throw new Error('condition and value are required');
  }
  return {
    condition: req.body.condition,
    value: req.body.value,
    timeout: req.body.timeout,
  };
}

function validateExtract(req: Request): ExtractCommand {
  if (!req.body.fields || !Array.isArray(req.body.fields)) {
    throw new Error('fields array is required');
  }
  return {
    fields: req.body.fields,
  };
}

function validateScreenshot(req: Request): ScreenshotCommand {
  return {
    fullPage: req.body.fullPage,
    selector: req.body.selector,
    format: req.body.format,
  };
}

//=============================================================================
// Error Handling
//=============================================================================

function handleError(err: Error, req: Request, res: Response, _next: NextFunction): void {
  log('error', 'Request failed', { error: err.message, path: req.path });

  const response: BrowserResponse = {
    success: false,
    error: {
      code: 'BROWSER_NOT_READY',
      message: err.message,
    },
    duration: 0,
  };

  res.status(500).json(response);
}

//=============================================================================
// Routes
//=============================================================================

// Health check
app.get('/health', (_req: Request, res: Response): void => {
  const browser = getBrowser();
  const session = browser.getSession();

  res.json({
    status: 'healthy',
    browser: session ? 'initialized' : 'not_initialized',
    state: browser.getState(),
    timestamp: new Date().toISOString(),
  });
});

// Initialize browser
app.post('/initialize', async (_req: Request, res: Response, next: NextFunction): Promise<void> => {
  try {
    await initializeBrowser();
    log('info', 'Browser initialized');

    res.json({
      success: true,
      message: 'Browser initialized',
      timestamp: new Date().toISOString(),
    });
  } catch (error) {
    next(error);
  }
});

// Close browser
app.post('/close', async (_req: Request, res: Response, next: NextFunction): Promise<void> => {
  try {
    await closeBrowser();
    log('info', 'Browser closed');

    res.json({
      success: true,
      message: 'Browser closed',
      timestamp: new Date().toISOString(),
    });
  } catch (error) {
    next(error);
  }
});

// Get session info
app.get('/session', (_req: Request, res: Response): void => {
  const browser = getBrowser();
  const session = browser.getSession();

  if (!session) {
    res.json({
      success: false,
      error: 'No active session',
    });
    return;
  }

  res.json({
    success: true,
    data: {
      id: session.id,
      state: session.state,
      currentUrl: session.currentUrl,
      createdAt: session.createdAt.toISOString(),
      lastActivity: session.lastActivity.toISOString(),
    },
  });
});

// Navigate to URL
app.post('/navigate', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  try {
    const command = validateNavigate(req);
    log('info', 'Navigating', { url: command.url });

    const browser = getBrowser();
    const result = await browser.navigate(command);

    res.json(result);
  } catch (error) {
    next(error);
  }
});

// Fill form fields
app.post('/fill', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  try {
    const command = validateFill(req);
    log('info', 'Filling form', { fieldCount: command.fields.length });

    const browser = getBrowser();
    const result = await browser.fill(command);

    res.json(result);
  } catch (error) {
    next(error);
  }
});

// Click element
app.post('/click', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  try {
    const command = validateClick(req);
    log('info', 'Clicking element', { selector: command.selector });

    const browser = getBrowser();
    const result = await browser.click(command);

    res.json(result);
  } catch (error) {
    next(error);
  }
});

// Wait for condition
app.post('/wait', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  try {
    const command = validateWait(req);
    log('info', 'Waiting', { condition: command.condition });

    const browser = getBrowser();
    const result = await browser.wait(command);

    res.json(result);
  } catch (error) {
    next(error);
  }
});

// Extract data
app.post('/extract', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  try {
    const command = validateExtract(req);
    log('info', 'Extracting data', { fieldCount: command.fields.length });

    const browser = getBrowser();
    const result = await browser.extract(command);

    res.json(result);
  } catch (error) {
    next(error);
  }
});

// Take screenshot
app.post('/screenshot', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  try {
    const command = validateScreenshot(req);
    log('info', 'Taking screenshot', { fullPage: command.fullPage });

    const browser = getBrowser();
    const result = await browser.screenshot(command);

    res.json(result);
  } catch (error) {
    next(error);
  }
});

// Combined workflow endpoint
app.post('/workflow', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  const startTime = Date.now();

  try {
    const steps = req.body.steps;
    if (!Array.isArray(steps) || steps.length === 0) {
      throw new Error('steps array is required');
    }

    log('info', 'Executing workflow', { stepCount: steps.length });

    const browser = getBrowser();
    const results: BrowserResponse[] = [];

    for (let i = 0; i < steps.length; i++) {
      const step = steps[i];
      log('debug', `Executing step ${i + 1}`, { action: step.action });

      let result: BrowserResponse;

      switch (step.action) {
        case 'navigate':
          result = await browser.navigate(step);
          break;
        case 'fill':
          result = await browser.fill(step);
          break;
        case 'click':
          result = await browser.click(step);
          break;
        case 'wait':
          result = await browser.wait(step);
          break;
        case 'extract':
          result = await browser.extract(step);
          break;
        case 'screenshot':
          result = await browser.screenshot(step);
          break;
        default:
          throw new Error(`Unknown action: ${step.action}`);
      }

      results.push(result);

      // Stop workflow on failure
      if (!result.success) {
        log('warn', 'Workflow stopped due to failure', { step: i + 1 });
        break;
      }
    }

    res.json({
      success: results.every(r => r.success),
      data: {
        steps: results,
        totalSteps: steps.length,
        completedSteps: results.length,
      },
      duration: Date.now() - startTime,
    });
  } catch (error) {
    next(error);
  }
});

//=============================================================================
// Error Handler
//=============================================================================

app.use(handleError);

//=============================================================================
// Start Server
//=============================================================================

async function startServer(): Promise<void> {
  try {
    // Pre-initialize browser on startup (optional)
    if (process.env.INIT_ON_START === 'true') {
      log('info', 'Pre-initializing browser');
      await initializeBrowser();
    }

    app.listen(PORT, () => {
      log('info', `Browser service listening on port ${PORT}`);
    });
  } catch (error) {
    log('error', 'Failed to start server', { error: (error as Error).message });
    process.exit(1);
  }
}

// Graceful shutdown
process.on('SIGTERM', async () => {
  log('info', 'Received SIGTERM, shutting down');
  await closeBrowser();
  process.exit(0);
});

process.on('SIGINT', async () => {
  log('info', 'Received SIGINT, shutting down');
  await closeBrowser();
  process.exit(0);
});

// Start the server
startServer();
