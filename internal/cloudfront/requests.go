package cloudfront

import (
	"net/http"

	"github.com/edwardofclt/cloudfront-emulator/internal/types"
)

func writeRequestHeaders(w http.ResponseWriter, respData types.CfHeaderArray) {
	for _, header := range respData {
		for _, val := range header {
			w.Header().Add(val.Key, val.Value)
		}
	}
}
