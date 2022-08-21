package types

import (
	"net/http"
	"sync"

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

type Cloudfront struct {
	Server      *http.Server
	Handler     http.Handler
	PathToCerts string
	Wg          *sync.WaitGroup
}

type CloudfrontConfig struct {
	Address       *string           `mapstructure:"address"`
	Port          *int              `mapstructure:"port"`
	OriginConfigs map[string]Origin `mapstructure:"origins"`
	Behaviors     []Behavior        `mapstructure:"behaviors"`
}

type Origin struct {
	Domain string
	Path   string
}

type EventType string

const (
	ViewerRequest  EventType = "viewer-request"
	OriginRequest  EventType = "origin-request"
	OriginResponse EventType = "origin-response"
	ViewerResponse EventType = "viewer-response"
)

var EventTypes []EventType = []EventType{
	ViewerRequest,
	OriginRequest,
	OriginResponse,
	ViewerResponse,
}

type Behavior struct {
	Path   string
	Origin string
	Events map[EventType]Event
}

type Event struct {
	Path    string
	Handler string
}

type EventResponse struct {
}
