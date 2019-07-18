package conf

import (
	"github.com/integration-system/isp-journal/rx"
	"github.com/integration-system/isp-lib/structure"
	"time"
)

const (
	KB = int64(1024)
	MB = int64(1 << 20)

	defaultSyncTimeout   = 30 * time.Second
	defaultStreamTimeout = 60 * time.Second

	defaultBufferSize         = 4 * KB
	defaultMaxRequestBodySize = 512 * MB
)

type RemoteConfig struct {
	EnableOriginalProtoErrors            bool                          `schema:"Proxy error to protobuf, default: false"`
	ProxyGrpcErrorDetails                bool                          `schema:"Proxy first element from GRPC error details, default: false"`
	MultipartDataTransferBufferSizeBytes int64                         `schema:"Multipart data transfer buffer size,In bytes, default: 4 KB"`
	MaxRequestBodySizeBytes              int64                         `schema:"Max request body size,In bytes, default: 512 MB"`
	SyncInvokeMethodTimeoutMs            int64                         `schema:"Timeout to invoke sync method, default: 30000"`
	StreamInvokeMethodTimeoutMs          int64                         `schema:"Timeout to transfer and handle file, default: 60000"`
	Metrics                              structure.MetricConfiguration `schema:"Metrics"`
	Journal                              rx.Config                     `schema:"Journal"`
	JournalingMethodsPatterns            []string                      `schema:"Journaling methods patterns"`
}

func (cfg RemoteConfig) GetSyncInvokeTimeout() time.Duration {
	if cfg.SyncInvokeMethodTimeoutMs <= 0 {
		return defaultSyncTimeout
	}
	return time.Duration(cfg.SyncInvokeMethodTimeoutMs) * time.Millisecond
}

func (cfg RemoteConfig) GetStreamInvokeTimeout() time.Duration {
	if cfg.StreamInvokeMethodTimeoutMs <= 0 {
		return defaultStreamTimeout
	}
	return time.Duration(cfg.StreamInvokeMethodTimeoutMs) * time.Millisecond
}

func (cfg RemoteConfig) GetTransferFileBufferSize() int64 {
	if cfg.MultipartDataTransferBufferSizeBytes <= 0 {
		return defaultBufferSize
	}
	return cfg.MultipartDataTransferBufferSizeBytes
}

func (cfg RemoteConfig) GetMaxRequestBodySize() int64 {
	if cfg.MaxRequestBodySizeBytes <= 0 {
		return defaultMaxRequestBodySize
	}
	return cfg.MaxRequestBodySizeBytes
}
