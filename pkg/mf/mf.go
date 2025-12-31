package mf

import (
	"encoding/hex"
	"fmt"

	"git.quad4.io/Networks/Reticulum-Go/pkg/destination"
	"git.quad4.io/Networks/Reticulum-Go/pkg/identity"
	"git.quad4.io/Networks/Reticulum-Go/pkg/packet"
	"git.quad4.io/Networks/Reticulum-Go/pkg/transport"
)

// Message represents a basic chat message in the Reticulum-Go WASM format.
type Message struct {
	SenderHash []byte
	Text       string
}

// NewMessage creates a new Message with validation.
func NewMessage(senderHash []byte, text string) (*Message, error) {
	m := &Message{
		SenderHash: senderHash,
		Text:       text,
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return m, nil
}

// NewMessageFromHex creates a new Message from a hex-encoded sender hash.
func NewMessageFromHex(senderHashHex string, text string) (*Message, error) {
	hash, err := hex.DecodeString(senderHashHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hash: %w", err)
	}
	return NewMessage(hash, text)
}

// Validate checks if the message is valid.
func (m *Message) Validate() error {
	if len(m.SenderHash) != SenderHashLength {
		return fmt.Errorf(errFmtExpected, ErrInvalidHashLength, SenderHashLength, len(m.SenderHash))
	}
	if len(m.Text) > MaxMessageSize {
		return fmt.Errorf("%w: max %d, got %d", ErrMessageTooLong, MaxMessageSize, len(m.Text))
	}
	return nil
}

// Len returns the length of the packed message in bytes.
func (m *Message) Len() int {
	return SenderHashLength + len(m.Text)
}

// FormatSenderHash returns the hexadecimal representation of the sender hash.
func (m *Message) FormatSenderHash() string {
	return hex.EncodeToString(m.SenderHash)
}

// String returns a string representation of the message for debugging.
func (m *Message) String() string {
	return fmt.Sprintf("Message{Sender: %s, Text: %q}", m.FormatSenderHash(), m.Text)
}

// Equal reports whether m and other are equal.
func (m *Message) Equal(other *Message) bool {
	if other == nil {
		return false
	}
	if m.Text != other.Text {
		return false
	}
	if len(m.SenderHash) != len(other.SenderHash) {
		return false
	}
	for i := range m.SenderHash {
		if m.SenderHash[i] != other.SenderHash[i] {
			return false
		}
	}
	return true
}

// Pack serializes the message into a byte slice.
// The format is: [16 bytes sender hash][text payload]
func (m *Message) Pack() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	payload := make([]byte, m.Len())
	copy(payload[:SenderHashLength], m.SenderHash)
	copy(payload[SenderHashLength:], []byte(m.Text))
	return payload, nil
}

// Unpack parses a byte slice into a Message.
func Unpack(data []byte) (*Message, error) {
	if len(data) < SenderHashLength {
		return nil, fmt.Errorf("%w: minimum %d bytes for sender hash", ErrMessageTooShort, SenderHashLength)
	}

	senderHash := make([]byte, SenderHashLength)
	copy(senderHash, data[:SenderHashLength])
	text := string(data[SenderHashLength:])

	return NewMessage(senderHash, text)
}

// Peer represents a discovered peer in the network.
type Peer struct {
	Hash    []byte
	AppData string
}

// Validate checks if the peer is valid.
func (p *Peer) Validate() error {
	if len(p.Hash) != SenderHashLength {
		return fmt.Errorf(errFmtExpected, ErrInvalidHashLength, SenderHashLength, len(p.Hash))
	}
	return nil
}

// FormatHash returns the hexadecimal representation of the peer's hash.
func (p *Peer) FormatHash() string {
	return hex.EncodeToString(p.Hash)
}

// String returns a string representation of the peer for debugging.
func (p *Peer) String() string {
	return fmt.Sprintf("Peer{Hash: %s, AppData: %q}", p.FormatHash(), p.AppData)
}

// Messenger handles sending and receiving MF messages over Reticulum.
type Messenger struct {
	transport *transport.Transport
	dest      *destination.Destination
}

// NewMessenger creates a new Messenger instance.
func NewMessenger(t *transport.Transport, d *destination.Destination) *Messenger {
	return &Messenger{
		transport: t,
		dest:      d,
	}
}

// GetDestinationHash returns the local destination hash.
func (m *Messenger) GetDestinationHash() []byte {
	return m.dest.GetHash()
}

// GetDestination returns the internal Reticulum destination.
func (m *Messenger) GetDestination() *destination.Destination {
	return m.dest
}

// SendMessage sends a text message to a specific destination hash.
func (m *Messenger) SendMessage(destHash []byte, text string) error {
	if len(destHash) != SenderHashLength {
		return fmt.Errorf("invalid destination hash: %w", ErrInvalidHashLength)
	}

	remoteIdentity, err := identity.Recall(destHash)
	if err != nil {
		return fmt.Errorf("identity not found: %w", err)
	}

	targetDest, err := destination.FromHash(destHash, remoteIdentity, destination.SINGLE, m.transport)
	if err != nil {
		return fmt.Errorf("failed to create target destination: %w", err)
	}

	msg, err := NewMessage(m.GetDestinationHash(), text)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	payload, err := msg.Pack()
	if err != nil {
		return fmt.Errorf("failed to pack message: %w", err)
	}

	encrypted, err := targetDest.Encrypt(payload)
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
	pkt.DestinationHash = destHash

	if err := pkt.Pack(); err != nil {
		return fmt.Errorf("packet packing failed: %w", err)
	}

	if err := m.transport.SendPacket(pkt); err != nil {
		return fmt.Errorf("packet sending failed: %w", err)
	}

	return nil
}

// SenderHashFromHex decodes a hex string to a sender hash byte slice.
func SenderHashFromHex(s string) ([]byte, error) {
	hash, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if len(hash) != SenderHashLength {
		return nil, fmt.Errorf(errFmtExpected, ErrInvalidHashLength, SenderHashLength, len(hash))
	}
	return hash, nil
}

// ValidateSenderHash checks if the sender hash is valid.
func ValidateSenderHash(hash []byte) error {
	if len(hash) != SenderHashLength {
		return fmt.Errorf(errFmtExpected, ErrInvalidHashLength, SenderHashLength, len(hash))
	}
	return nil
}
