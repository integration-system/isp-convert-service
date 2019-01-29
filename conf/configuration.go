package conf

import "github.com/integration-system/isp-lib/structure"

type Configuration struct {
	ConfigServiceAddress structure.AddressConfiguration `valid:"required~Required" json:"configServiceAddress"`
	ModuleName           string                         `valid:"required~Required" json:"moduleName"`
	InstanceUuid         string                         `valid:"required~Required" json:"instanceUuid"`
	HttpOuterAddress     structure.AddressConfiguration `valid:"required~Required" json:"httpOuterAddress"`
	HttpInnerAddress     structure.AddressConfiguration `valid:"required~Required" json:"httpInnerAddress"`
}
