package originresponse

import (
	"github.com/edwardofclt/cloudfront-emulator/internal/types"
)

type OriginResponseEvent struct {
}

func New() types.CloudfrontEvent {
	return &OriginResponseEvent{}
}

func (e *OriginResponseEvent) Execute(config types.CloudfrontEventInput) error {
	err := validateResponse(types.OriginRequest, *config.CfRequest, config.CallbackResponse)
	if err != nil {
		return err
	}

	config.CfResponse.BaseConfig = types.MergeBaseConfigs(config.CfResponse.BaseConfig, config.CallbackResponse.BaseConfig)
	types.MergeHeaders(config.CfResponse.Headers, config.FinalResponse.Headers)
	types.MergeHeaders(config.CfResponse.Headers, config.CallbackResponse.Headers)
	if config.CallbackResponse.Headers != nil {
		// TODO: make sure this makes sense and headers are showing up properly
		types.MergeHeaders(config.FinalResponse.Headers, config.CfResponse.Headers)
	}
	return nil
}

func validateResponse(eventType types.EventType, request types.CfRequest, response types.CallbackResponse) error {
	if response.Headers != nil {
		if err := types.CheckHeaders(eventType, *request.Headers, *response.Headers); err != nil {
			return err
		}

		// TODO: Validate that the headers that are being modified are allowed
		// for _, header := range *response.Headers {
		// 	if err := types.CheckReadOnlyHeader(types.OriginRequestReadOnlyHeaders, header, *request.Headers); err != nil {
		// 		return errors.Wrap(err, "read only headeres were modified")
		// 	}
		// }
	}

	return nil
}
