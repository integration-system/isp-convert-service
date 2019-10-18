package log_code

const (
	MdTypeData = "typeData"
	MdMethod   = "method"
)

var (
	TypeData = typeData{
		ProxyMultipart: "proxy_multipart",
		DownloadFile:   "download_file",
		MethodInvoke:   "api_invoke",
	}
)

type (
	typeData struct {
		ProxyMultipart string
		DownloadFile   string
		MethodInvoke   string
	}
)
