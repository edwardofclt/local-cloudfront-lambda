package cloudfront

import (
	"net/http"

	"github.com/edwardofclt/cloudfront-emulator/internal/types"
	"github.com/pkg/errors"
)

func writeResponseHeaders(w http.ResponseWriter, respData types.CfResponse) {
	if respData.Headers == nil {
		return
	}

	for _, header := range *respData.Headers {
		for _, val := range header {
			w.Header().Add(val.Key, val.Value)
		}
	}
}

func validateResponse(eventType types.EventType, reqData types.RequestPayload, respData types.CfResponse) error {
	if respData.Headers != nil {
		if err := checkHeaders(eventType, *reqData.Records[0].Cf.Request.Headers, *respData.Headers); err != nil {
			return errors.Wrap(err, "headers could not be validated")
		}
	}

	return nil
}
