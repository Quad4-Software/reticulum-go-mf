// SPDX-License-Identifier: 0BSD
package rrc

// AppName is the Reticulum destination application name for RRC hubs.
const AppName = "rrc"

// HubAspect is the destination aspect for RRC hubs (rrc.hub).
const HubAspect = "hub"

// ProtocolVersion is the RRC on-wire protocol version for this implementation.
const ProtocolVersion uint64 = 1

// Fixed wire field sizes.
const (
	MessageIDLength  = 8
	IdentityLength   = 16
	FixedEnvelopeMin = 43
)

// Envelope field keys (3-RRC).
const (
	KeyVersion   uint64 = 0
	KeyType      uint64 = 1
	KeyMsgID     uint64 = 2
	KeyTimestamp uint64 = 3
	KeySender    uint64 = 4
	KeyRoom      uint64 = 5
	KeyBody      uint64 = 6
	KeyNick      uint64 = 7
)

// Message type assignments (3-RRC).
const (
	TypeHello   uint64 = 1
	TypeWelcome uint64 = 2
	TypeJoin    uint64 = 10
	TypeJoined  uint64 = 11
	TypePart    uint64 = 12
	TypeParted  uint64 = 13
	TypeMsg     uint64 = 20
	TypeNotice  uint64 = 21
	TypeAction  uint64 = 22
	TypePing    uint64 = 30
	TypePong    uint64 = 31
	TypeError   uint64 = 40
)

// HELLO body keys.
const (
	HelloKeyClientName    uint64 = 0
	HelloKeyClientVersion uint64 = 1
	HelloKeyCapabilities  uint64 = 2
)

// WELCOME body keys.
const (
	WelcomeKeyHubName      uint64 = 0
	WelcomeKeyHubVersion   uint64 = 1
	WelcomeKeyCapabilities uint64 = 2
	WelcomeKeyHubLimits    uint64 = 3
)

// HubLimits nested map keys (within WELCOME body key 3), matching rrcd.
const (
	LimitMaxNickBytes           uint64 = 0
	LimitMaxRoomNameBytes       uint64 = 1
	LimitMaxMsgBodyBytes        uint64 = 2
	LimitMaxRoomsPerSession     uint64 = 3
	LimitRateLimitMsgsPerMinute uint64 = 4
)

// Default hub limits used when HubConfig leaves a field at zero.
const (
	DefaultMaxNickBytes           = 32
	DefaultMaxRoomsPerSession     = 32
	DefaultMaxRoomNameBytes       = 64
	DefaultMaxMsgBodyBytes        = 350
	DefaultRateLimitMsgsPerMinute = 60
)
