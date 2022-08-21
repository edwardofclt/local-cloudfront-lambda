package viewerrequest

import (
	"fmt"

	"github.com/edwardofclt/cloudfront-emulator/internal/types"
)

type ViewerRequestEvent struct {
}

func New() types.CloudfrontEvent {
	return &ViewerRequestEvent{}
}

func (e *ViewerRequestEvent) Execute(config types.CloudfrontEventConfig) error {
	respData, err := types.ParseRequestBody(config.CallbackResponse)
	if err != nil {
		return err
	}

	fmt.Println(respData.Status)
	fmt.Println("hi")

	types.MergeRequestBody(config.CfRequest, respData)

	err = validateRequest(types.ViewerRequest, *config.CfRequest, respData)
	if err != nil {
		return err
	}

	return nil
}

func validateRequest(eventType types.EventType, request types.CfRequest, respData types.CfRequest) error {
	if respData.Headers != nil {
		if err := types.CheckHeaders(eventType, *request.Headers, *respData.Headers); err != nil {
			return err
		}
	}

	return nil
}
