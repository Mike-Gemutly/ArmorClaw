package browser

type ActionType string

const (
	ActionClick    ActionType = "click"
	ActionInput    ActionType = "input"
	ActionNavigate ActionType = "navigate"
	ActionWait     ActionType = "wait"
	ActionAssert   ActionType = "assert"
)

type NavChart struct {
	Version      int                    `json:"version"`
	TargetDomain string                 `json:"target_domain"`
	Metadata     ChartMetadata          `json:"metadata"`
	ActionMap    map[string]ChartAction `json:"action_map"`
}

type ChartMetadata struct {
	GeneratedBy string `json:"generated_by"`
	Timestamp   string `json:"timestamp"`
	SessionID   string `json:"session_id,omitempty"`
}

type ChartAction struct {
	ActionType     ActionType       `json:"action_type"`
	Selector       *ChartSelector   `json:"selector,omitempty"`
	Value          string           `json:"value,omitempty"`
	URL            string           `json:"url,omitempty"`
	FrameRouting   *FrameRouting    `json:"frame_routing,omitempty"`
	PostActionWait *WaitCondition   `json:"post_action_wait,omitempty"`
	Assertion      *Assertion       `json:"assertion,omitempty"`
}

type ChartSelector struct {
	PrimaryCSS     string `json:"primary_css"`
	SecondaryXPath string `json:"secondary_xpath,omitempty"`
	FallbackJS     string `json:"fallback_js,omitempty"`
}

type FrameRouting struct {
	Selector string `json:"selector,omitempty"`
	Name     string `json:"name,omitempty"`
	Origin   string `json:"origin,omitempty"`
}

type WaitCondition struct {
	Type     string          `json:"type"`
	Selector *ChartSelector  `json:"selector,omitempty"`
	Timeout  int             `json:"timeout,omitempty"`
}

type Assertion struct {
	Type     string      `json:"type"`
	Expected interface{} `json:"expected"`
}
