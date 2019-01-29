package conf

import (
	"github.com/integration-system/isp-lib/structure"
)

type RemoteConfig struct {
	MultipartDataTransferTimeoutMs       int64
	MultipartDataTransferBufferSizeBytes int64
	MaxRequestBodySizeBytes              int64
	Metrics                              structure.MetricConfiguration
}
