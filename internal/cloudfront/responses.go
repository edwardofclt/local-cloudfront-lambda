package cloudfront

import (
	"net/http"

	"github.com/edwardofclt/cloudfront-emulator/internal/types"
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
