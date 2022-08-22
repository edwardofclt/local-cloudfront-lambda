package types

type CallbackResponse struct {
	// TODO: bodyEncoding string `json:"bodyEncoding"`
	// Body              *string        `json:"body,omitempty"`
	// Headers           *CfHeaderArray `json:"headers,omitempty"`
	// Status            *string        `json:"status,omitempty"`
	// StatusDescription *string        `json:"statusDescription,omitempty"`
	BaseConfig
	CfResponse
}
