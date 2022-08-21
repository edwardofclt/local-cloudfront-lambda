package originresponse

import "github.com/edwardofclt/cloudfront-emulator/internal/types"

type OriginResponseEvent struct {
}

func New() types.CloudfrontEvent {
	return &OriginResponseEvent{}
}

func (e *OriginResponseEvent) Execute(config types.CloudfrontEventConfig) error {
	return nil
}
