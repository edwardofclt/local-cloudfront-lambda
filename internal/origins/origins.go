package origins

import (
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/edwardofclt/cloudfront-emulator/internal/types"
	"github.com/pkg/errors"
)

type OriginRequestConfig struct {
	HTTPRequest *http.Request
	CfRequest   types.RequestPayload
	Origin      types.Origin
}

func Request(config *OriginRequestConfig) (*types.CfResponse, error) {
	fullURL := url.URL{
		Host:   config.Origin.Domain,
		Path:   filepath.Join(config.Origin.Path, config.CfRequest.Records[0].Cf.Request.URI),
		Scheme: strings.Split(config.HTTPRequest.Proto, "/")[0],
	}

	originRequest, _ := http.NewRequest(config.CfRequest.Records[0].Cf.Request.Method, fullURL.String(), config.HTTPRequest.Body)

	for _, value := range *config.CfRequest.Records[0].Cf.Request.Headers {
		if len(value) == 0 {
			continue
		}
		if originRequest == nil {
			continue
		}

		originRequest.Header.Add(value[0].Key, value[0].Value)
	}

	client := http.Client{
		Timeout: time.Second * 5,
	}

	originResponse, err := client.Do(originRequest)
	if err != nil {
		return nil, errors.Wrap(err, "error while fetching the origin")
	}

	originResponseData, err := io.ReadAll(originResponse.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error while parsing origin response")
	}

	statusCode := strconv.Itoa(originResponse.StatusCode)

	finalResponse := &types.CfResponse{
		BaseConfig: types.BaseConfig{
			Body:    aws.String(string(originResponseData)),
			Status:  &statusCode,
			Headers: &types.CfHeaderArray{},
		},
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
