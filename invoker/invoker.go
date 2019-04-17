package invoker

import (
	"github.com/integration-system/isp-lib/backend"
	"github.com/integration-system/isp-lib/logger"
	"github.com/integration-system/isp-lib/structure"
	"google.golang.org/grpc"
)

var (
	RouterClient = backend.NewRxGrpcClient(
		backend.WithDialOptions(grpc.WithInsecure(), grpc.WithBlock()),
		backend.WithDialingErrorHandler(func(err error) {
			logger.Warnf("router dialing err: %v", err)
		}),
	)
)

func HandleRoutesAddresses(list []structure.AddressConfiguration) bool {
	if RouterClient.ReceiveAddressList(list) {
		logger.Infof("Successfully connected to routes: %v", list)
		return true
	}
	return false
}
