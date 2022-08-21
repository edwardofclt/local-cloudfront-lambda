package types

type CloudfrontEventInput struct {
	CfRequest        *CfRequest
	CfResponse       *CfResponse
	CallbackResponse []byte
}

type CloudfrontEvent interface {
	Execute(CloudfrontEventInput) error
}
