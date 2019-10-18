package streaming

import (
	"fmt"
	log "github.com/integration-system/isp-log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"io"
	"isp-convert-service/log_code"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"isp-convert-service/conf"
	"isp-convert-service/utils"

	"github.com/integration-system/isp-lib/backend"
	"github.com/integration-system/isp-lib/config"
	"github.com/integration-system/isp-lib/proto/stubs"
	s "github.com/integration-system/isp-lib/streaming"
	u "github.com/integration-system/isp-lib/utils"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
)

const (
	headerKeyContentDisposition = "Content-Disposition"
	headerKeyContentType        = "Content-Type"
	headerKeyContentLength      = "Content-Length"
	headerKeyTransferEncoding   = "Transfer-Encoding"

	ErrorMsgInternal   = "Internal server error"
	ErrorMsgInvalidArg = "Not able to read request body"
)

func SendMultipartData(ctx *fasthttp.RequestCtx, method string) {
	cfg := config.GetRemote().(*conf.RemoteConfig)
	timeout := cfg.GetStreamInvokeTimeout()
	bufferSize := cfg.GetTransferFileBufferSize()

	stream, cancel, err := openStream(&ctx.Request.Header, method, timeout)
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()
	if err != nil {
		utils.LogRequestHandlerError(log_code.TypeData.ProxyMultipart, method, err)
		utils.SendError(ErrorMsgInternal, codes.Internal, []interface{}{err.Error()}, ctx)
		return
	}

	form, err := ctx.MultipartForm()
	defer func() {
		if form != nil {
			_ = form.RemoveAll()
		}
	}()
	if err != nil {
		utils.LogRequestHandlerError(log_code.TypeData.ProxyMultipart, method, err)
		utils.SendError(ErrorMsgInvalidArg, codes.InvalidArgument, []interface{}{err.Error()}, ctx)
		return
	}

	formData := make(map[string]interface{}, len(form.Value))

	for k, v := range form.Value {
		if len(v) > 0 {
			formData[k] = v[0]
		}
	}

	response := make([]string, 0)
	buffer := make([]byte, bufferSize)
	ok := true
	eof := false
	for formDataName, files := range form.File {
		if len(files) == 0 {
			continue
		}
		file := files[0]
		fileName := file.Filename
		contentType := file.Header.Get(headerKeyContentType)
		contentLength := file.Size
		bf := s.BeginFile{
			FileName:      fileName,
			FormDataName:  formDataName,
			ContentType:   contentType,
			ContentLength: contentLength,
			FormData:      formData,
		}
		err = stream.Send(bf.ToMessage())
		if ok, eof = checkError(err, ctx); !ok || eof {
			break
		}

		f, err := file.Open()
		if ok, eof = checkError(err, ctx); !ok || eof {
			break
		}
		if ok, eof = transferFile(f, stream, buffer, ctx); ok {
			msg, err := stream.Recv()
			v, _, err := utils.GetResponse(msg, err)
			if err == nil {
				response = append(response, string(v))
			}
			ok = err == nil
		}

		if !ok || eof {
			break
		}
	}

	err = stream.CloseSend()
	if err != nil {
		utils.LogRequestHandlerError(log_code.TypeData.ProxyMultipart, method, err)
	}

	if ok {
		arrayBody := strings.Join(response, ",")
		_, err = ctx.WriteString("[" + arrayBody + "]")
		if err != nil {
			utils.LogRequestHandlerError(log_code.TypeData.ProxyMultipart, method, err)
		}
	}
}

func GetFile(ctx *fasthttp.RequestCtx, method string) {
	cfg := config.GetRemote().(*conf.RemoteConfig)
	timeout := cfg.GetStreamInvokeTimeout()

	req, err := utils.ReadJsonBody(ctx)
	if err != nil {
		utils.LogRequestHandlerError(log_code.TypeData.DownloadFile, method, err)
		utils.SendError(err.Error(), codes.InvalidArgument, nil, ctx)
		return
	}

	stream, cancel, err := openStream(&ctx.Request.Header, method, timeout)
	defer func() {
		if cancel != nil {
			cancel()
		}
	}()
	if err != nil {
		utils.LogRequestHandlerError(log_code.TypeData.DownloadFile, method, err)
		utils.SendError(ErrorMsgInternal, codes.Internal, []interface{}{err.Error()}, ctx)
		return
	}

	if req != nil {
		value := u.ConvertInterfaceToGrpcStruct(req)
		err := stream.Send(backend.WrapBody(value))
		if err != nil {
			utils.LogRequestHandlerError(log_code.TypeData.DownloadFile, method, err)
			utils.SendError(ErrorMsgInternal, codes.Internal, []interface{}{err.Error()}, ctx)
			return
		}
	}

	msg, err := stream.Recv()
	if err != nil {
		bytes, status, err := utils.GetResponse(nil, err)
		if err == nil {
			ctx.SetStatusCode(status)
			ctx.SetBody(bytes)
		}
		return
	}
	bf := s.BeginFile{}
	err = bf.FromMessage(msg)
	if err != nil {
		bytes, status, err := utils.GetResponse(nil, err)
		if err == nil {
			ctx.SetStatusCode(status)
			ctx.SetBody(bytes)
		}
		return
	}
	header := &ctx.Response.Header
	header.Set(headerKeyContentDisposition, fmt.Sprintf("attachment; filename=%s", bf.FileName))
	header.Set(headerKeyContentType, bf.ContentType)
	if bf.ContentLength > 0 {
		header.Set(headerKeyContentLength, strconv.Itoa(int(bf.ContentLength)))
	} else {
		header.Set(headerKeyTransferEncoding, "chunked")
	}

	for {
		msg, err := stream.Recv()
		if s.IsEndOfFile(msg) || err == io.EOF {
			break
		}
		if err != nil {
			utils.LogRequestHandlerError(log_code.TypeData.DownloadFile, method, err)
			break
		}
		bytes := msg.GetBytesBody()
		if bytes == nil {
			log.WithMetadata(map[string]interface{}{
				log_code.MdTypeData: log_code.TypeData.DownloadFile,
				log_code.MdMethod:   method,
			}).Errorf(log_code.WarnRequestHandler, "method %s. expected bytes array", method)
			break
		}
		_, err = ctx.Write(bytes)
		if err != nil {
			utils.LogRequestHandlerError(log_code.TypeData.DownloadFile, method, err)
			break
		}
	}
}

func openStream(headers *fasthttp.RequestHeader, method string, timeout time.Duration) (isp.BackendService_RequestStreamClient, context.CancelFunc, error) {
	client, err := utils.GetGrpcClient()
	if err != nil {
		return nil, nil, err
	}
	md, _ := utils.MakeMetadata(headers, method)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	stream, err := client.RequestStream(ctx)
	if err != nil {
		return nil, nil, err
	}
	return stream, cancel, nil
}

func transferFile(f multipart.File, stream isp.BackendService_RequestStreamClient, buffer []byte, ctx *fasthttp.RequestCtx) (bool, bool) {
	ok := true
	eof := false
	for {
		n, err := f.Read(buffer)
		if n > 0 {
			err = stream.Send(&isp.Message{Body: &isp.Message_BytesBody{buffer[:n]}})
			if ok, eof = checkError(err, ctx); !ok || eof {
				break
			}
		}
		if err != nil {
			if ok, eof = checkError(err, ctx); ok && eof {
				err = stream.Send(s.FileEnd())
				ok, eof = checkError(err, ctx)
			}
			break
		}
	}
	return ok, eof
}

func checkError(err error, ctx *fasthttp.RequestCtx) (bool, bool) {
	if err != nil {
		if err != io.EOF {
			utils.LogRequestHandlerError(log_code.TypeData.DownloadFile, "", err)
			utils.SendError(ErrorMsgInternal, codes.Internal, []interface{}{err.Error()}, ctx)
			return false, false
		}
		return true, true
	}
	return true, false
}
