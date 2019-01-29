package streaming

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"isp-convert-service/conf"
	"isp-convert-service/utils"

	"github.com/integration-system/isp-lib/backend"
	"github.com/integration-system/isp-lib/config"
	"github.com/integration-system/isp-lib/logger"
	"github.com/integration-system/isp-lib/proto/stubs"
	s "github.com/integration-system/isp-lib/streaming"
	u "github.com/integration-system/isp-lib/utils"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
)

func SendMultipartData(ctx *fasthttp.RequestCtx, method string) {
	cfg := config.GetRemote().(*conf.RemoteConfig)
	timeout := time.Duration(cfg.MultipartDataTransferTimeoutMs) * time.Millisecond
	bufferSize := cfg.MultipartDataTransferBufferSizeBytes
	if timeout <= 0 {
		timeout = utils.DefaultTransferTimeout
	}
	if bufferSize <= 0 {
		bufferSize = utils.DefaultBufferSize
	}

	stream, cancel, err := openStream(&ctx.Request.Header, method, timeout)
	defer cancel()
	if err != nil {
		utils.WriteAndLogError("Internal server error", err, ctx, http.StatusInternalServerError)
		return
	}

	form, err := ctx.MultipartForm()
	defer func() {
		if form != nil {
			form.RemoveAll()
		}
	}()
	if err != nil {
		utils.WriteAndLogError("Not able to read request body", err, ctx, http.StatusBadRequest)
		return
	}

	formData := make(map[string]interface{}, len(form.Value))

	for k, v := range form.Value {
		if len(v) > 0 {
			formData[k] = v[0]
		}
	}

	response := make([]interface{}, 0)
	buffer := make([]byte, bufferSize)
	ok := true
	eof := false
	for formDataName, files := range form.File {
		if len(files) == 0 {
			continue
		}
		file := files[0]
		fileName := file.Filename
		contentType := file.Header.Get("Content-Type")
		contentLength := file.Size
		bf := s.BeginFile{FileName: fileName,
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
			v, err := utils.ConvertAndWriteResponse(msg, err, ctx)
			if err == nil {
				response = append(response, v)
			}
			ok = err == nil
		}

		if !ok || eof {
			break
		}
	}

	err = stream.CloseSend()
	if err != nil {
		logger.Warn(err)
	}

	if ok {
		bytes, _ := json.Marshal(response)
		_, err = ctx.Write(bytes)
		if err != nil {
			logger.Warn(err)
		}
	}
}

func GetFile(ctx *fasthttp.RequestCtx, method string) {
	cfg := config.GetRemote().(*conf.RemoteConfig)
	timeout := time.Duration(cfg.MultipartDataTransferTimeoutMs)
	if timeout <= 0 {
		timeout = utils.DefaultTransferTimeout
	}

	req, err := utils.ReadJsonBody(ctx)
	if err != nil {
		utils.WriteAndLogError(err.Error(), err, ctx, http.StatusBadRequest)
		return
	}

	stream, cancel, err := openStream(&ctx.Request.Header, method, timeout)
	defer cancel()
	if err != nil {
		utils.WriteAndLogError("Internal server error", err, ctx, http.StatusInternalServerError)
		return
	}

	if req != nil {
		value := u.ConvertInterfaceToGrpcStruct(req)
		err := stream.Send(backend.WrapBody(value))
		if err != nil {
			utils.WriteAndLogError("Internal server error", err, ctx, http.StatusInternalServerError)
			return
		}
	}

	msg, err := stream.Recv()
	if err != nil {
		utils.ConvertAndWriteResponse(nil, err, ctx)
		return
	}
	bf := s.BeginFile{}
	err = bf.FromMessage(msg)
	if err != nil {
		utils.ConvertAndWriteResponse(nil, err, ctx)
		return
	}
	header := &ctx.Request.Header
	header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", bf.FileName))
	header.Set("Content-Type", bf.ContentType)
	if bf.ContentLength > 0 {
		header.Set("Content-Length", strconv.Itoa(int(bf.ContentLength)))
	} else {
		header.Set("Transfer-Encoding", "chunked")
	}

	for {
		msg, err := stream.Recv()
		if s.IsEndOfFile(msg) || err == io.EOF {
			break
		}
		if err != nil {
			logger.Warn(err)
			break
		}
		bytes := msg.GetBytesBody()
		if bytes == nil {
			logger.Errorf("Method %s. Expected bytes array", method)
			break
		}
		_, err = ctx.Write(bytes)
		if err != nil {
			logger.Warn(err)
			break
		}
	}
}

func openStream(headers *fasthttp.RequestHeader, method string, timeout time.Duration) (isp.BackendService_RequestStreamClient, context.CancelFunc, error) {
	client, err := utils.GetGrpcClient()
	if err != nil {
		return nil, nil, err
	}
	md := utils.MakeMetadata(headers, method)
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
			utils.WriteAndLogError("Internal server error", err, ctx, http.StatusInternalServerError)
			return false, false
		}
		return true, true
	}
	return true, false
}
