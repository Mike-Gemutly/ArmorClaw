(function() {
  'use strict';

  let isDragging = false;
  let startX, startY, initialX, initialY;
  let currentMode = 'click';
  let actionCount = 0;

  let helm, header, statusDisplay, countDisplay, closeButton, saveButton, clearButton, modeButtons;

  // Initialize HUD elements when container is ready
  function initializeHUD() {
    const container = document.getElementById('jetski-helm-container');
    if (!container || !container.shadowRoot) {
      // Container not ready yet, retry after a short delay
      setTimeout(initializeHUD, 50);
      return;
    }

    const shadowRoot = container.shadowRoot;
    helm = shadowRoot.getElementById('jetski-helm');
    header = shadowRoot.getElementById('helm-header');
    statusDisplay = shadowRoot.getElementById('helm-status');
    countDisplay = shadowRoot.getElementById('helm-recorded-count');
    closeButton = shadowRoot.getElementById('helm-close');
    saveButton = shadowRoot.getElementById('helm-save');
    clearButton = shadowRoot.getElementById('helm-clear');
    modeButtons = shadowRoot.querySelectorAll('.mode-btn');

    // Start functionality after elements are initialized
    makeDraggable();
    setupEventListeners();
    updateStatus('Ready to chart...');

    console.log('🧭 Jetski Chartmaker: The Helm is ready');
  }

  function makeDraggable() {
    header.addEventListener('mousedown', (e) => {
      if (e.target === closeButton) return;
      isDragging = true;
      startX = e.clientX;
      startY = e.clientY;
      const rect = helm.getBoundingClientRect();
      initialX = rect.left;
      initialY = rect.top;
      helm.style.position = 'fixed';
      helm.style.right = 'auto';
    });

    document.addEventListener('mousemove', (e) => {
      if (!isDragging) return;
      const deltaX = e.clientX - startX;
      const deltaY = e.clientY - startY;
      helm.style.left = (initialX + deltaX) + 'px';
      helm.style.top = (initialY + deltaY) + 'px';
    });

    document.addEventListener('mouseup', () => {
      isDragging = false;
    });
  }

  function updateStatus(message) {
    statusDisplay.textContent = message;
  }

  function updateActionCount(count) {
    actionCount = count;
    countDisplay.textContent = `Actions: ${actionCount}`;
  }

  function closeHUD() {
    helm.style.display = 'none';
  }

  async function saveNavChart() {
    updateStatus('Saving Nav-Chart...');
    
    if (window.jetskiSave) {
      try {
        await window.jetskiSave();
        updateStatus('Nav-Chart saved successfully!');
      } catch (error) {
        updateStatus(`Save failed: ${error.message}`);
      }
    } else {
      updateStatus('Save function not available');
    }

    setTimeout(() => {
      updateStatus('Ready to chart...');
    }, 2000);
  }

  async function clearActions() {
    updateStatus('Clearing actions...');
    
    if (window.jetskiClear) {
      try {
        await window.jetskiClear();
        actionCount = 0;
        updateActionCount(0);
        updateStatus('Actions cleared');
      } catch (error) {
        updateStatus(`Clear failed: ${error.message}`);
      }
    } else {
      updateStatus('Clear function not available');
    }

    setTimeout(() => {
      updateStatus('Ready to chart...');
    }, 2000);
  }

  function setMode(mode) {
    currentMode = mode;
    modeButtons.forEach(btn => {
      btn.classList.toggle('active', btn.dataset.mode === mode);
    });
    
    const modeNames = {
      click: 'Recording Click',
      input: 'Recording Input',
      assert: 'Recording Assertion'
    };
    updateStatus(modeNames[mode]);
  }

  function detectFrameRouting() {
    if (window.self !== window.top) {
      const iframeSelector = findIframeSelector();
      return {
        selector: iframeSelector,
        name: window.name || undefined,
        origin: window.location.origin
      };
    }
    return null;
  }

  function findIframeSelector() {
    try {
      let selector = '';
      let currentWindow = window;

      while (currentWindow !== window.top) {
        const parentWindow = currentWindow.parent;

        if (!parentWindow || !parentWindow.document) {
          break;
        }

        const frames = Array.from(parentWindow.document.querySelectorAll('iframe, frame'));
        let frameElement = null;

        for (const frame of frames) {
          if (frame.contentWindow === currentWindow) {
            frameElement = frame;
            break;
          }
        }

        if (frameElement) {
          const frameSelector = frameElement.id
            ? `#${CSS.escape(frameElement.id)}`
            : frameElement.name
            ? `[name="${CSS.escape(frameElement.name)}"]`
            : `${frameElement.tagName.toLowerCase()}`;

          selector = selector ? `${frameSelector} >> ${selector}` : frameSelector;
        }

        currentWindow = parentWindow;
      }

      return selector || null;
    } catch (e) {
      console.warn('🧭 Jetski: Error detecting iframe selector:', e);
      return null;
    }
  }

  function recordAction(action) {
    actionCount++;
    updateActionCount(actionCount);

    const modeNames = {
      click: 'Click',
      input: 'Input',
      assert: 'Assertion'
    };

    if (action.selector) {
      updateStatus(`Recorded ${modeNames[action.action_type]}: ${action.selector.primary_css}`);
    } else {
      updateStatus(`Recorded ${modeNames[action.action_type]}`);
    }

    action.frame_routing = detectFrameRouting();

    if (window.jetskiRecord) {
      window.jetskiRecord(action);
    } else {
      window.parent.postMessage({
        type: 'JETSKI_RECORD',
        action: action
      }, '*');
    }

    setTimeout(() => {
      updateStatus(`Mode: ${currentMode.charAt(0).toUpperCase() + currentMode.slice(1)}`);
    }, 1500);
  }

  function buildShadowPath(element) {
    const path = [];
    let current = element;
    let maxDepth = 10;
    let depth = 0;

    while (current && depth < maxDepth) {
      try {
        const root = current.getRootNode();

        if (root instanceof ShadowRoot) {
          // We hit a Shadow DOM boundary
          path.unshift(getSelectorForElement(current));
          current = root.host;
          depth++;
        } else if (root === document) {
          // Reached the main document
          path.unshift(getSelectorForElement(current));
          break;
        } else {
          // Move up the tree in regular DOM
          current = current.parentElement;
          depth++;
        }
      } catch (e) {
        // Handle closed Shadow DOM or other errors gracefully
        console.warn('🧭 Jetski: Error while building Shadow DOM path:', e);
        path.unshift(getSelectorForElement(current));
        break;
      }
    }

    return path;
  }

  function getSelectorForElement(element) {
    // Tier 1: data-automation-id (best)
    if (element.dataset?.automationId) {
      return `[data-automation-id="${element.dataset.automationId}"]`;
    }

    // Tier 2: data-testid (good)
    if (element.dataset?.testid) {
      return `[data-testid="${element.dataset.testid}"]`;
    }

    // Tier 3: ID (acceptable)
    if (element.id) {
      return `#${CSS.escape(element.id)}`;
    }

    // Tier 4: tag + classes (fallback)
    const tag = element.tagName.toLowerCase();
    const classes = Array.from(element.classList)
      .filter(c => {
        // Filter out Tailwind hash classes (css-*, scss-*, random hashes)
        return !c.match(/^(css-|scss-|_[a-f0-9]{5,}|[a-f0-9]{6,})$/);
      });

    if (classes.length > 0) {
      return `${tag}.${classes.map(c => CSS.escape(c)).join('.')}`;
    }

    return tag;
  }

  function generateSelector(element) {
    try {
      const shadowPath = buildShadowPath(element);

      if (shadowPath.length === 0) {
        console.warn('🧭 Jetski: Empty Shadow DOM path generated');
      }

      const selector = {
        primary_css: shadowPath.join(' >> '),
        secondary_xpath: null,
        fallback_js: null
      };

      // Generate secondary XPath
      const xpathPath = shadowPath.map((segment, index) => {
        if (segment.startsWith('[data-automation-id=')) {
          const id = segment.match(/data-automation-id="([^"]+)"/)?.[1];
          return `//*[@data-automation-id='${id}']`;
        } else if (segment.startsWith('[data-testid=')) {
          const id = segment.match(/data-testid="([^"]+)"/)?.[1];
          return `//*[@data-testid='${id}']`;
        } else if (segment.startsWith('#')) {
          const id = segment.slice(1);
          return `//*[@id='${id}']`;
        } else {
          const tag = segment.split('.')[0];
          return `//${tag}`;
        }
      });
      selector.secondary_xpath = xpathPath.join('/');

      // Generate fallback JS
      const jsPath = shadowPath.map((segment) => {
        if (segment.startsWith('[data-automation-id=')) {
          const id = segment.match(/data-automation-id="([^"]+)"/)?.[1];
          return `querySelector('[data-automation-id="${id}"]')`;
        } else if (segment.startsWith('[data-testid=')) {
          const id = segment.match(/data-testid="([^"]+)"/)?.[1];
          return `querySelector('[data-testid="${id}"]')`;
        } else if (segment.startsWith('#')) {
          const id = segment.slice(1);
          return `getElementById('${id}')`;
        } else {
          return `querySelector('${segment}')`;
        }
      });

      if (jsPath.length === 1) {
        selector.fallback_js = `document.${jsPath[0]}`;
      } else {
        let jsExpression = `document`;
        jsPath.forEach(segment => {
          jsExpression += `.${segment}`;
        });
        selector.fallback_js = jsExpression;
      }

      return selector;
    } catch (e) {
      console.error('🧭 Jetski: Error generating selector:', e);
      return {
        primary_css: element.id ? `#${CSS.escape(element.id)}` : element.tagName.toLowerCase(),
        secondary_xpath: null,
        fallback_js: null
      };
    }
  }

  function handleInteraction(e) {
    const target = e.target;
    const selector = generateSelector(target);

    const action = {
      action_type: currentMode,
      selector,
      timestamp: Date.now(),
      url: window.location.href
    };

    if (currentMode === 'input') {
      action.value = target.value || '';
    }

    recordAction(action);
  }

  function setupEventListeners() {
    modeButtons.forEach(btn => {
      btn.addEventListener('click', () => setMode(btn.dataset.mode));
    });

    closeButton.addEventListener('click', closeHUD);
    saveButton.addEventListener('click', saveNavChart);
    clearButton.addEventListener('click', clearActions);

    document.addEventListener('click', (e) => {
      if (helm.contains(e.target)) return;
      if (currentMode === 'click' || currentMode === 'assert') {
        handleInteraction(e);
      }
    }, true);

    document.addEventListener('change', (e) => {
      if (helm.contains(e.target)) return;
      if (currentMode === 'input') {
        handleInteraction(e);
      }
    }, true);
  }

  function throttle(fn, delay) {
    let lastCall = 0;
    return function(...args) {
      const now = Date.now();
      if (now - lastCall >= delay) {
        lastCall = now;
        fn.apply(this, args);
      }
    };
  }

  window.jetskiHelm = {
    updateStatus,
    updateActionCount,
    setMode,
    closeHUD
  };

  initializeHUD();
})();
