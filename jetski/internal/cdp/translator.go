package cdp

import (
	"encoding/json"
	"fmt"
)

type Translator struct{}

func NewTranslator() *Translator {
	return &Translator{}
}

func (t *Translator) Translate(msg *CDPMessage) (*CDPMessage, error) {
	if msg.Method == "Input.dispatchMouseEvent" {
		return t.translateMouseEvent(msg)
	}

	return msg, nil
}

func (t *Translator) translateMouseEvent(msg *CDPMessage) (*CDPMessage, error) {
	var params map[string]any
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mouse event params: %w", err)
	}

	x, _ := params["x"].(float64)
	y, _ := params["y"].(float64)

	expression := t.translateMouseClick(int(x), int(y))

	translatedParams := map[string]any{
		"expression":                  expression,
		"awaitPromise":                true,
		"returnByValue":               true,
		"generatePreview":             false,
		"userGesture":                 true,
		"replMode":                    false,
		"allowUnsafeEvalBlockedByCSP": false,
	}

	paramsJSON, _ := json.Marshal(translatedParams)

	return &CDPMessage{
		ID:     msg.ID,
		Method: "Runtime.evaluate",
		Params: paramsJSON,
	}, nil
}

func (t *Translator) translateMouseClick(x, y int) string {
	return fmt.Sprintf(`(() => {
  const elem = document.elementFromPoint(%d, %d);
  if (!elem) return null;

  let target = elem;

  if (elem.shadowRoot) {
    const shadowElem = elem.shadowRoot.elementFromPoint(%d, %d);
    target = shadowElem || elem;
  }

  const selector = generateSelector(target);
  target.click();
  return {selector: selector, element: target.tagName};
})()

function generateSelector(elem) {
  if (!elem) return null;

  const automationId = elem.getAttribute('data-automation-id');
  if (automationId) {
    return '[data-automation-id="' + automationId + '"]';
  }

  if (elem.id) {
    return '#' + elem.id;
  }

  let selector = elem.tagName.toLowerCase();
  if (elem.className && typeof elem.className === 'string') {
    const classes = elem.className.trim().split(/\s+/).filter(c => c);
    if (classes.length > 0) {
      selector += '.' + classes.join('.');
    }
  }

  return selector;
}`, x, y, x, y)
}
