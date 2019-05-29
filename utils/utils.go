package utils

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/integration-system/isp-lib/config"
	"isp-convert-service/conf"
	"net/http"
	"strings"

	"isp-convert-service/invoker"
	"isp-convert-service/structure"

	"github.com/golang/protobuf/ptypes/struct"
	"github.com/integration-system/isp-lib/backend"
	http2 "github.com/integration-system/isp-lib/http"
	"github.com/integration-system/isp-lib/logger"
	"github.com/integration-system/isp-lib/proto/stubs"
	"github.com/integration-system/isp-lib/utils"
	"github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"time"
)

const (
	JsonContentType = "application/json; charset=utf-8"

	KB                        = int64(1024)
	MB                        = int64(1 << 20)
	DefaultBufferSize         = 4 * KB
	DefaultMaxRequestBodySize = 512 * MB
	DefaultTransferTimeout    = 1 * time.Minute
)

var (
	json = jsoniter.ConfigFastest
)

func ReadJsonBody(ctx *fasthttp.RequestCtx) (interface{}, error) {
	requestBody := ctx.Request.Body()
	var body interface{}
	if len(requestBody) == 0 {
		requestBody = []byte("{}")
	}
	if requestBody[0] == '{' {
		body = make(map[string]interface{})
	} else if requestBody[0] == '[' {
		body = make([]interface{}, 0)
	} else {
		return nil, errors.New("Invalid json format. Expected object or array")
	}

	err := json.Unmarshal(requestBody, &body)

	if err != nil {
		return nil, errors.New("Not able to read request body")
	}
	return body, err
}

func ConvertAndWriteResponse(msg *isp.Message, err error, ctx *fasthttp.RequestCtx) ([]byte, error) {
	if err != nil {
		s, ok := status.FromError(err)
		if ok {
			ctx.SetStatusCode(http2.CodeToHttpStatus(s.Code()))
			cfg := config.GetRemote().(*conf.RemoteConfig)
			if cfg.EnableOriginalProtoErrors {
				msg, err := proto.Marshal(s.Proto())
				if err != nil {
					writeServiceError(ctx, err)
				} else {
					ctx.Write(msg)
				}
			} else {
				details := s.Details()
				newDetails := make([]interface{}, len(details))
				for i, detail := range details {
					switch typeOfDetail := detail.(type) {
					case *structpb.Struct:
						newDetails[i] = utils.ConvertGrpcStructToInterface(
							&structpb.Value{Kind: &structpb.Value_StructValue{StructValue: typeOfDetail}},
						)
					case *isp.Message:
						newDetails[i] = utils.ConvertGrpcStructToInterface(
							backend.ResolveBody(typeOfDetail),
						)
					default:
						newDetails[i] = typeOfDetail
					}
				}

				var respBody interface{}
				if cfg.ProxyGrpcErrorDetails && len(newDetails) > 0 {
					respBody = newDetails[0]
				} else {
					respBody = structure.GrpcError{ErrorMessage: s.Message(), ErrorCode: s.Code().String(), Details: newDetails}
				}
				if errorData, err := json.Marshal(respBody); err != nil {
					writeServiceError(ctx, err)
				} else {
					ctx.Write(errorData)
				}
			}
		} else {
			writeServiceError(ctx, err)
		}
		return nil, nil
	}

	bytes := msg.GetBytesBody()
	if bytes != nil {
		return bytes, nil
	}
	result := backend.ResolveBody(msg)
	data := utils.ConvertGrpcStructToInterface(result)
	byteResponse, err := json.Marshal(data)
	return byteResponse, err
}

func MakeMetadata(r *fasthttp.RequestHeader, method string) metadata.MD {
	if strings.HasPrefix(method, "/api/") {
		method = strings.TrimPrefix(method, "/api/")
	}
	md := metadata.Pairs(utils.ProxyMethodNameHeader, method)
	r.VisitAll(func(key, v []byte) {
		lowerHeader := strings.ToLower(string(key))
		if len(v) > 0 && strings.HasPrefix(lowerHeader, "x-") {
			md = metadata.Join(md, metadata.Pairs(lowerHeader, string(v)))
		}
	})
	return md
}

func WriteAndLogError(message string, err error, ctx *fasthttp.RequestCtx, code int) {
	logger.Warn(message, err)

	ctx.SetStatusCode(code)
	ctx.Write([]byte(fmt.Sprintf("{\"errorMessage\": \"%s\"}", message)))
}

func GetGrpcClient() (isp.BackendServiceClient, error) {
	return invoker.RouterClient.Conn()
}

func writeServiceError(ctx *fasthttp.RequestCtx, err error) {
	logger.Warn(err)
	ctx.SetStatusCode(http.StatusServiceUnavailable)
	ctx.Write([]byte(utils.ServiceError))
}
