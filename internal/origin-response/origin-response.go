package originresponse

import "github.com/edwardofclt/cloudfront-emulator/internal/types"

type OriginResponseEvent struct {
}

func New() types.CloudfrontEvent {
	return &OriginResponseEvent{}
}

func (e *OriginResponseEvent) Execute(config types.CloudfrontEventInput) error {
	response, err := types.ParseResponseBody(config.CallbackResponse)
	if err != nil {
		return err
	}

	err = validateResponse(types.OriginRequest, *config.CfRequest, response)
	if err != nil {
		return err
	}

	types.MergeResponseBody(config.CfResponse, response)
	return nil
}

func validateResponse(eventType types.EventType, request types.CfRequest, response types.CfResponse) error {
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
