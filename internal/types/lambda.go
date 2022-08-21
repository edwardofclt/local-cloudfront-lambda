package types

type CallbackResponse struct {
	// TODO: bodyEncoding string `json:"bodyEncoding"`
	Body              *string        `json:"body"`
	Headers           *CfHeaderArray `json:"headers"`
	Status            *string        `json:"status"`
	StatusDescription *string        `json:"statusDescription"`
}
