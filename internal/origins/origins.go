package origins

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/edwardofclt/cloudfront-emulator/internal/types"
	"github.com/pkg/errors"
)

type OriginRequestConfig struct {
	HTTPRequest *http.Request
	CfRequest   types.RequestPayload
	Origin      types.Origin
}

func Request(config *OriginRequestConfig) (*types.CfResponse, error) {
	requestURL := filepath.Clean(fmt.Sprintf("%s/%s/%s", config.Origin.Domain, config.Origin.Path, config.HTTPRequest.URL.Path))
	fullURL := fmt.Sprintf("%s://%s", strings.ToLower(strings.Split(config.HTTPRequest.Proto, "/")[0]), requestURL)

	originRequest, _ := http.NewRequest(config.HTTPRequest.Method, fullURL, config.HTTPRequest.Body)

	for _, value := range *config.CfRequest.Records[0].Cf.Request.Headers {
		originRequest.Header.Add(value[0].Key, value[0].Value)
	}

	originResponse, err := http.DefaultClient.Do(originRequest)
	if err != nil {
		return nil, errors.Wrap(err, "error while fetching the origin")
	}

	originResponseData, err := ioutil.ReadAll(originResponse.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error while parsing origin response")
	}

	statusCode := strconv.Itoa(originResponse.StatusCode)

	finalResponse := &types.CfResponse{
		Body:    originResponseData,
		Status:  &statusCode,
		Headers: &types.CfHeaderArray{},
	}

	for key, value := range originResponse.Header {
		header := *finalResponse.Headers
		header[key] = []types.CfHeader{
			{
				Key:   key,
				Value: value[0],
			},
		}
		finalResponse.Headers = &header
	}

	return finalResponse, nil
}
