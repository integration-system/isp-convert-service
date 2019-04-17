package controllers

import (
	"github.com/integration-system/isp-lib/proto/stubs"
	"isp-convert-service/service"
	"mime"
	"net/http"
	"time"

	"isp-convert-service/streaming"
	"isp-convert-service/utils"

	u "github.com/integration-system/isp-lib/utils"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
	_ "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/metadata"
)

func HandlerAllRequest(ctx *fasthttp.RequestCtx) {
	currentTime := time.Now()

	uri := string(ctx.RequestURI())
	proxyRequestHandle(ctx, uri)

	executionTime := time.Since(currentTime) / 1e6
	metrics := service.GetMetrics()
	metrics.UpdateStatusCounter(ctx.Response.StatusCode())
	if ctx.Response.StatusCode() == 200 {
		metrics.UpdateResponseTime(executionTime)
		metrics.UpdateMethodResponseTime(uri, executionTime)
	}
}

func handleJson(c *fasthttp.RequestCtx, method string) {
	//body, err := utils.ReadJsonBody(c)
	body := c.Request.Body()
	/*if err != nil {
		utils.WriteAndLogError(err.Error(), err, c, http.StatusBadRequest)
		return
	}*/

	md := utils.MakeMetadata(&c.Request.Header, method)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client, err := utils.GetGrpcClient()
	if err != nil {
		utils.WriteAndLogError("Internal server error", err, c, http.StatusInternalServerError)
		return
	}

	//structBody := u.ConvertInterfaceToGrpcStruct(body)
	currentTime := time.Now()
	response, err := client.Request(ctx, &isp.Message{
		Body: &isp.Message_BytesBody{
			BytesBody: []byte(body),
		},
	})
	service.GetMetrics().UpdateRouterResponseTime(time.Since(currentTime) / 1e6)

	if data, err := utils.ConvertAndWriteResponse(response, err, c); err == nil {
		_, _ = c.Write(data)
	} else {
		utils.WriteAndLogError("Internal server error", err, c, http.StatusInternalServerError)
	}
}

func proxyRequestHandle(ctx *fasthttp.RequestCtx, method string) {
	isMultipart := isMultipart(ctx)
	isExpectFile := string(ctx.Request.Header.Peek(u.ExpectFileHeader)) == "true"
	if isMultipart {
		ctx.Response.Header.SetContentType(utils.JsonContentType)
		streaming.SendMultipartData(ctx, method)
	} else if isExpectFile {
		streaming.GetFile(ctx, method)
	} else {
		ctx.Response.Header.SetContentType(utils.JsonContentType)
		handleJson(ctx, method)
	}
}

func isMultipart(ctx *fasthttp.RequestCtx) bool {
	if !ctx.IsPost() {
		return false
	}
	v := string(ctx.Request.Header.ContentType())
	if v == "" {
		return false
	}
	d, params, err := mime.ParseMediaType(v)
	if err != nil || d != "multipart/form-data" {
		return false
	}
	_, ok := params["boundary"]
	if !ok {
		return false
	}
	return true
}
