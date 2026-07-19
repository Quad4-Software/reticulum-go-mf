// SPDX-License-Identifier: 0BSD
package rrc

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestAdversarial_ForwardedSenderIsAuthenticatedPeer(t *testing.T) {
	if testing.Short() {
		t.Skip("adversarial mesh skipped in -short")
	}

	m := newTestMesh(t, 42810, HubConfig{
		Name: "adv-hub",
		Limits: HubLimits{
			MaxNickBytes:           8,
			MaxMsgBodyBytes:        64,
			RateLimitMsgsPerMinute: 120,
		},
	})

	got := make(chan *Envelope, 1)
	joinedA := make(chan struct{}, 1)
	joinedB := make(chan struct{}, 1)

	a := dialMeshClient(t, m, 'A', ClientConfig{
		Nick: "a",
		Handlers: ClientHandlers{
			OnJoined: func(room string, _ [][]byte, _ *Envelope) {
				if room == "#adv" {
					select {
					case joinedA <- struct{}{}:
					default:
					}
				}
			},
		},
	})
	b := dialMeshClient(t, m, 'B', ClientConfig{
		Nick: "b",
		Handlers: ClientHandlers{
			OnJoined: func(room string, _ [][]byte, _ *Envelope) {
				if room == "#adv" {
					select {
					case joinedB <- struct{}{}:
					default:
					}
				}
			},
			OnMsg: func(env *Envelope) {
				select {
				case got <- env:
				default:
				}
			},
		},
	})

	if err := a.Join("#adv"); err != nil {
		t.Fatal(err)
	}
	if err := b.Join("#adv"); err != nil {
		t.Fatal(err)
	}
	waitJoined(t, joinedA, "A")
	waitJoined(t, joinedB, "B")

	fakeSender := bytes.Repeat([]byte{0xee}, IdentityLength)
	env, err := NewEnvelope(TypeMsg, fakeSender)
	if err != nil {
		t.Fatal(err)
	}
	env.Room = "#adv"
	env.HasRoom = true
	env.Body = "spoofed"
	env.HasBody = true
	if err := a.sess.sendEnvelope(env); err != nil {
		t.Fatal(err)
	}

	select {
	case fwd := <-got:
		if bytes.Equal(fwd.Sender, fakeSender) {
			t.Fatal("hub forwarded client-claimed Sender (spoof succeeded)")
		}
		if !bytes.Equal(fwd.Sender, a.sender) {
			t.Fatalf("forwarded sender=%x want peer %x", fwd.Sender, a.sender)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for forwarded MSG")
	}
}

func TestAdversarial_PostWelcomeOversizedNickRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("adversarial mesh skipped in -short")
	}

	m := newTestMesh(t, 42820, HubConfig{
		Limits: HubLimits{MaxNickBytes: 4, MaxMsgBodyBytes: 64, RateLimitMsgsPerMinute: 60},
	})
	errCh := make(chan string, 1)
	joined := make(chan struct{}, 1)
	c := dialMeshClient(t, m, 'A', ClientConfig{
		Nick: "ok",
		Handlers: ClientHandlers{
			OnJoined: func(room string, _ [][]byte, _ *Envelope) {
				if room == "#n" {
					select {
					case joined <- struct{}{}:
					default:
					}
				}
			},
			OnError: func(env *Envelope) {
				if s, ok := BodyAsString(env.Body); ok {
					select {
					case errCh <- s:
					default:
					}
				}
			},
		},
	})
	if err := c.Join("#n"); err != nil {
		t.Fatal(err)
	}
	waitJoined(t, joined, "A")

	c.SetNick(strings.Repeat("Z", 32))
	if err := c.SendMsg("#n", "hi"); err != nil {
		t.Fatal(err)
	}

	select {
	case msg := <-errCh:
		if !strings.Contains(strings.ToLower(msg), "nick") {
			t.Fatalf("error body=%q", msg)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("expected nickname too long ERROR")
	}
}

func TestAdversarial_ControlOnlyNickNotForwardedRaw(t *testing.T) {
	if testing.Short() {
		t.Skip("adversarial mesh skipped in -short")
	}

	m := newTestMesh(t, 42830, HubConfig{
		Limits: HubLimits{MaxNickBytes: 16, MaxMsgBodyBytes: 64, RateLimitMsgsPerMinute: 60},
	})
	got := make(chan *Envelope, 1)
	joinedA := make(chan struct{}, 1)
	joinedB := make(chan struct{}, 1)
	a := dialMeshClient(t, m, 'A', ClientConfig{
		Handlers: ClientHandlers{
			OnJoined: func(room string, _ [][]byte, _ *Envelope) {
				if room == "#c" {
					select {
					case joinedA <- struct{}{}:
					default:
					}
				}
			},
		},
	})
	b := dialMeshClient(t, m, 'B', ClientConfig{
		Handlers: ClientHandlers{
			OnJoined: func(room string, _ [][]byte, _ *Envelope) {
				if room == "#c" {
					select {
					case joinedB <- struct{}{}:
					default:
					}
				}
			},
			OnMsg: func(env *Envelope) { got <- env },
		},
	})
	if err := a.Join("#c"); err != nil {
		t.Fatal(err)
	}
	if err := b.Join("#c"); err != nil {
		t.Fatal(err)
	}
	waitJoined(t, joinedA, "A")
	waitJoined(t, joinedB, "B")

	env := mustEnvelope(t, TypeMsg, a.sender)
	env.Room = "#c"
	env.HasRoom = true
	env.Body = "x"
	env.HasBody = true
	env.Nick = "\n\r\x00"
	env.HasNick = true
	if err := a.sess.sendEnvelope(env); err != nil {
		t.Fatal(err)
	}

	select {
	case fwd := <-got:
		if fwd.HasNick && strings.ContainsAny(fwd.Nick, "\n\r\x00") {
			t.Fatalf("raw control nick forwarded: %q", fwd.Nick)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for forwarded MSG")
	}
}

func TestAdversarial_NonTextBodyRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("adversarial mesh skipped in -short")
	}

	m := newTestMesh(t, 42840, HubConfig{
		Limits: HubLimits{MaxMsgBodyBytes: 32, RateLimitMsgsPerMinute: 60},
	})
	errCh := make(chan string, 1)
	joined := make(chan struct{}, 1)
	c := dialMeshClient(t, m, 'A', ClientConfig{
		Handlers: ClientHandlers{
			OnJoined: func(room string, _ [][]byte, _ *Envelope) {
				if room == "#b" {
					select {
					case joined <- struct{}{}:
					default:
					}
				}
			},
			OnError: func(env *Envelope) {
				if s, ok := BodyAsString(env.Body); ok {
					select {
					case errCh <- s:
					default:
					}
				}
			},
		},
	})
	if err := c.Join("#b"); err != nil {
		t.Fatal(err)
	}
	waitJoined(t, joined, "A")

	env := mustEnvelope(t, TypeMsg, c.sender)
	env.Room = "#b"
	env.HasRoom = true
	env.Body = map[uint64]any{0: true}
	env.HasBody = true
	if err := c.sess.sendEnvelope(env); err != nil {
		t.Fatal(err)
	}

	select {
	case msg := <-errCh:
		if !strings.Contains(strings.ToLower(msg), "body") {
			t.Fatalf("error=%q", msg)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("expected invalid/oversized body ERROR")
	}
}

func TestAdversarial_ApplyInboundNickUnit(t *testing.T) {
	h := &Hub{cfg: HubConfig{Limits: HubLimits{MaxNickBytes: 4}}}
	h.cfg.applyDefaults()
	p := &hubPeer{sess: &session{}}
	if err := h.applyInboundNick(p, "abcd"); err != nil {
		t.Fatal(err)
	}
	if err := h.applyInboundNick(p, "abcde"); err != ErrNickTooLong {
		t.Fatalf("err=%v", err)
	}
	if err := h.applyInboundNick(p, "\n\n"); err != nil {
		t.Fatal(err)
	}
	if p.sess.getNick() != "abcd" {
		t.Fatalf("nick mutated to %q", p.sess.getNick())
	}
}

func TestAdversarial_DropPeerIfIgnoresStale(t *testing.T) {
	h := &Hub{
		peers: map[string]*hubPeer{},
		rooms: map[string]map[string]struct{}{},
	}
	hash := bytes.Repeat([]byte{0x01}, IdentityLength)
	old := &hubPeer{peerHash: hash, rooms: map[string]struct{}{}}
	neu := &hubPeer{peerHash: hash, rooms: map[string]struct{}{}}
	key := peerKey(hash)
	h.peers[key] = neu
	h.dropPeerIf(old)
	if h.peers[key] != neu {
		t.Fatal("stale close removed replacement peer")
	}
	h.dropPeerIf(neu)
	if _, ok := h.peers[key]; ok {
		t.Fatal("current peer not dropped")
	}
}
