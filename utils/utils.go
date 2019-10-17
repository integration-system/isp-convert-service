package utils

import (
	"github.com/golang/protobuf/proto"
	"github.com/integration-system/isp-lib/config"
	"github.com/integration-system/isp-lib/structure"
	log "github.com/integration-system/isp-log"
	"google.golang.org/grpc/codes"
	"isp-convert-service/conf"
	"isp-convert-service/log_code"
	"net/http"
	"strings"

	"isp-convert-service/invoker"

	"github.com/golang/protobuf/ptypes/struct"
	"github.com/integration-system/isp-lib/backend"
	http2 "github.com/integration-system/isp-lib/http"
	"github.com/integration-system/isp-lib/proto/stubs"
	"github.com/integration-system/isp-lib/utils"
	"github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	JsonContentType = "application/json; charset=utf-8"
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

func GetResponse(msg *isp.Message, err error) ([]byte, int, error) {
	if err != nil {
		errorBody, errorStatus := convertError(err)
		return errorBody, errorStatus, nil
	}

	bytes := msg.GetBytesBody()
	if bytes != nil {
		return bytes, http.StatusOK, nil
	}
	result := backend.ResolveBody(msg)
	data := utils.ConvertGrpcStructToInterface(result)
	byteResponse, err := json.Marshal(data)
	return byteResponse, http.StatusOK, err
}

func MakeMetadata(r *fasthttp.RequestHeader, method string) (metadata.MD, string) {
	method = strings.TrimPrefix(method, "/api/")
	md := metadata.Pairs(utils.ProxyMethodNameHeader, method)
	r.VisitAll(func(key, v []byte) {
		lowerHeader := strings.ToLower(string(key))
		if len(v) > 0 && strings.HasPrefix(lowerHeader, "x-") {
			md = metadata.Join(md, metadata.Pairs(lowerHeader, string(v)))
		}
	})
	return md, method
}

func LogRequestHandlerError(typeData, method string, err error) {
	log.WithMetadata(map[string]interface{}{
		log_code.MdTypeData: typeData,
		log_code.MdMethod:   method,
	}).Warn(log_code.WarnRequestHandler, err)
}

func SendError(errorMessage string, errorCode codes.Code, details []interface{}, ctx *fasthttp.RequestCtx) {
	grpcCode := errorCode.String()

	structureError := structure.GrpcError{
		ErrorMessage: errorMessage,
		ErrorCode:    grpcCode,
		Details:      details,
	}

	ctx.SetStatusCode(http2.CodeToHttpStatus(errorCode))
	msg, _ := json.Marshal(structureError)
	_, _ = ctx.Write(msg)
}

func GetGrpcClient() (isp.BackendServiceClient, error) {
	return invoker.RouterClient.Conn()
}

func convertError(err error) ([]byte, int) {
	s, ok := status.FromError(err)
	if ok {
		cfg := config.GetRemote().(*conf.RemoteConfig)
		if cfg.EnableOriginalProtoErrors {
			if body, err := proto.Marshal(s.Proto()); err != nil {
				return []byte(utils.ServiceError), http.StatusServiceUnavailable
			} else {
				return body, http2.CodeToHttpStatus(s.Code())
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
				log.Warn(log_code.WarnConvertErrorDataMarshalResponse, err)
				return []byte(utils.ServiceError), http.StatusServiceUnavailable
			} else {
				return errorData, http2.CodeToHttpStatus(s.Code())
			}
		}
	} else {
		return []byte(utils.ServiceError), http.StatusServiceUnavailable
	}
}
