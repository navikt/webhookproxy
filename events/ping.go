package events

// https://developer.github.com/webhooks/#ping-event
type PingEvent struct {
	Zen    string                 `json:"zen"`
	HookId int                    `json:"hook_id"`
	Hook   map[string]interface{} `json:"hook"`
}
