package originrequest

import "github.com/edwardofclt/cloudfront-emulator/internal/types"

type OriginRequestEvent struct {
}

func New() types.CloudfrontEvent {
	return &OriginRequestEvent{}
}

func (e *OriginRequestEvent) Execute(config types.CloudfrontEventConfig) error {
	return nil
}
