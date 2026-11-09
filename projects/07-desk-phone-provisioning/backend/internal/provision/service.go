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
	return indexInto(make(map[string]Device, len(devices)), devices)
}

func indexInto(index map[string]Device, devices []Device) map[string]Device {
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
	return s.registerDevice(mac, Device{Vendor: vendor, Model: model, Extension: extension, Location: location})
}

func (s *Service) registerDevice(mac string, device Device) (Device, error) {
	normalized, err := NormalizeMAC(mac)
	return s.registerNormalized(normalized, err, device)
}

func (s *Service) registerNormalized(mac string, err error, device Device) (Device, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.registerLocked(mac, err, device)
}

func (s *Service) registerLocked(mac string, err error, device Device) (Device, error) {
	return s.save(device.withMAC(mac, s.lookup(mac)), registrationErr(err, s.templateExists(device), s.extensionFree(device.Extension)))
}

func (s *Service) save(saved Device, savedErr error) (Device, error) {
	s.remember(saved, savedErr == nil)
	return saved, savedErr
}

func (s *Service) remember(saved Device, ok bool) {
	s.devices[saved.MAC] = saved
	s.order = appendUnique(s.order, saved.MAC, ok)
}

func (d Device) withMAC(mac string, existing Device) Device {
	return fn.With(d, func(next *Device) {
		next.MAC = mac
		next.UpdatedAt = time.Now().UTC()
		next.Firmware = chooseText(existing.Firmware, "2.0.0.0")
		next.Provisioned = true
	})
}

func chooseText(value, fallback string) string {
	return fn.Choose(value == "", fallback, value)
}

func registrationErr(macErr error, known bool, free bool) error {
	return fn.Choose(macErr != nil, macErr, fn.Choose(!known, ErrUnknownTemplate, fn.Choose(!free, ErrAlreadyBound, nil)))
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
	return fn.Choose(known, existing, Device{})
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
	return s.render(device, s.accountOf(device.Extension)), device, err
}

func (s *Service) find(mac string) (Device, error) {
	normalized, normalErr := NormalizeMAC(mac)
	return s.findDevice(normalized, normalErr)
}

func (s *Service) findDevice(normalized string, normalErr error) (Device, error) {
	device, known := s.devices[normalized]
	return device, chooseErr(normalErr, known)
}

func chooseErr(normalErr error, known bool) error {
	return fn.Choose(normalErr != nil, normalErr, fn.Choose(known, nil, ErrUnknownDevice))
}

func (s *Service) accountOf(extension string) Account {
	account, known := s.accounts[extension]
	return accountOr(known, account, extension)
}

func accountOr(known bool, account Account, extension string) Account {
	blank := Account{Extension: extension, Domain: "sip.mti.ru", Codec: "PCMU", Outbound: "sip.mti.ru:5060"}
	return fn.Choose(known, account, blank)
}

func (s *Service) render(device Device, account Account) string {
	return renderOf(s.templateOf(device), device, account, s.server)
}

func (s *Service) templateOf(device Device) Template {
	return s.templates[device.Vendor+"/"+device.Model]
}

func renderOf(template Template, device Device, account Account, server string) string {
	return defaultRender(template)(device, account, server)
}

func defaultRender(template Template) func(Device, Account, string) string {
	return fn.Choose(template.Render == nil, YealinkXML, template.Render)
}
