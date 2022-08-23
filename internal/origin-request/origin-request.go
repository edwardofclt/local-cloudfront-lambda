package originrequest

import (
	"github.com/edwardofclt/cloudfront-emulator/internal/types"
	"github.com/pkg/errors"
)

type OriginRequestEvent struct {
}

func New() types.CloudfrontEvent {
	return &OriginRequestEvent{}
}

func (e *OriginRequestEvent) Execute(config types.CloudfrontEventInput) error {
	err := validateRequest(types.OriginRequest, *config.CfRequest, config.CallbackResponse)
	if err != nil {
		return err
	}

	config.CfRequest.BaseConfig = types.MergeBaseConfigs(config.CfRequest.BaseConfig, config.CallbackResponse.BaseConfig)
	types.MergeHeaders(config.CfRequest.Headers, config.CallbackResponse.Headers)

	return nil
}

func validateRequest(eventType types.EventType, request types.CfRequest, response types.CallbackResponse) error {
	if response.Headers != nil {
		if err := types.CheckHeaders(eventType, *request.Headers, *response.Headers); err != nil {
			return err
		}

		for _, header := range *response.Headers {
			if err := types.CheckReadOnlyHeader(types.OriginRequestReadOnlyHeaders, header, *request.Headers); err != nil {
				return errors.Wrap(err, "read only headeres were modified")
			}
		}
	}

	return nil
}
