package provision

import "errors"

var ErrUnknownTemplate = errors.New("provision: vendor/model template not found")

func DefaultDevices() []Device {
	return []Device{
		{MAC: "805e0c1a2b3c", Vendor: "yealink", Model: "t33g", Extension: "101", Location: "приёмная, корпус А", Firmware: "124.86.0.40", Provisioned: true},
		{MAC: "805e0c1a2b4d", Vendor: "yealink", Model: "t46u", Extension: "104", Location: "приёмная, корпус А", Firmware: "124.86.0.40", Provisioned: true},
		{MAC: "000b8212aa01", Vendor: "grandstream", Model: "grp2602", Extension: "107", Location: "поддержка, 3 этаж", Firmware: "1.0.3.14", Provisioned: true},
		{MAC: "000b8212aa02", Vendor: "grandstream", Model: "gxp1625", Extension: "118", Location: "экзаменационный центр", Firmware: "1.0.3.14", Provisioned: false},
	}
}

func DefaultAccounts() map[string]Account {
	return map[string]Account{
		"101": {Extension: "101", Password: "e4Kq9-Lm2", Domain: "sip.mti.ru", Outbound: "sip.mti.ru:5060", Codec: "PCMU"},
		"104": {Extension: "104", Password: "Rr7-tP4a", Domain: "sip.mti.ru", Outbound: "sip.mti.ru:5060", Codec: "OPUS"},
		"107": {Extension: "107", Password: "Nw3-zX8b", Domain: "sip.mti.ru", Outbound: "sip.mti.ru:5060", Codec: "PCMA"},
		"118": {Extension: "118", Password: "Qp1-vB6c", Domain: "sip.mti.ru", Outbound: "sip.mti.ru:5060", Codec: "PCMU"},
	}
}
