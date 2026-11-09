package provision

import (
	"errors"
	"fmt"
	"github.com/scc-platform/shared/fn"
	"strings"
	"time"
)

var (
	ErrUnknownDevice = errors.New("provision: unknown device")
	ErrBadMAC        = errors.New("provision: mac must be 12 hex chars")
	ErrAlreadyBound  = errors.New("provision: extension already bound to another device")
)

type Device struct {
	MAC         string    `json:"mac"`
	Vendor      string    `json:"vendor"`
	Model       string    `json:"model"`
	Extension   string    `json:"extension"`
	Location    string    `json:"location"`
	Firmware    string    `json:"firmware"`
	UpdatedAt   time.Time `json:"updated_at"`
	Provisioned bool      `json:"provisioned"`
}

type Account struct {
	Extension string `json:"extension"`
	Password  string `json:"password"`
	Domain    string `json:"domain"`
	Outbound  string `json:"outbound_proxy"`
	Codec     string `json:"codec"`
}

type Template struct {
	Vendor string
	Model  string
	Render func(device Device, account Account, server string) string
}

func NormalizeMAC(mac string) (string, error) {
	clean := strings.ToLower(strings.NewReplacer(":", "", "-", "", ".", "").Replace(strings.TrimSpace(mac)))
	return clean, macErr(clean)
}

func macErr(clean string) error {
	valid := len(clean) == 12 && isHex(clean)
	return fn.Choose(valid, nil, ErrBadMAC)
}

func isHex(value string) bool {
	valid := true
	for _, symbol := range value {
		valid = valid && isHexSymbol(symbol)
	}
	return valid
}

func isHexSymbol(symbol rune) bool {
	digit := symbol >= '0' && symbol <= '9'
	letter := symbol >= 'a' && symbol <= 'f'
	return digit || letter
}

func YealinkXML(device Device, account Account, server string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<y000000000046.cfg>
  <account>
    <Enable>1</Enable>
    <Label>%s</Label>
    <DisplayName>%s</DisplayName>
    <UserName>%s</UserName>
    <AuthName>%s</AuthName>
    <Password>%s</Password>
    <SIPServerHost>%s</SIPServerHost>
    <SIPServerPort>5060</SIPServerPort>
    <OutboundProxy>%s</OutboundProxy>
    <Codec>%s</Codec>
  </account>
  <phone>
    <Location>%s</Location>
    <Firmware>%s</Firmware>
    <ProvisioningServer>%s</ProvisioningServer>
  </phone>
</y000000000046.cfg>`, account.Extension, device.Location, account.Extension, account.Extension,
		account.Password, server, account.Outbound, account.Codec, device.Location, device.Firmware, server)
}

func GrandstreamXML(device Device, account Account, server string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<gs_provision version="1">
  <config version="1">
    <P271>%s</P271>
    <P270>%s</P270>
    <P35>%s</P35>
    <P36>%s</P36>
    <P34>%s</P34>
    <P47>%s</P47>
    <P48>%s</P48>
    <P196>%s</P196>
    <P140>%s</P140>
    <P192>%s</P192>
  </config>
</gs_provision>`, account.Extension, account.Password, account.Domain, server, account.Outbound,
		device.MAC, device.Location, device.Firmware, account.Codec, server)
}

func Templates() map[string]Template {
	return map[string]Template{
		"yealink/t33g":      {Vendor: "yealink", Model: "t33g", Render: YealinkXML},
		"yealink/t46u":      {Vendor: "yealink", Model: "t46u", Render: YealinkXML},
		"grandstream/grp26": {Vendor: "grandstream", Model: "grp2602", Render: GrandstreamXML},
		"grandstream/gxp16": {Vendor: "grandstream", Model: "gxp1625", Render: GrandstreamXML},
	}
}
