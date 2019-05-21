package main

import (
	"github.com/integration-system/isp-lib/config/schema"
	"github.com/integration-system/isp-lib/structure"
	"isp-convert-service/controllers"
	"isp-convert-service/service"
	"os"
	"sync"
	"time"

	"isp-convert-service/conf"
	"isp-convert-service/invoker"

	"github.com/buaazp/fasthttprouter"
	"github.com/integration-system/isp-lib/bootstrap"
	"github.com/integration-system/isp-lib/config"
	"github.com/integration-system/isp-lib/logger"
	"github.com/integration-system/isp-lib/metric"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
	u "isp-convert-service/utils"
)

var (
	version = "0.1.0"
	date    = "undefined"

	srvLock = sync.Mutex{}
	httpSrv *fasthttp.Server
)

func main() {
	bootstrap.
		ServiceBootstrap(&conf.Configuration{}, &conf.RemoteConfig{}).
		OnLocalConfigLoad(onLocalConfigLoad).
		DefaultRemoteConfigPath(schema.ResolveDefaultConfigPath("default_remote_config.json")).
		SocketConfiguration(socketConfiguration).
		DeclareMe(routesData).
		RequireModule("router", invoker.HandleRoutesAddresses, true).
		OnShutdown(onShutdown).
		OnRemoteConfigReceive(onRemoteConfigReceive).
		Run()
}

func onLocalConfigLoad(cfg *conf.Configuration) {
	logger.Infof("Outer http address is %s", cfg.HttpOuterAddress.GetAddress())
}

func onRemoteConfigReceive(cfg, oldRemoteConfig *conf.RemoteConfig) {
	createRestServer(cfg)
	metric.InitCollectors(cfg.Metrics, oldRemoteConfig.Metrics)
	metric.InitHttpServer(cfg.Metrics)
	//metric.InitStatusChecker("router-grpc", helper.GetRoutersAndStatus)
	service.InitMetrics()
}

// Start a HTTP server.
func createRestServer(appConfig *conf.RemoteConfig) {
	router := fasthttprouter.New()
	// === REST ===
	router.Handle("POST", "/api/*any", controllers.HandlerAllRequest)
	router.Handle("GET", "/api/*any", controllers.HandlerAllRequest)

	maxRequestBodySize := appConfig.MaxRequestBodySizeBytes
	if appConfig.MaxRequestBodySizeBytes <= 0 {
		maxRequestBodySize = u.DefaultMaxRequestBodySize
	}

	srvLock.Lock()

	if httpSrv != nil {
		if err := httpSrv.Shutdown(); err != nil {
			logger.Warn(err)
		}
	}

	cfg := config.Get().(*conf.Configuration)
	restAddress := cfg.HttpInnerAddress.GetAddress()
	httpSrv = &fasthttp.Server{
		Handler:            router.Handler,
		WriteTimeout:       time.Second * 60,
		ReadTimeout:        time.Second * 60,
		MaxRequestBodySize: int(maxRequestBodySize),
	}
	logger.Infof("Serving at %s ...", restAddress)
	go func() {
		if err := httpSrv.ListenAndServe(restAddress); err != nil {
			logger.Error(err)
		}
	}()

	srvLock.Unlock()
}

func socketConfiguration(cfg interface{}) structure.SocketConfiguration {
	appConfig := cfg.(*conf.Configuration)
	return structure.SocketConfiguration{
		Host:   appConfig.ConfigServiceAddress.IP,
		Port:   appConfig.ConfigServiceAddress.Port,
		Secure: false,
		UrlParams: map[string]string{
			"module_name":   appConfig.ModuleName,
			"instance_uuid": appConfig.InstanceUuid,
		},
	}
}

func onShutdown(_ context.Context, _ os.Signal) {
	_ = httpSrv.Shutdown()
	invoker.RouterClient.Close()
}

func routesData(localConfig interface{}) bootstrap.ModuleInfo {
	cfg := localConfig.(*conf.Configuration)
	return bootstrap.ModuleInfo{
		ModuleName:       cfg.ModuleName,
		ModuleVersion:    version,
		GrpcOuterAddress: cfg.HttpOuterAddress,
		Handlers:         []interface{}{},
	}
}
