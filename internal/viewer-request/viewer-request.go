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
	response, err := types.ParseRequestBody(config.CallbackResponse)
	if err != nil {
		return err
	}

	err = validateRequest(types.ViewerRequest, *config.CfRequest, response)
	if err != nil {
		return err
	}

	types.MergeRequestBody(config.CfRequest, response)

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
