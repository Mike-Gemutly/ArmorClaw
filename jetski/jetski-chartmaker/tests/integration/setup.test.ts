import { withTestFixture, getFixturePath } from './test-utils';

describe('Integration Test Infrastructure', () => {
  it('should launch browser and navigate to test page', async () => {
    await withTestFixture(async (fixture) => {
      const fixturePath = getFixturePath('test-page.html');
      await fixture.navigateTo(`file://${fixturePath}`);

      const title = await fixture.getPage().title();
      expect(title).toBe('Test Page for Jetski Chartmaker');
    });
  });

  it('should find and click button with data-automation-id', async () => {
    await withTestFixture(async (fixture) => {
      const fixturePath = getFixturePath('test-page.html');
      await fixture.navigateTo(`file://${fixturePath}`);

      await fixture.click('#test-button-1');
      await fixture.waitForSelector('#result');

      const resultText = await fixture.getPage().textContent('#result');
      expect(resultText).toBe('Button 1 clicked!');
    });
  });

  it('should take screenshot without errors', async () => {
    await withTestFixture(async (fixture) => {
      const fixturePath = getFixturePath('test-page.html');
      await fixture.navigateTo(`file://${fixturePath}`);

      await fixture.takeScreenshot('test-infrastructure-screenshot.png');

      const fs = await import('fs');
      const evidencePath = require('./test-utils').getEvidencePath('test-infrastructure-screenshot.png');
      expect(fs.existsSync(evidencePath)).toBe(true);
    });
  });
});
