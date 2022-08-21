package viewerresponse

import "github.com/edwardofclt/cloudfront-emulator/internal/types"

type ViewerResponseEvent struct {
}

func New() types.CloudfrontEvent {
	return &ViewerResponseEvent{}
}

func (e *ViewerResponseEvent) Execute(config types.CloudfrontEventConfig) error {
	return nil
}
