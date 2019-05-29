package conf

import (
	"github.com/integration-system/isp-lib/structure"
)

type RemoteConfig struct {
	EnableOriginalProtoErrors            bool                          `schema:"Proxy error to protobuf, default: false"`
	ProxyGrpcErrorDetails                bool                          `schema:"Proxy first element from GRPC error details, default: false"`
	MultipartDataTransferTimeoutMs       int64                         `schema:"Multipart data transfer timeout,In milliseconds, default: 60000"`
	MultipartDataTransferBufferSizeBytes int64                         `schema:"Multipart data transfer buffer size,In bytes, default: 4 KB"`
	MaxRequestBodySizeBytes              int64                         `schema:"Max request body size,In bytes, default: 512 MB"`
	Metrics                              structure.MetricConfiguration `schema:"Metrics"`
}
