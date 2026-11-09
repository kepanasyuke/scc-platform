package sip

import "errors"

var ErrUnknownTrunk = errors.New("sip: unknown trunk")

func DefaultTrunks() []Trunk {
	return []Trunk{
		{Name: "mti-sip-01", Host: "10.10.20.11", Port: 5060, Transport: "udp", LocalAddr: "10.10.20.5:5060", Carrier: "Ростелеком", Channels: 42, Capacity: 120},
		{Name: "mti-sip-02", Host: "10.10.20.12", Port: 5060, Transport: "udp", LocalAddr: "10.10.20.5:5060", Carrier: "МТС", Channels: 118, Capacity: 120},
		{Name: "mti-sip-03", Host: "10.10.20.13", Port: 5061, Transport: "tls", LocalAddr: "10.10.20.5:5061", Carrier: "Мегафон", Channels: 17, Capacity: 60},
		{Name: "mti-out-01", Host: "10.10.30.21", Port: 5060, Transport: "udp", LocalAddr: "10.10.30.5:5060", Carrier: "Telfin", Channels: 6, Capacity: 90},
		{Name: "mti-webrtc-01", Host: "10.10.40.11", Port: 7443, Transport: "wss", LocalAddr: "10.10.40.5:7443", Carrier: "internal", Channels: 64, Capacity: 200},
	}
}
