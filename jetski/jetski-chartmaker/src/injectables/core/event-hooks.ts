export const EVENT_HOOKS_SCRIPT = `
(function() {
  'use strict';

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

  function generate3TierSelector(element) {
    let selector = {
      primary_css: null,
      secondary_xpath: null,
      fallback_js: null
    };
    
    if (element.dataset?.automationId) {
      selector.primary_css = \`[data-automation-id="\${element.dataset.automationId}"]\`;
      selector.secondary_xpath = \`//*[@data-automation-id='\${element.dataset.automationId}']\`;
      selector.fallback_js = \`document.querySelector('[data-automation-id="\${element.dataset.automationId}"]')\`;
    } else if (element.id) {
      selector.primary_css = \`#\${CSS.escape(element.id)}\`;
      selector.secondary_xpath = \`//*[@id='\${element.id}']\`;
      selector.fallback_js = \`document.getElementById('\${element.id}')\`;
    } else {
      const tag = element.tagName.toLowerCase();
      const classes = element.className.split(' ').filter(c => c);
      const classSelector = classes.map(c => \`.\${CSS.escape(c)}\`).join('');
      
      selector.primary_css = \`\${tag}\${classSelector}\`;
      selector.secondary_xpath = \`//\${tag}[@class='\${element.className}']\`;
      selector.fallback_js = \`document.querySelector('\${tag}\${classes[0] ? '.' + CSS.escape(classes[0]) : ''}')\`;
    }
    
    return selector;
  }

  const handleInteraction = throttle((event) => {
    const target = pierceShadowDOM(event.target, event.clientX, event.clientY);
    const selector = generate3TierSelector(target);
    
    const action = {
      action_type: event.type === 'click' ? 'click' : 'input',
      selector,
      timestamp: Date.now(),
      url: window.location.href
    };
    
    if (window.jetskiRecord) {
      window.jetskiRecord(action);
    }
    
    if (window.jetskiHelm) {
      window.jetskiHelm.updateStatus(\`Recorded: \${selector.primary_css}\`);
    }
  }, 100);

  document.addEventListener('click', handleInteraction, true);
  document.addEventListener('input', handleInteraction, true);
  document.addEventListener('change', handleInteraction, true);

  console.log('🧭 Jetski Chartmaker: Event hooks active (100ms throttle)');
})();
`;
