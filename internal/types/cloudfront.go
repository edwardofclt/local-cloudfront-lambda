package types

type CloudfrontEventInput struct {
	CfRequest        *CfRequest
	CfResponse       *CfResponse
	FinalResponse    *CfResponse
	CallbackResponse CallbackResponse
}

type CloudfrontEvent interface {
	Execute(CloudfrontEventInput) error
}
