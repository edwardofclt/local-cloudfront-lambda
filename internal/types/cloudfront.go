package types

type CloudfrontEventConfig struct {
	CfRequest        *CfRequest
	CfResponse       *CfResponse
	CallbackResponse []byte
}

type CloudfrontEvent interface {
	Execute(CloudfrontEventConfig) error
}
