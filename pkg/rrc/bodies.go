// SPDX-License-Identifier: 0BSD
package rrc

import "fmt"

// HelloBody is the optional HELLO body map (3-RRC).
type HelloBody struct {
	ClientName    string
	ClientVersion string
	Capabilities  map[uint64]any
	HasName       bool
	HasVersion    bool
	HasCaps       bool
}

// ToMap converts HelloBody to a CBOR-friendly map with uint keys.
func (h *HelloBody) ToMap() map[uint64]any {
	if h == nil {
		return nil
	}
	m := make(map[uint64]any)
	if h.HasName {
		m[HelloKeyClientName] = h.ClientName
	}
	if h.HasVersion {
		m[HelloKeyClientVersion] = h.ClientVersion
	}
	if h.HasCaps && h.Capabilities != nil {
		m[HelloKeyCapabilities] = h.Capabilities
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// ParseHelloBody extracts a HelloBody from an envelope body value.
func ParseHelloBody(body any) (*HelloBody, error) {
	if body == nil {
		return &HelloBody{}, nil
	}
	raw, ok := body.(map[uint64]any)
	if !ok {
		// cbor may decode as map[interface{}]interface{} depending on mode
		raw2, err := coerceUintMap(body)
		if err != nil {
			return nil, fmt.Errorf("%w: hello body", ErrInvalidEnvelope)
		}
		raw = raw2
	}
	h := &HelloBody{}
	if v, ok := raw[HelloKeyClientName]; ok {
		if s, ok := asString(v); ok {
			h.ClientName = s
			h.HasName = true
		}
	}
	if v, ok := raw[HelloKeyClientVersion]; ok {
		if s, ok := asString(v); ok {
			h.ClientVersion = s
			h.HasVersion = true
		}
	}
	if v, ok := raw[HelloKeyCapabilities]; ok {
		if caps, err := coerceUintMap(v); err == nil {
			h.Capabilities = caps
			h.HasCaps = true
		}
	}
	return h, nil
}

// HubLimits describes advisory hub limits in WELCOME body key 3.
type HubLimits struct {
	MaxNickBytes           uint64
	MaxRoomsPerSession     uint64
	MaxRoomNameBytes       uint64
	MaxMsgBodyBytes        uint64
	RateLimitMsgsPerMinute uint64
}

// ToMap encodes HubLimits as a CBOR map with numeric keys (rrcd order).
func (l HubLimits) ToMap() map[uint64]any {
	return map[uint64]any{
		LimitMaxNickBytes:           l.MaxNickBytes,
		LimitMaxRoomNameBytes:       l.MaxRoomNameBytes,
		LimitMaxMsgBodyBytes:        l.MaxMsgBodyBytes,
		LimitMaxRoomsPerSession:     l.MaxRoomsPerSession,
		LimitRateLimitMsgsPerMinute: l.RateLimitMsgsPerMinute,
	}
}

// ParseHubLimits reads HubLimits from a CBOR map value.
func ParseHubLimits(v any) (HubLimits, bool) {
	raw, err := coerceUintMap(v)
	if err != nil {
		return HubLimits{}, false
	}
	var l HubLimits
	if n, ok := asUint64(raw[LimitMaxNickBytes]); ok {
		l.MaxNickBytes = n
	}
	if n, ok := asUint64(raw[LimitMaxRoomNameBytes]); ok {
		l.MaxRoomNameBytes = n
	}
	if n, ok := asUint64(raw[LimitMaxMsgBodyBytes]); ok {
		l.MaxMsgBodyBytes = n
	}
	if n, ok := asUint64(raw[LimitMaxRoomsPerSession]); ok {
		l.MaxRoomsPerSession = n
	}
	if n, ok := asUint64(raw[LimitRateLimitMsgsPerMinute]); ok {
		l.RateLimitMsgsPerMinute = n
	}
	return l, true
}

// WelcomeBody is the optional WELCOME body map (3-RRC).
type WelcomeBody struct {
	HubName      string
	HubVersion   string
	Capabilities map[uint64]any
	Limits       HubLimits
	HasName      bool
	HasVersion   bool
	HasCaps      bool
	HasLimits    bool
}

// ToMap converts WelcomeBody to a CBOR-friendly map with uint keys.
func (w *WelcomeBody) ToMap() map[uint64]any {
	if w == nil {
		return nil
	}
	m := make(map[uint64]any)
	if w.HasName {
		m[WelcomeKeyHubName] = w.HubName
	}
	if w.HasVersion {
		m[WelcomeKeyHubVersion] = w.HubVersion
	}
	if w.HasCaps && w.Capabilities != nil {
		m[WelcomeKeyCapabilities] = w.Capabilities
	}
	if w.HasLimits {
		m[WelcomeKeyHubLimits] = w.Limits.ToMap()
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// ParseWelcomeBody extracts a WelcomeBody from an envelope body value.
func ParseWelcomeBody(body any) (*WelcomeBody, error) {
	if body == nil {
		return &WelcomeBody{}, nil
	}
	raw, err := coerceUintMap(body)
	if err != nil {
		return nil, fmt.Errorf("%w: welcome body", ErrInvalidEnvelope)
	}
	w := &WelcomeBody{}
	if v, ok := raw[WelcomeKeyHubName]; ok {
		if s, ok := asString(v); ok {
			w.HubName = s
			w.HasName = true
		}
	}
	if v, ok := raw[WelcomeKeyHubVersion]; ok {
		if s, ok := asString(v); ok {
			w.HubVersion = s
			w.HasVersion = true
		}
	}
	if v, ok := raw[WelcomeKeyCapabilities]; ok {
		if caps, err := coerceUintMap(v); err == nil {
			w.Capabilities = caps
			w.HasCaps = true
		}
	}
	if v, ok := raw[WelcomeKeyHubLimits]; ok {
		if lim, ok := ParseHubLimits(v); ok {
			w.Limits = lim
			w.HasLimits = true
		}
	}
	return w, nil
}

// ParseJoinedMembers extracts an advisory member list from a JOINED body.
// Accepts a CBOR array of 16-byte identities, or nil/empty.
func ParseJoinedMembers(body any) ([][]byte, error) {
	if body == nil {
		return nil, nil
	}
	arr, ok := body.([]any)
	if !ok {
		return nil, nil
	}
	out := make([][]byte, 0, len(arr))
	for _, item := range arr {
		b, ok := asBytes(item)
		if !ok || len(b) != IdentityLength {
			continue
		}
		out = append(out, append([]byte(nil), b...))
	}
	return out, nil
}

// BodyAsString returns body as a text string when possible.
func BodyAsString(body any) (string, bool) {
	return asString(body)
}

func coerceUintMap(v any) (map[uint64]any, error) {
	switch m := v.(type) {
	case map[uint64]any:
		return m, nil
	case map[any]any:
		out := make(map[uint64]any, len(m))
		for k, val := range m {
			uk, ok := asUint64(k)
			if !ok {
				continue
			}
			out[uk] = val
		}
		return out, nil
	default:
		return nil, ErrInvalidEnvelope
	}
}
