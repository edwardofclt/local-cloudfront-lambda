package viewerrequest

import (
	"github.com/edwardofclt/cloudfront-emulator/internal/types"
)

type ViewerRequestEvent struct {
}

func New() types.CloudfrontEvent {
	return &ViewerRequestEvent{}
}

func (e *ViewerRequestEvent) Execute(config types.CloudfrontEventInput) error {
	err := validateRequest(types.ViewerRequest, *config.CfRequest, *config.CfRequest)
	if err != nil {
		return err
	}

	types.MergeHeaders(config.CfRequest.Headers, config.CallbackResponse.Headers)
	return nil
}

func validateRequest(eventType types.EventType, request types.CfRequest, response types.CfRequest) error {
	if response.Headers != nil {
		if err := types.CheckHeaders(eventType, *request.Headers, *response.Headers); err != nil {
			return err
		}
	}

	return nil
}
