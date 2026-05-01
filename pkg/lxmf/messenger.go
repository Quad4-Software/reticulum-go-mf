// SPDX-License-Identifier: 0BSD
package lxmf

import (
	"errors"
	"fmt"
	"sync"

	"git.quad4.io/Networks/Reticulum-Go/pkg/common"
	"git.quad4.io/Networks/Reticulum-Go/pkg/destination"
	"git.quad4.io/Networks/Reticulum-Go/pkg/identity"
	"git.quad4.io/Networks/Reticulum-Go/pkg/packet"
	"git.quad4.io/Networks/Reticulum-Go/pkg/transport"
)

// MessageHandler receives one unpacked inbound LXMessage per packet (signature state is on msg).
type MessageHandler func(msg *LXMessage, iface common.NetworkInterface)

// Messenger sends and receives LXMF over a Transport. The destination must be inbound Single.
type Messenger struct {
	transport *transport.Transport
	dest      *destination.Destination

	mu       sync.RWMutex
	handler  MessageHandler
	resolver SourceResolver
}

// NewMessenger registers d's packet callback for inbound LXMF. Use NewDeliveryDestination for lxmf.delivery naming.
func NewMessenger(t *transport.Transport, d *destination.Destination) *Messenger {
	m := &Messenger{
		transport: t,
		dest:      d,
		resolver:  RecallSource,
	}
	d.SetPacketCallback(m.onPacket)
	return m
}

// NewDeliveryDestination returns the inbound lxmf.delivery destination for id.
func NewDeliveryDestination(id *identity.Identity, t *transport.Transport) (*destination.Destination, error) {
	return destination.New(id, destination.In, destination.Single, AppName, t, "delivery")
}

// NewDeliveryMessenger is NewDeliveryDestination plus NewMessenger.
func NewDeliveryMessenger(id *identity.Identity, t *transport.Transport) (*Messenger, error) {
	dest, err := NewDeliveryDestination(id, t)
	if err != nil {
		return nil, err
	}
	return NewMessenger(t, dest), nil
}

// Destination returns the local RNS destination.
func (m *Messenger) Destination() *destination.Destination {
	return m.dest
}

// DestinationHash returns the local destination hash.
func (m *Messenger) DestinationHash() []byte {
	return m.dest.GetHash()
}

// SetMessageHandler sets the inbound callback; nil disables delivery.
func (m *Messenger) SetMessageHandler(h MessageHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handler = h
}

// SetSourceResolver sets signature verification lookup; default is RecallSource.
func (m *Messenger) SetSourceResolver(r SourceResolver) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r == nil {
		m.resolver = RecallSource
		return
	}
	m.resolver = r
}

// Compose builds an outbound message from this destination as source.
func (m *Messenger) Compose(destinationHash []byte, title, content string, fields map[byte]any) (*LXMessage, error) {
	return NewMessage(destinationHash, m.DestinationHash(), []byte(title), []byte(content), fields)
}

// Send packs, signs, and sends one opportunistic encrypted packet. The peer must be in identity.Recall.
func (m *Messenger) Send(msg *LXMessage) error {
	if msg == nil {
		return errors.New("lxmf: nil message")
	}
	if len(msg.DestinationHash) != DestinationLength {
		return fmt.Errorf("destination: %w", ErrInvalidHashLength)
	}

	remoteIdentity, err := identity.Recall(msg.DestinationHash)
	if err != nil {
		return fmt.Errorf("destination identity not found: %w", err)
	}
	if remoteIdentity == nil {
		return ErrDestinationUnknown
	}

	target, err := destination.FromHash(msg.DestinationHash, remoteIdentity, destination.Single, m.transport)
	if err != nil {
		return fmt.Errorf("create target destination: %w", err)
	}

	signer := m.dest.GetIdentity()
	if signer == nil {
		return errors.New("lxmf: local destination has no identity")
	}

	if _, err := msg.Pack(signer); err != nil {
		return err
	}

	innerPayload, err := msg.EncryptedPayload()
	if err != nil {
		return err
	}

	encrypted, err := target.Encrypt(innerPayload)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	pkt := packet.NewPacket(
		packet.DestinationSingle,
		encrypted,
		packet.PacketTypeData,
		packet.ContextNone,
		packet.PropagationBroadcast,
		packet.HeaderType1,
		nil,
		true,
		packet.FlagUnset,
	)
	pkt.DestinationHash = append([]byte(nil), msg.DestinationHash...)

	if err := pkt.Pack(); err != nil {
		return fmt.Errorf("packet packing failed: %w", err)
	}

	if err := m.transport.SendPacket(pkt); err != nil {
		return fmt.Errorf("packet sending failed: %w", err)
	}

	msg.Method = MethodOpportunistic
	msg.Representation = RepresentationPacket
	msg.State = StateSent
	return nil
}

// SendText composes and sends a text-only message.
func (m *Messenger) SendText(destinationHash []byte, title, content string) (*LXMessage, error) {
	msg, err := m.Compose(destinationHash, title, content, nil)
	if err != nil {
		return nil, err
	}
	if err := m.Send(msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (m *Messenger) onPacket(plaintext []byte, iface common.NetworkInterface) {
	m.mu.RLock()
	handler := m.handler
	resolver := m.resolver
	m.mu.RUnlock()

	if handler == nil {
		return
	}

	if len(plaintext) < DestinationLength+SignatureLength {
		return
	}

	msg, err := UnpackFromBytes(m.DestinationHash(), plaintext, resolver)
	if err != nil && msg == nil {
		Warning("inbound lxmf unpack failed", "error", err, "plaintext_len", len(plaintext))
		return
	}
	if err != nil {
		Debug("inbound lxmf unpack completed with error", "error", err,
			"signature_validated", msg.SignatureValidated, "unverified_reason", msg.UnverifiedReason)
	}

	handler(msg, iface)
}
