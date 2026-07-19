// SPDX-License-Identifier: 0BSD
package rrc

import (
	"bytes"
	"testing"
)

func TestNormalizeRoom(t *testing.T) {
	if got := NormalizeRoom("  #Lobby "); got != "#lobby" {
		t.Fatalf("NormalizeRoom = %q", got)
	}
}

func TestSanitizeNick(t *testing.T) {
	if got := SanitizeNick("  alice\n"); got != "alice" {
		t.Fatalf("SanitizeNick = %q", got)
	}
}

func TestEnvelopeRoundTripMSG(t *testing.T) {
	sender := bytes.Repeat([]byte{0x9c}, IdentityLength)
	env, err := NewEnvelope(TypeMsg, sender)
	if err != nil {
		t.Fatal(err)
	}
	env.Room = "#lobby"
	env.HasRoom = true
	env.Body = "Hello, world!"
	env.HasBody = true
	env.Nick = "alice"
	env.HasNick = true
	env.Timestamp = 1737849600000
	env.MsgID = []byte{0x7a, 0x3f, 0x8e, 0x12, 0x45, 0xc9, 0xa1, 0x6d}

	raw, err := env.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < FixedEnvelopeMin {
		t.Fatalf("encoded len %d < fixed min %d", len(raw), FixedEnvelopeMin)
	}

	got, err := UnmarshalEnvelope(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != TypeMsg || got.Version != ProtocolVersion {
		t.Fatalf("type/version = %d/%d", got.Type, got.Version)
	}
	if !bytes.Equal(got.Sender, sender) {
		t.Fatal("sender mismatch")
	}
	if got.Room != "#lobby" || !got.HasRoom {
		t.Fatalf("room = %q has=%v", got.Room, got.HasRoom)
	}
	s, ok := BodyAsString(got.Body)
	if !ok || s != "Hello, world!" {
		t.Fatalf("body = %#v", got.Body)
	}
	if got.Nick != "alice" || !got.HasNick {
		t.Fatalf("nick = %q", got.Nick)
	}
}

func TestEnvelopeIgnoresUnknownKeys(t *testing.T) {
	sender := bytes.Repeat([]byte{0x11}, IdentityLength)
	env, err := NewEnvelope(TypePing, sender)
	if err != nil {
		t.Fatal(err)
	}
	raw2, err := mustMarshalWithExtra(*env, 99, "future")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := UnmarshalEnvelope(raw2)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Type != TypePing {
		t.Fatalf("type = %d", parsed.Type)
	}
}

func TestEnvelopeWrongVersion(t *testing.T) {
	sender := bytes.Repeat([]byte{0x22}, IdentityLength)
	env, err := NewEnvelope(TypeHello, sender)
	if err != nil {
		t.Fatal(err)
	}
	env.Version = 99
	_, err = env.Marshal()
	if err != ErrWrongVersion {
		t.Fatalf("marshal err = %v", err)
	}
	env.Version = ProtocolVersion
	raw, err := env.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	// Tamper version in map
	raw2, err := mustMarshalWithVersion(env, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = UnmarshalEnvelope(raw2)
	if err != ErrWrongVersion {
		t.Fatalf("unmarshal err = %v", err)
	}
	_ = raw
}

func TestEnvelopeMissingFields(t *testing.T) {
	_, err := UnmarshalEnvelope(nil)
	if err != ErrInvalidEnvelope {
		t.Fatalf("nil data err = %v", err)
	}
}

func TestHelloWelcomeBodies(t *testing.T) {
	h := &HelloBody{ClientName: "go-rrc", HasName: true, ClientVersion: "0.1.0", HasVersion: true}
	sender := bytes.Repeat([]byte{0x33}, IdentityLength)
	env, err := NewEnvelope(TypeHello, sender)
	if err != nil {
		t.Fatal(err)
	}
	env.Body = h.ToMap()
	env.HasBody = true
	raw, err := env.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	got, err := UnmarshalEnvelope(raw)
	if err != nil {
		t.Fatal(err)
	}
	hb, err := ParseHelloBody(got.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !hb.HasName || hb.ClientName != "go-rrc" {
		t.Fatalf("hello body = %+v", hb)
	}

	w := &WelcomeBody{
		HubName: "ExampleHub", HasName: true,
		HubVersion: "0.1.0", HasVersion: true,
		Limits: HubLimits{
			MaxNickBytes: 32, MaxRoomsPerSession: 32,
			MaxRoomNameBytes: 64, MaxMsgBodyBytes: 350,
			RateLimitMsgsPerMinute: 60,
		},
		HasLimits: true,
	}
	env2, err := NewEnvelope(TypeWelcome, sender)
	if err != nil {
		t.Fatal(err)
	}
	env2.Body = w.ToMap()
	env2.HasBody = true
	raw2, err := env2.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	got2, err := UnmarshalEnvelope(raw2)
	if err != nil {
		t.Fatal(err)
	}
	wb, err := ParseWelcomeBody(got2.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !wb.HasName || wb.HubName != "ExampleHub" || !wb.HasLimits || wb.Limits.MaxMsgBodyBytes != 350 {
		t.Fatalf("welcome body = %+v", wb)
	}
}

func TestJoinedMembers(t *testing.T) {
	a := bytes.Repeat([]byte{0x01}, IdentityLength)
	b := bytes.Repeat([]byte{0x02}, IdentityLength)
	members, err := ParseJoinedMembers([]any{a, b, "skip"})
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Fatalf("len = %d", len(members))
	}
}
