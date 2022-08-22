package types

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

func (p *RequestPayload) EncodeJSON() ([]byte, error) {
	return json.Marshal(p)
}

func ParseRequestBody(response []byte) (CfRequest, error) {
	p := CfRequest{}
	err := json.Unmarshal(response, &p)
	return p, err
}

func ParseResponseBody(response []byte) (CfResponse, error) {
	p := CfResponse{}
	err := json.Unmarshal(response, &p)
	return p, err
}

func MergeRequestBody(request *CfRequest, respData CfRequest) {
	if respData.URI != request.URI {
		request.URI = respData.URI
	}

	if respData.Status != request.Status {
		request.Status = respData.Status
	}

	if respData.Body != request.Body {
		request.Body = respData.Body
	}

	if respData.Headers != request.Headers {
		copyOfRequest := *request.Headers
		for key, header := range *respData.Headers {
			copyOfRequest[key] = header
		}
		request.Headers = &copyOfRequest
	}
	return
}

func MergeResponseBody(request *CfResponse, respData CfResponse) {
	if respData.URI != request.URI {
		request.URI = respData.URI
	}

	if respData.Status != request.Status {
		request.Status = respData.Status
	}

	if respData.Headers != request.Headers {
		MergeHeaders(request.Headers, respData.Headers)
	}

	return
}

func SendErrorResponse(w http.ResponseWriter, content, payload string) {
	w.WriteHeader(502)
	w.Header().Add("content-type", "text/html")
	fmt.Fprintf(w, `<html><body><h1>502 Error</h1><hr /><p><em>If you're seeing this it means something went wrong executing the logic in your lambda... More context can be found below:</em></p><hr /><pre>%s</pre><hr /><pre>%s</pre></body></html>`, content, payload)
}

func CheckHeaders(eventType EventType, reqHeaders CfHeaderArray, respHeaders CfHeaderArray) error {
	for respHeaderKey, header := range respHeaders {
		if strings.ToLower(header[0].Key) != strings.ToLower(respHeaderKey) {
			return fmt.Errorf("got %s saw key value %s", header[0].Key, respHeaderKey)
		}

		if err := CheckReadOnlyHeader(AlwaysReadOnlyHeaders, header, reqHeaders); err != nil {
			return errors.Wrap(err, "read only headeres were modified")
		}

		if eventType == ViewerRequest {
			if err := CheckReadOnlyHeader(ViewerRequestReadOnlyHeaders, header, reqHeaders); err != nil {
				return errors.Wrap(err, "read only headeres were modified")
			}
		}
	}

	return nil
}

func CheckReadOnlyHeader(headerList ReadOnlyHeader, header []CfHeader, reqHeaders CfHeaderArray) error {
	if _, ok := headerList[header[0].Key]; ok {
		if reqHeader, ok := reqHeaders[strings.ToLower(header[0].Key)]; ok {
			if reqHeader[0].Value != header[0].Value {
				return fmt.Errorf("this header is never allowed to be modified: %s got %s expected value %s", header[0].Key, header[0].Value, reqHeader[0].Value)
			}
		} else {
			return fmt.Errorf("this header is never allowed to be modified: %s", header[0].Key)
		}
	}
	return nil
}

func MergeHeaders(to *CfHeaderArray, from *CfHeaderArray) {
	if to == nil {
		to = &CfHeaderArray{}
	}
	if from == nil {
		return
	}

	finalCopy := *to
	if from != nil {
		for key, header := range *from {
			finalCopy[key] = header
		}
	}
	to = &finalCopy
}
