import { withTestFixture, getFixturePath, getEvidencePath } from './test-utils';
import { StateCompiler } from '../../src/core/recorder/state-compiler';

describe('GitHub Integration Test', () => {
  it('should record clicks on GitHub-like page with dynamic classes', async () => {
    await withTestFixture(async (fixture) => {
      const recordedActions: any[] = [];

      await fixture.getPage().exposeFunction('jetskiRecord', (action: any) => {
        recordedActions.push(action);
      });

      await fixture.getPage().addInitScript(`
        window.recordedClicks = [];
        function generateSelector(element) {
          if (element.dataset?.automationId) {
            return '[data-automation-id="' + element.dataset.automationId + '"]';
          }
          if (element.id) {
            return '#' + element.id;
          }
          const tag = element.tagName.toLowerCase();
          const classes = Array.from(element.classList).filter(c => c);
          const classSelector = classes.map(c => '.' + c).join('');
          return tag + classSelector;
        }

        document.addEventListener('click', (e) => {
          const target = e.target.closest('[data-automation-id], [id]') || e.target;
          const selector = generateSelector(target);
          const action = {
            action_type: 'click',
            selector: {
              primary_css: selector
            },
            timestamp: Date.now(),
            url: window.location.href
          };
          window.recordedClicks.push(action);
          if (window.jetskiRecord) {
            window.jetskiRecord(action);
          }
        }, true);
      `);

      const fixturePath = getFixturePath('github-like.html');
      await fixture.navigateTo(`file://${fixturePath}`);

      await fixture.click('#fork-button');
      await new Promise(resolve => setTimeout(resolve, 100));
      await fixture.click('#star-button');
      await new Promise(resolve => setTimeout(resolve, 100));
      await fixture.click('[data-automation-id="file-readme"]');
      await new Promise(resolve => setTimeout(resolve, 100));

      const clicks = await fixture.getPage().evaluate(() => (globalThis as any).recordedClicks || []);
      expect(clicks.length).toBeGreaterThanOrEqual(3);
      expect(clicks[0].action_type).toBe('click');
      expect(clicks[0].selector?.primary_css).toContain('fork-button');

      expect(recordedActions.length).toBeGreaterThanOrEqual(3);
      expect(recordedActions[0].action_type).toBe('click');
      expect(recordedActions[0].selector?.primary_css).toContain('fork-button');

      await fixture.takeScreenshot('github-click-test.png');
    });
  });

  it('should compile recorded events to valid Nav-Chart', async () => {
    const compiler = new StateCompiler();
    const sampleEvents = [
      {
        action_type: 'click',
        selector: {
          primary_css: '[data-automation-id="file-readme"]',
          secondary_xpath: '//*[@data-automation-id="file-readme"]',
          fallback_js: "document.querySelector('[data-automation-id=\"file-readme\"]')"
        },
        timestamp: Date.now(),
        url: 'https://github.com/test/repo'
      },
      {
        action_type: 'click',
        selector: {
          primary_css: '#fork-button',
          secondary_xpath: '//*[@id="fork-button"]',
          fallback_js: "document.getElementById('fork-button')"
        },
        timestamp: Date.now(),
        url: 'https://github.com/test/repo'
      }
    ];

    const navChart = compiler.compileEvents(sampleEvents, {
      sessionId: 'test-github-session',
      addPostActionWaits: true
    });

    expect(navChart.version).toBe(1);
    expect(navChart.target_domain).toBe('https://github.com');
    expect(navChart.metadata.generated_by).toBe('@armorclaw/jetski-chartmaker');
    expect(navChart.metadata.total_actions).toBe(2);
    expect(navChart.action_map).toBeDefined();
    expect(Object.keys(navChart.action_map)).toHaveLength(2);
    expect(navChart.action_map['action_1']).toBeDefined();
    expect(navChart.action_map['action_1'].action_type).toBe('click');
    expect(navChart.action_map['action_1'].selector?.primary_css).toBe('[data-automation-id="file-readme"]');
    expect(navChart.action_map['action_1'].post_action_wait).toBeDefined();
    expect(navChart.action_map['action_1'].post_action_wait?.type).toBe('waitForVisible');

    const fs = await import('fs');
    const outputPath = getEvidencePath('github-nav-chart.json');
    await StateCompiler.saveToFile(navChart, outputPath);
    expect(fs.existsSync(outputPath)).toBe(true);

    const savedChart = JSON.parse(fs.readFileSync(outputPath, 'utf-8'));
    expect(savedChart.version).toBe(1);
    expect(savedChart.action_map['action_1'].action_type).toBe('click');
  });

  it('should demonstrate selector resilience with dynamic classes', async () => {
    await withTestFixture(async (fixture) => {
      const selectors: string[] = [];

      await fixture.getPage().exposeFunction('jetskiRecord', (action: any) => {
        selectors.push(action.selector?.primary_css);
      });

      await fixture.getPage().addInitScript(`
        window.recordedSelectors = [];
        function generateSelector(element) {
          if (element.dataset?.automationId) {
            return '[data-automation-id="' + element.dataset.automationId + '"]';
          }
          if (element.id) {
            return '#' + element.id;
          }
          const tag = element.tagName.toLowerCase();
          const classes = Array.from(element.classList).filter(c => c);
          const classSelector = classes.map(c => '.' + c).join('');
          return tag + classSelector;
        }

        document.addEventListener('click', (e) => {
          const selector = generateSelector(e.target);
          window.recordedSelectors.push(selector);
          if (window.jetskiRecord) {
            window.jetskiRecord({
              action_type: 'click',
              selector: {
                primary_css: selector
              },
              timestamp: Date.now(),
              url: window.location.href
            });
          }
        }, true);
      `);

      const fixturePath = getFixturePath('github-like.html');
      await fixture.navigateTo(`file://${fixturePath}`);

      const tabElement = await fixture.getPage().locator('.tab-item').first();
      await tabElement.click();
      await new Promise(resolve => setTimeout(resolve, 100));

      const recorded = await fixture.getPage().evaluate(() => (globalThis as any).recordedSelectors || []);
      expect(recorded.length).toBeGreaterThan(0);
      expect(recorded[0]).toMatch(/\.tab-item/);
    });
  });

  it('should handle navigation-like actions', async () => {
    const compiler = new StateCompiler();
    const navEvents = [
      {
        action_type: 'navigate',
        url: 'https://github.com/test/repo',
        timestamp: Date.now()
      },
      {
        action_type: 'click',
        selector: {
          primary_css: '[data-automation-id="file-readme"]'
        },
        timestamp: Date.now(),
        url: 'https://github.com/test/repo'
      }
    ];

    const navChart = compiler.compileEvents(navEvents, {
      sessionId: 'test-nav-session'
    });

    expect(navChart.action_map['action_1'].action_type).toBe('navigate');
    expect(navChart.action_map['action_1'].url).toBe('https://github.com/test/repo');
    expect(navChart.action_map['action_1'].post_action_wait).toBeDefined();
    expect(navChart.action_map['action_1'].post_action_wait?.type).toBe('waitForSelector');

    expect(navChart.action_map['action_2'].action_type).toBe('click');
    expect(navChart.action_map['action_2'].selector?.primary_css).toBe('[data-automation-id="file-readme"]');
  });
});
