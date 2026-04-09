import { withTestFixture, getFixturePath, getEvidencePath } from './test-utils';
import { StateCompiler } from '../../src/core/recorder/state-compiler';

describe('Web Components Test', () => {
  it('should pierce Shadow DOM and detect nested components', async () => {
    await withTestFixture(async (fixture) => {
      const fixturePath = getFixturePath('web-components.html');
      await fixture.navigateTo(`file://${fixturePath}`);

      const outerComponent = await fixture.getPage().locator('outer-component').elementHandle();
      expect(outerComponent).not.toBeNull();

      await fixture.takeScreenshot('web-components-visible.png');
    });
  });

  it('should click through multiple levels of Shadow DOM', async () => {
    await withTestFixture(async (fixture) => {
      const fixturePath = getFixturePath('web-components.html');
      await fixture.navigateTo(`file://${fixturePath}`);

      const resultBefore = await fixture.getPage().textContent('#result');
      expect(resultBefore).toContain('Click components');

      await fixture.getPage().evaluate(() => {
        const outerComponent = (globalThis as any).document.querySelector('outer-component');
        if (outerComponent && outerComponent.shadowRoot) {
          const middleComponent = outerComponent.shadowRoot.querySelector('middle-component');
          if (middleComponent && middleComponent.shadowRoot) {
            const innerComponent = middleComponent.shadowRoot.querySelector('inner-component');
            if (innerComponent && innerComponent.shadowRoot) {
              const button = innerComponent.shadowRoot.querySelector('#deepest-button');
              if (button) {
                button.click();
              }
            }
          }
        }
      });

      await new Promise(resolve => setTimeout(resolve, 100));

      const resultAfter = await fixture.getPage().textContent('#result');
      expect(resultAfter).toContain('Deepest button clicked');
      expect(resultAfter).toContain('Shadow DOM pierced');

      await fixture.takeScreenshot('web-components-shadow-dom-clicked.png');
    });
  });

  it('should record clicks in Shadow DOM with proper selectors', async () => {
    await withTestFixture(async (fixture) => {
      const recordedActions: any[] = [];

      await fixture.getPage().exposeFunction('jetskiRecord', (action: any) => {
        recordedActions.push(action);
      });

      await fixture.getPage().addInitScript(`
        window.recordedShadowClicks = [];

        function pierceShadowDOM(element, x, y) {
          let current = element;
          let maxDepth = 10;
          let depth = 0;

          while (current.shadowRoot && depth < maxDepth) {
            const shadowElement = current.shadowRoot.elementFromPoint(x, y);
            if (!shadowElement || shadowElement === current) break;
            current = shadowElement;
            depth++;
          }

          return current;
        }

        document.addEventListener('click', (e) => {
          const pierced = pierceShadowDOM(e.target, e.clientX, e.clientY);

          let selector;
          if (pierced.id) {
            selector = '#' + pierced.id;
          } else if (pierced.dataset?.automationId) {
            selector = '[data-automation-id="' + pierced.dataset.automationId + '"]';
          } else {
            const tag = pierced.tagName.toLowerCase();
            const classes = Array.from(pierced.classList).filter(c => c);
            selector = tag + classes.map(c => '.' + c).join('');
          }

          const action = {
            action_type: 'click',
            selector: {
              primary_css: selector
            },
            timestamp: Date.now(),
            url: window.location.href
          };

          window.recordedShadowClicks.push(action);

          if (window.jetskiRecord) {
            window.jetskiRecord(action);
          }
        }, true);
      `);

      const fixturePath = getFixturePath('web-components.html');
      await fixture.navigateTo(`file://${fixturePath}`);

      await fixture.getPage().evaluate(() => {
        const outerComponent = (globalThis as any).document.querySelector('outer-component');
        if (outerComponent && outerComponent.shadowRoot) {
          const middleComponent = outerComponent.shadowRoot.querySelector('middle-component');
          if (middleComponent && middleComponent.shadowRoot) {
            const innerComponent = middleComponent.shadowRoot.querySelector('inner-component');
            if (innerComponent && innerComponent.shadowRoot) {
              const button = innerComponent.shadowRoot.querySelector('#deepest-button');
              if (button) {
                button.click();
              }
            }
          }
        }
      });

      await new Promise(resolve => setTimeout(resolve, 100));

      const shadowClicks = await fixture.getPage().evaluate(() => (globalThis as any).recordedShadowClicks || []);
      expect(shadowClicks.length).toBeGreaterThan(0);

      expect(recordedActions.length).toBeGreaterThan(0);
      expect(recordedActions[0].action_type).toBe('click');

      await fixture.takeScreenshot('web-components-shadow-recording.png');
    });
  });

  it('should compile Shadow DOM actions to valid Nav-Chart', async () => {
    const compiler = new StateCompiler();
    const shadowDomEvents = [
      {
        action_type: 'click',
        selector: {
          primary_css: '#deepest-button',
          secondary_xpath: '//*[@id="deepest-button"]',
          fallback_js: "document.querySelector('#deepest-button')"
        },
        timestamp: Date.now(),
        url: 'file:///web-components.html'
      },
      {
        action_type: 'click',
        selector: {
          primary_css: '[data-automation-id="deepest-button"]',
          secondary_xpath: '//*[@data-automation-id="deepest-button"]',
          fallback_js: "document.querySelector('[data-automation-id=\"deepest-button\"]')"
        },
        timestamp: Date.now(),
        url: 'file:///web-components.html'
      }
    ];

    const navChart = compiler.compileEvents(shadowDomEvents, {
      sessionId: 'test-shadow-dom-session',
      addPostActionWaits: true
    });

    expect(navChart.version).toBe(1);
    expect(navChart.action_map['action_1']).toBeDefined();
    expect(navChart.action_map['action_1'].action_type).toBe('click');
    expect(navChart.action_map['action_1'].selector?.primary_css).toBe('#deepest-button');
    expect(navChart.action_map['action_1'].post_action_wait).toBeDefined();
    expect(navChart.action_map['action_1'].post_action_wait?.type).toBe('waitForVisible');

    const fs = await import('fs');
    const outputPath = getEvidencePath('shadow-dom-nav-chart.json');
    await StateCompiler.saveToFile(navChart, outputPath);
    expect(fs.existsSync(outputPath)).toBe(true);

    const savedChart = JSON.parse(fs.readFileSync(outputPath, 'utf-8'));
    expect(savedChart.version).toBe(1);
    expect(savedChart.action_map['action_1'].selector?.primary_css).toBe('#deepest-button');
  });

  it('should handle Shadow DOM with data-automation-id attributes', async () => {
    await withTestFixture(async (fixture) => {
      const fixturePath = getFixturePath('web-components.html');
      await fixture.navigateTo(`file://${fixturePath}`);

      const hasOuterComponent = await fixture.getPage().evaluate(() => {
        const outer = (globalThis as any).document.querySelector('outer-component');
        return outer && outer.hasAttribute('data-automation-id');
      });

      expect(hasOuterComponent).toBe(true);

      const automationIds = await fixture.getPage().evaluate(() => {
        const elements = (globalThis as any).document.querySelectorAll('[data-automation-id]');
        const ids = Array.from(elements).map((el: any) => el.getAttribute('data-automation-id'));

        elements.forEach((element: any) => {
          if (element.shadowRoot) {
            const shadowElements = element.shadowRoot.querySelectorAll('[data-automation-id]');
            shadowElements.forEach((shadowEl: any) => {
              ids.push(shadowEl.getAttribute('data-automation-id'));

              if (shadowEl.shadowRoot) {
                const deeperElements = shadowEl.shadowRoot.querySelectorAll('[data-automation-id]');
                deeperElements.forEach((deeperEl: any) => {
                  ids.push(deeperEl.getAttribute('data-automation-id'));

                  if (deeperEl.shadowRoot) {
                    const deepestElements = deeperEl.shadowRoot.querySelectorAll('[data-automation-id]');
                    deepestElements.forEach((deepestEl: any) => {
                      ids.push(deepestEl.getAttribute('data-automation-id'));
                    });
                  }
                });
              }
            });
          }
        });

        return ids;
      });

      expect(automationIds).toContain('outer-component');
      expect(automationIds).toContain('middle-component');
      expect(automationIds).toContain('inner-component');
      expect(automationIds).toContain('deepest-button');

      await fixture.takeScreenshot('web-components-automation-ids.png');
    });
  });

  it('should pierce Shadow DOM at multiple depth levels', async () => {
    await withTestFixture(async (fixture) => {
      const fixturePath = getFixturePath('web-components.html');
      await fixture.navigateTo(`file://${fixturePath}`);

      const shadowDOMInfo = await fixture.getPage().evaluate(() => {
        let depth = 0;
        const path: string[] = [];

        const outer = (globalThis as any).document.querySelector('outer-component');
        if (outer && outer.shadowRoot) {
          depth++;
          path.push('outer-component');

          const middle = outer.shadowRoot.querySelector('middle-component');
          if (middle && middle.shadowRoot) {
            depth++;
            path.push('middle-component');

            const inner = middle.shadowRoot.querySelector('inner-component');
            if (inner && inner.shadowRoot) {
              depth++;
              path.push('inner-component');

              const button = inner.shadowRoot.querySelector('#deepest-button');
              if (button) {
                path.push('deepest-button');
              }
            }
          }
        }

        return { depth, path };
      });

      expect(shadowDOMInfo.depth).toBe(3);
      expect(shadowDOMInfo.path).toContain('outer-component');
      expect(shadowDOMInfo.path).toContain('middle-component');
      expect(shadowDOMInfo.path).toContain('inner-component');
      expect(shadowDOMInfo.path).toContain('deepest-button');

      await fixture.takeScreenshot('web-components-depth-levels.png');
    });
  });
});
