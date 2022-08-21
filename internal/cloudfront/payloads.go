package cloudfront

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/edwardofclt/cloudfront-emulator/internal/types"
	"github.com/google/uuid"
)

type RequestPayload struct {
	Records []Record `json:"Records"`
}

type Record struct {
	Cf CfRecord `json:"cf"`
}

type CfRecord struct {
	Config   CfType      `json:"config"`
	Request  *CfRequest  `json:"request,omitempty"`
	Response *CfResponse `json:"response,omitempty"`
}

type CfRequest struct {
	Headers           *CfHeaderArray `json:"headers,omitempty"`
	ClientIP          string         `json:"clientIp"`
	Method            string         `json:"method,omitempty"`
	QueryString       string         `json:"querystring,omitempty"`
	URI               string         `json:"uri,omitempty"`
	Origin            *CfOrigin      `json:"origin,omitempty"`
	Status            *string        `json:"status,omitempty"`
	Body              string         `json:"body,omitempty"`
	StatusDescription *string        `json:"statusDescription,omitempty"`
}

type CfHeaderArray map[string][]CfHeader

type CfResponse struct {
	Status            *string        `json:"status,omitempty"`
	Body              []byte         `json:"body,omitempty"`
	StatusDescription *string        `json:"statusDescription,omitempty"`
	Headers           *CfHeaderArray `json:"headers,omitempty"`
	ClientIP          string         `json:"clientIp"`
	Method            string         `json:"method,omitempty"`
	QueryString       string         `json:"querystring,omitempty"`
	URI               string         `json:"uri,omitempty"`
	Origin            *CfOrigin      `json:"origin,omitempty"`
}

type CfOrigin struct {
	Custom *CfCustomOrigin `json:"custom,omitempty"`
	S3     *CfS3Origin     `json:"s3,omitempty"`
}

type CfS3Origin struct {
	DomainName   string        `json:"domainName,omitempty"`
	Region       string        `json:"region,omitempty"`
	AuthMethod   string        `json:"authMethod,omitempty"`
	Path         string        `json:"path,omitempty"`
	CustomHeader CfHeaderArray `json:"customHeaders"`
}

type CfCustomOrigin struct {
	CustomHeaders    *CfHeaderArray `json:"customHeaders,omitempty"`
	DomainName       string         `json:"domainName,omitempty"`
	KeepAliveTimeout uint           `json:"keepaliveTimeout"`
	Path             string         `json:"path,omitempty"`
	Port             uint           `json:"port"`
	Protocol         string         `json:"protocol,omitempty"`
	ReadTimeout      uint           `json:"readTimeout,omitempty"`
	SSLProtocols     []string       `json:"sslProtocols,omitempty"`
}

type CfHeader struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type CfType struct {
	DistributionName string    `json:"distributionName,omitempty"`
	DistributionId   string    `json:"distributionId,omitempty"`
	EventType        EventType `json:"eventType"`
	RequestId        uuid.UUID `json:"requestId,omitempty"` // using uuid to distinquish requests for debugging
}

func (p *RequestPayload) EncodeJSON() ([]byte, error) {
	return json.Marshal(p)
}

func generateRequestBody(requestId uuid.UUID, eventType types.EventType, r *http.Request) *types.CfRequest {
	p := &types.CfRequest{
		ClientIP:    strings.Split(r.RemoteAddr, ":")[0],
		Method:      r.Method,
		QueryString: r.URL.RawQuery,
		URI:         r.URL.Path,
		Headers:     parseHeaders(r.Header),
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

func parseRequestData(response string) (types.CfRequest, error) {
	p := types.CfRequest{}
	err := json.Unmarshal([]byte(response), &p)
	return p, err
}

func parseResponseData(response string) (types.CfResponse, error) {
	p := types.CfResponse{}
	err := json.Unmarshal([]byte(response), &p)
	return p, err
}
