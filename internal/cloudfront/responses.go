package cloudfront

import (
	"net/http"

	"github.com/pkg/errors"
)

func writeResponseHeaders(w http.ResponseWriter, respData CfResponse) {
	if respData.Headers == nil {
		return
	}

	for _, header := range *respData.Headers {
		for _, val := range header {
			w.Header().Add(val.Key, val.Value)
		}
	}
}

func validateResponse(eventType EventType, reqData RequestPayload, respData CfResponse) error {
	if respData.Headers != nil {
		if err := checkHeaders(eventType, *reqData.Records[0].Cf.Request.Headers, *respData.Headers); err != nil {
			return errors.Wrap(err, "headers could not be validated")
		}
	}

	return nil
}
