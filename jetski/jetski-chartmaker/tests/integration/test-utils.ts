import { chromium, BrowserContext, Page, Browser } from 'playwright';
import path from 'path';
import fs from 'fs';

export interface TestFixtureOptions {
  headless?: boolean;
  viewport?: { width: number; height: number };
}

export class TestFixture {
  private browser?: Browser;
  private context?: BrowserContext;
  private page?: Page;

  async setup(options: TestFixtureOptions = {}): Promise<Page> {
    this.browser = await chromium.launch({
      headless: options.headless ?? true
    });

    this.context = await this.browser.newContext({
      viewport: options.viewport ?? { width: 1280, height: 720 }
    });

    this.page = await this.context.newPage();

    return this.page;
  }

  async teardown(): Promise<void> {
    if (this.context) {
      await this.context.close();
    }
    if (this.browser) {
      await this.browser.close();
    }
  }

  getPage(): Page {
    if (!this.page) {
      throw new Error('Page not initialized. Call setup() first.');
    }
    return this.page;
  }

  async navigateTo(url: string): Promise<void> {
    const page = this.getPage();
    await page.goto(url, { waitUntil: 'networkidle' });
  }

  async takeScreenshot(filename: string): Promise<void> {
    const page = this.getPage();
    const screenshotDir = path.join(process.cwd(), '.sisyphus', 'evidence');
    fs.mkdirSync(screenshotDir, { recursive: true });
    await page.screenshot({ path: path.join(screenshotDir, filename) });
  }

  async injectScript(content: string): Promise<void> {
    const page = this.getPage();
    await page.addInitScript(content);
  }

  async executeScript(script: string): Promise<any> {
    const page = this.getPage();
    return await page.evaluate(script);
  }

  async waitForSelector(selector: string, timeout = 5000): Promise<void> {
    const page = this.getPage();
    await page.waitForSelector(selector, { timeout });
  }

  async click(selector: string): Promise<void> {
    const page = this.getPage();
    await page.click(selector);
  }

  async fill(selector: string, value: string): Promise<void> {
    const page = this.getPage();
    await page.fill(selector, value);
  }
}

export async function createTestFixture(): Promise<TestFixture> {
  return new TestFixture();
}

export async function withTestFixture<T>(
  testFn: (fixture: TestFixture) => Promise<T>
): Promise<T> {
  const fixture = new TestFixture();
  try {
    await fixture.setup({ headless: true });
    return await testFn(fixture);
  } finally {
    await fixture.teardown();
  }
}

export function getFixturePath(fixtureName: string): string {
  return path.join(__dirname, 'fixtures', fixtureName);
}

export function readFixture(fixtureName: string): string {
  const fixturePath = getFixturePath(fixtureName);
  return fs.readFileSync(fixturePath, 'utf-8');
}

export function getEvidencePath(filename: string): string {
  const evidenceDir = path.join(process.cwd(), '.sisyphus', 'evidence');
  fs.mkdirSync(evidenceDir, { recursive: true });
  return path.join(evidenceDir, filename);
}
