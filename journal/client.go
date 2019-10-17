package journal

import (
	"github.com/integration-system/isp-journal/rx"
	"github.com/integration-system/isp-lib/backend"
	"github.com/integration-system/isp-lib/structure"
	log "github.com/integration-system/isp-log"
	"google.golang.org/grpc"
	"isp-convert-service/log_code"
)

var (
	journalServiceClient = backend.NewRxGrpcClient(
		backend.WithDialOptions(grpc.WithInsecure(), grpc.WithBlock()),
		backend.WithDialingErrorHandler(func(err error) {
			log.Warnf(log_code.WarnJournalClientDialing, "journal client dialing err: %v", err)
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
