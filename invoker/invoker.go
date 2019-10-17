package invoker

import (
	"github.com/integration-system/isp-lib/backend"
	"github.com/integration-system/isp-lib/structure"
	log "github.com/integration-system/isp-log"
	"google.golang.org/grpc"
	"isp-convert-service/conf"
	"isp-convert-service/log_code"
)

var (
	RouterClient = backend.NewRxGrpcClient(
		backend.WithDialOptions(
			grpc.WithInsecure(), grpc.WithBlock(),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(int(conf.DefaultMaxResponseBodySize))),
			grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(int(conf.DefaultMaxResponseBodySize))),
		),
		backend.WithDialingErrorHandler(func(err error) {
			log.Errorf(log_code.ErrorRouterClientDialing, "router dialing err: %v", err)
		}),
	)
)

func HandleRoutesAddresses(list []structure.AddressConfiguration) bool {
	if RouterClient.ReceiveAddressList(list) {
		return true
	}
	return false
}
