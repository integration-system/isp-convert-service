package conf

import (
	"github.com/integration-system/isp-lib/structure"
)

type RemoteConfig struct {
	EnableOriginalProtoErrors            bool                          `schema:"Enable proto response, default: false"`
	MultipartDataTransferTimeoutMs       int64                         `schema:"Multipart data transfer timeout,In milliseconds, default: 60000"`
	MultipartDataTransferBufferSizeBytes int64                         `schema:"Multipart data transfer buffer size,In bytes, default: 4 KB"`
	MaxRequestBodySizeBytes              int64                         `schema:"Max request body size,In bytes, default: 512 MB"`
	Metrics                              structure.MetricConfiguration `schema:"Metrics"`
}
