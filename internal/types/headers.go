package types

type ReadOnlyHeader map[string]struct{}

var AlwaysReadOnlyHeaders = ReadOnlyHeader{
	"connection":                    {},
	"expect":                        {},
	"keep-alive":                    {},
	"proxy-authenticate":            {},
	"proxy-authorization":           {},
	"proxy-connection":              {},
	"trailer":                       {},
	"upgrade":                       {},
	"x-accel-buffering":             {},
	"x-accel-charset":               {},
	"x-accel-limit-rate":            {},
	"x-accel-redirect":              {},
	"x-amz-cf-(.*)":                 {},
	"x-amzn-auth":                   {},
	"x-amzn-cf-billing":             {},
	"x-amzn-cf-id":                  {},
	"x-amzn-cf-xff":                 {},
	"x-amzn-errortype":              {},
	"x-amzn-fle-profile":            {},
	"x-amzn-header-count":           {},
	"x-amzn-header-order":           {},
	"x-amzn-lambda-integration-tag": {},
	"x-amzn-requestid":              {},
	"x-cache":                       {},
	"x-edge-(.*)":                   {},
	"x-forwarded-proto":             {},
	"x-real-ip":                     {},
}

var ViewerRequestReadOnlyHeaders = ReadOnlyHeader{
	"content-length":    {},
	"host":              {},
	"transfer-encoding": {},
	"via":               {},
}

var OriginRequestReadOnlyHeaders = ReadOnlyHeader{
	"accept-encoding":     {},
	"content-length":      {},
	"if-modified-since":   {},
	"if-none-match":       {},
	"if-range":            {},
	"if-unmodified-since": {},
	"transfer-encoding":   {},
	"via":                 {},
}
