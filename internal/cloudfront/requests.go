package cloudfront

import (
	"net/http"
	"strings"

	"github.com/edwardofclt/cloudfront-emulator/internal/types"
	"github.com/google/uuid"
)

func writeRequestHeaders(w http.ResponseWriter, respData types.CfHeaderArray) {
	for _, header := range respData {
		for _, val := range header {
			w.Header().Add(val.Key, val.Value)
		}
	}
}

func generateRequestBody(requestId uuid.UUID, eventType types.EventType, r *http.Request) *types.CfRequest {
	p := &types.CfRequest{
		BaseConfig: types.BaseConfig{
			ClientIP:    strings.Split(r.RemoteAddr, ":")[0],
			Method:      r.Method,
			QueryString: r.URL.RawQuery,
			URI:         r.URL.Path,
			Headers:     parseHeaders(r.Header),
		},
	}
	return p
}

func parseHeaders(headers http.Header) *types.CfHeaderArray {
	h := &types.CfHeaderArray{}
	for key, val := range headers {
		vals := []types.CfHeader{}
		for _, v := range val {
			vals = append(vals, types.CfHeader{
				Key:   key,
				Value: v,
			})
		}
		hCopy := *h
		hCopy[strings.ToLower(key)] = vals
		h = &hCopy
	}
	return h
}
