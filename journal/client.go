package journal

import (
	"github.com/integration-system/isp-journal/rx"
	"github.com/integration-system/isp-lib/backend"
	"github.com/integration-system/isp-lib/logger"
	"github.com/integration-system/isp-lib/structure"
	"google.golang.org/grpc"
)

var (
	journalServiceClient = backend.NewRxGrpcClient(
		backend.WithDialOptions(grpc.WithInsecure(), grpc.WithBlock()),
		backend.WithDialingErrorHandler(func(err error) {
			logger.Warnf("journal client dialing err: %v", err)
		}),
	)
	Client = rx.NewDefaultRxJournal(journalServiceClient)
)

func ReceiveJournalServiceAddressList(list []structure.AddressConfiguration) bool {
	ok := journalServiceClient.ReceiveAddressList(list)
	if !ok {
		return false
	}

	go Client.CollectAndTransferExistedLogs()

	return true
}
