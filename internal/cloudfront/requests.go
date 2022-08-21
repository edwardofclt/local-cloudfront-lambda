package cloudfront

import (
	"net/http"

	"github.com/edwardofclt/cloudfront-emulator/internal/types"
	"github.com/pkg/errors"
)

func writeRequestHeaders(w http.ResponseWriter, respData CfRequest) {
	if respData.Headers == nil {
		return
	}

	for _, header := range *respData.Headers {
		for _, val := range header {
			w.Header().Add(val.Key, val.Value)
		}
	}
}

func writeRequestHeadersV2(w http.ResponseWriter, respData types.CfRequest) {
	if respData.Headers == nil {
		return
	}

	for _, header := range *respData.Headers {
		for _, val := range header {
			w.Header().Add(val.Key, val.Value)
		}
	}
}

func validateRequest(eventType EventType, reqData RequestPayload, respData CfRequest) error {
	if respData.Headers != nil {
		if err := checkHeaders(eventType, *reqData.Records[0].Cf.Request.Headers, *respData.Headers); err != nil {
			return errors.Wrap(err, "headers could not be validated")
		}
	}

	return nil
}

func validateRequestV2(eventType types.EventType, reqData types.RequestPayload, respData types.CfRequest) error {
	if respData.Headers != nil {
		if err := checkHeadersV2(eventType, *reqData.Records[0].Cf.Request.Headers, *respData.Headers); err != nil {
			return errors.Wrap(err, "headers could not be validated")
		}
	}

	return nil
}
