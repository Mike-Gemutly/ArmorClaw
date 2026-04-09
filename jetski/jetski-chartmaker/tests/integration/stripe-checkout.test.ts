import { withTestFixture, getFixturePath, getEvidencePath } from './test-utils';
import { StateCompiler } from '../../src/core/recorder/state-compiler';

describe('Stripe Checkout Test', () => {
  it('should detect cross-origin iframe', async () => {
    await withTestFixture(async (fixture) => {
      const fixturePath = getFixturePath('stripe-like.html');
      await fixture.navigateTo(`file://${fixturePath}`);

      const iframe = await fixture.getPage().locator('#stripe-card-element').elementHandle();
      expect(iframe).not.toBeNull();

      if (iframe) {
        const frame = await iframe.contentFrame();
        if (frame) {
          const content = await frame.content();
          expect(content).toContain('card-number');
          expect(content).toContain('card-number-input');
        }
      }

      await fixture.takeScreenshot('stripe-iframe-detection.png');
    });
  });

  it('should record clicks in main page and iframe', async () => {
    await withTestFixture(async (fixture) => {
      const recordedActions: any[] = [];

      await fixture.getPage().exposeFunction('jetskiRecord', (action: any) => {
        recordedActions.push(action);
      });

      await fixture.getPage().addInitScript(`
        window.recordedActions = [];
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
          const action = {
            action_type: 'click',
            selector: {
              primary_css: selector
            },
            timestamp: Date.now(),
            url: window.location.href
          };
          window.recordedActions.push(action);
          if (window.jetskiRecord) {
            window.jetskiRecord(action);
          }
        }, true);
      `);

      const fixturePath = getFixturePath('stripe-like.html');
      await fixture.navigateTo(`file://${fixturePath}`);

      await fixture.click('#email');
      await new Promise(resolve => setTimeout(resolve, 100));
      await fixture.fill('#email', 'test@example.com');
      await new Promise(resolve => setTimeout(resolve, 100));

      const mainPageActions = await fixture.getPage().evaluate(() => (globalThis as any).recordedActions || []);
      expect(mainPageActions.length).toBeGreaterThan(0);

      expect(recordedActions.length).toBeGreaterThan(0);
      expect(recordedActions[0].action_type).toBe('click');
      expect(recordedActions[0].selector?.primary_css).toContain('email');

      await fixture.takeScreenshot('stripe-main-page-clicks.png');
    });
  });

  it('should compile events with frame routing metadata', async () => {
    const compiler = new StateCompiler();
    const iframeEvents = [
      {
        action_type: 'click',
        selector: {
          primary_css: '#card-number',
          secondary_xpath: '//*[@id="card-number"]',
          fallback_js: "document.getElementById('card-number')"
        },
        frame_routing: {
          selector: '#stripe-card-element',
          name: 'stripe-card-element',
          origin: 'cross-origin'
        },
        timestamp: Date.now(),
        url: 'https://example.com/checkout'
      },
      {
        action_type: 'input',
        selector: {
          primary_css: '#card-number'
        },
        value: '4242424242424242',
        frame_routing: {
          selector: '#stripe-card-element',
          name: 'stripe-card-element',
          origin: 'cross-origin'
        },
        timestamp: Date.now(),
        url: 'https://example.com/checkout'
      }
    ];

    const navChart = compiler.compileEvents(iframeEvents, {
      sessionId: 'test-stripe-session'
    });

    expect(navChart.version).toBe(1);
    expect(navChart.target_domain).toBe('https://example.com');
    expect(navChart.action_map['action_1']).toBeDefined();
    expect(navChart.action_map['action_1'].action_type).toBe('click');
    expect(navChart.action_map['action_1'].selector?.primary_css).toBe('#card-number');
    expect(navChart.action_map['action_1'].frame_routing).toBeDefined();
    expect(navChart.action_map['action_1'].frame_routing?.selector).toBe('#stripe-card-element');
    expect(navChart.action_map['action_1'].frame_routing?.origin).toBe('cross-origin');

    expect(navChart.action_map['action_2']).toBeDefined();
    expect(navChart.action_map['action_2'].action_type).toBe('input');
    expect(navChart.action_map['action_2'].value).toBe('4242424242424242');
    expect(navChart.action_map['action_2'].frame_routing?.origin).toBe('cross-origin');
  });

  it('should validate frame routing metadata', async () => {
    const compiler = new StateCompiler();

    const validFrameRouting = {
      selector: '#stripe-card-element',
      name: 'stripe-card-element',
      origin: 'cross-origin'
    };

    const isValid = compiler.validateFrameRouting({
      action_type: 'click',
      selector: {
        primary_css: '#card-number'
      },
      frame_routing: validFrameRouting,
      timestamp: Date.now(),
      url: 'https://example.com/checkout'
    });

    expect(isValid).toBe(true);

    const invalidFrameRouting = {
      selector: undefined,
      name: undefined,
      origin: undefined
    };

    const isInvalid = compiler.validateFrameRouting({
      action_type: 'click',
      selector: {
        primary_css: '#card-number'
      },
      frame_routing: invalidFrameRouting,
      timestamp: Date.now(),
      url: 'https://example.com/checkout'
    });

    expect(isInvalid).toBe(false);
  });

  it('should save compiled Nav-Chart with iframe actions', async () => {
    const compiler = new StateCompiler();
    const iframeEvents = [
      {
        action_type: 'click',
        selector: {
          primary_css: '#email',
          secondary_xpath: '//*[@id="email"]',
          fallback_js: "document.getElementById('email')"
        },
        timestamp: Date.now(),
        url: 'https://example.com/checkout'
      },
      {
        action_type: 'input',
        selector: {
          primary_css: '#email'
        },
        value: 'test@example.com',
        timestamp: Date.now(),
        url: 'https://example.com/checkout'
      },
      {
        action_type: 'click',
        selector: {
          primary_css: '#pay-button'
        },
        timestamp: Date.now(),
        url: 'https://example.com/checkout'
      }
    ];

    const navChart = compiler.compileEvents(iframeEvents, {
      sessionId: 'test-stripe-checkout',
      addPostActionWaits: true
    });

    expect(navChart.action_map['action_1'].post_action_wait).toBeDefined();
    expect(navChart.action_map['action_2'].post_action_wait).toBeDefined();
    expect(navChart.action_map['action_3'].post_action_wait).toBeDefined();

    const fs = await import('fs');
    const outputPath = getEvidencePath('stripe-nav-chart.json');
    await StateCompiler.saveToFile(navChart, outputPath);
    expect(fs.existsSync(outputPath)).toBe(true);

    const savedChart = JSON.parse(fs.readFileSync(outputPath, 'utf-8'));
    expect(savedChart.version).toBe(1);
    expect(savedChart.metadata.session_id).toBe('test-stripe-checkout');
    expect(savedChart.action_map['action_1'].action_type).toBe('click');
    expect(savedChart.action_map['action_2'].action_type).toBe('input');
    expect(savedChart.action_map['action_3'].action_type).toBe('click');
  });

  it('should not use real Stripe credentials (safety check)', async () => {
    const compiler = new StateCompiler();
    const sampleEvents = [
      {
        action_type: 'click',
        selector: {
          primary_css: '#email'
        },
        timestamp: Date.now(),
        url: 'https://example.com/checkout'
      }
    ];

    const navChart = compiler.compileEvents(sampleEvents, {
      sessionId: 'test-stripe-safety'
    });

    expect(navChart.action_map['action_1'].value).toBeUndefined();
    expect(navChart.action_map['action_1'].selector?.primary_css).not.toMatch(/sk_live_/);
    expect(navChart.action_map['action_1'].selector?.primary_css).not.toMatch(/sk_test_/);
    expect(navChart.action_map['action_1'].selector?.primary_css).not.toMatch(/pk_live_/);
    expect(navChart.action_map['action_1'].selector?.primary_css).not.toMatch(/pk_test_/);
  });
});
