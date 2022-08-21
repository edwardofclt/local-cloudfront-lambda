package helpers

import (
	"github.com/edwardofclt/cloudfront-emulator/internal/types"
)

func MergeHeadersToFinalResponse(finalResponse *types.CfHeaderArray, headers *types.CfHeaderArray) {
	finalCopy := *finalResponse
	if headers != nil {
		for key, header := range *headers {
			finalCopy[key] = header
		}
	}
	finalResponse = &finalCopy
}
