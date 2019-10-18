package log_code

const (
	InfoOnLocalConfigLoad                      = 601
	WarnCreateRestServerHttpSrvShutdown        = 602
	ErrorCreateRestServerHttpSrvListenAndServe = 603
	WarnRequestHandler                         = 604 //metadata: {"typeData":"", "method":""}
	WarnConvertErrorDataMarshalResponse        = 605
	ErrorRouterClientDialing                   = 606
	WarnJournalCouldNotWriteToFile             = 607
	WarnJournalClientDialing                   = 608
)
