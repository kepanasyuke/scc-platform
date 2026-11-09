package provision

import (
	"github.com/scc-platform/shared/fn"
	"sync"
	"time"

	"github.com/scc-platform/shared/lists"
)

type Service struct {
	mutex     sync.RWMutex
	templates map[string]Template
	devices   map[string]Device
	order     []string
	accounts  map[string]Account
	server    string
}

func NewService(templates map[string]Template, devices []Device, accounts map[string]Account, server string) *Service {
	return &Service{
		templates: templates,
		devices:   indexDevices(devices),
		accounts:  accounts,
		order:     extractMACs(devices),
		server:    server,
	}
}

func indexDevices(devices []Device) map[string]Device {
	index := make(map[string]Device, len(devices))
	for _, device := range devices {
		index[device.MAC] = device
	}
	return index
}

func extractMACs(devices []Device) []string {
	return lists.Map(devices, func(device Device) string { return device.MAC })
}

func (s *Service) Devices() []Device {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return lists.Map(s.order, s.deviceOf)
}

func (s *Service) deviceOf(mac string) Device {
	return s.devices[mac]
}

func (s *Service) Register(mac, vendor, model, extension, location string) (Device, error) {
	normalized, err := NormalizeMAC(mac)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.registerLocked(normalized, err, Device{Vendor: vendor, Model: model, Extension: extension, Location: location})
}

func (s *Service) registerLocked(mac string, err error, device Device) (Device, error) {
	saved := device.withMAC(mac, s.lookup(mac))
	savedErr := registrationErr(err, s.templateExists(device), s.extensionFree(device.Extension))
	s.devices[mac] = saved
	s.order = appendUnique(s.order, mac, savedErr == nil)
	return saved, savedErr
}

func (d Device) withMAC(mac string, existing Device) Device {
	next := d
	next.MAC = mac
	next.UpdatedAt = time.Now().UTC()
	next.Firmware = chooseText(existing.Firmware, "2.0.0.0")
	next.Provisioned = true
	return next
}

func chooseText(value, fallback string) string {
	return fn.Choose(value == "", fallback, value)
}

func registrationErr(macErr error, known bool, free bool) error {
	switch {
	case macErr != nil:
		return macErr
	case !known:
		return ErrUnknownTemplate
	case !free:
		return ErrAlreadyBound
	}
	return nil
}

func (s *Service) templateExists(device Device) bool {
	_, known := s.templates[device.Vendor+"/"+device.Model]
	return known
}

func (s *Service) lookup(mac string) Device {
	existing, known := s.devices[mac]
	return existingOf(known, existing)
}

func existingOf(known bool, existing Device) Device {
	result := Device{}
	if known {
		result = existing
	}
	return result
}

func (s *Service) extensionFree(extension string) bool {
	busy := lists.Filter(s.Devices(), func(device Device) bool { return device.Extension == extension })
	return len(busy) == 0
}

func appendUnique(items []string, item string, ok bool) []string {
	exists := lists.Filter(items, func(current string) bool { return current == item })
	return fn.Choose(ok && len(exists) == 0, append(items, item), items)
}

func (s *Service) Config(mac string) (string, Device, error) {
	device, err := s.find(mac)
	account := s.accountOf(device.Extension)
	return s.render(device, account), device, err
}

func (s *Service) find(mac string) (Device, error) {
	normalized, normalErr := NormalizeMAC(mac)
	device, known := s.devices[normalized]
	return device, chooseErr(normalErr, known)
}

func chooseErr(normalErr error, known bool) error {
	switch {
	case normalErr != nil:
		return normalErr
	case !known:
		return ErrUnknownDevice
	}
	return nil
}

func (s *Service) accountOf(extension string) Account {
	account, known := s.accounts[extension]
	return accountOr(known, account, extension)
}

func accountOr(known bool, account Account, extension string) Account {
	result := Account{Extension: extension, Domain: "sip.mti.ru", Codec: "PCMU", Outbound: "sip.mti.ru:5060"}
	if known {
		result = account
	}
	return result
}

func (s *Service) render(device Device, account Account) string {
	template := s.templates[device.Vendor+"/"+device.Model]
	render := defaultRender(template)
	return render(device, account, s.server)
}

func defaultRender(template Template) func(Device, Account, string) string {
	return fn.Choose(template.Render == nil, YealinkXML, template.Render)
}
