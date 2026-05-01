# Reticulum-Go Message Formats Library

A message format for Reticulum-Go. The repository ships two
complementary packages:

- `pkg/mf` - the lightweight native MF format, optimised for size and
  for use cases where the wire layout is fully under your control.
- `pkg/lxmf` - a wire-compatible implementation of the LXMF protocol
  used by the broader Reticulum ecosystem, so a Go peer can exchange
  messages with any other LXMF client.

Both packages run on top of the same Reticulum-Go transport, so a
single application can serve MF and LXMF traffic side by side.

## Installation

```bash
go get git.quad4.io/RNS-Things/reticulum-go-mf
```

## Usage

### High-Level API

The `Messenger` provides a simple way to send and receive MF messages over an existing Reticulum-Go transport.

```go
package main

import (
	"encoding/hex"
	"log"
	"time"

	"git.quad4.io/RNS-Things/reticulum-go-mf/pkg/mf"
	"git.quad4.io/Networks/Reticulum-Go/pkg/common"
	"git.quad4.io/Networks/Reticulum-Go/pkg/destination"
	"git.quad4.io/Networks/Reticulum-Go/pkg/identity"
	"git.quad4.io/Networks/Reticulum-Go/pkg/transport"
)

func main() {
	// Setup Reticulum transport
	cfg := common.DefaultConfig()
	tr := transport.NewTransport(cfg)
	
	// Start the transport
	if err := tr.Start(); err != nil {
		log.Fatalf("Failed to start transport: %v", err)
	}

	// Create your identity and destination
	id, err := identity.NewIdentity()
	if err != nil {
		log.Fatal(err)
	}

	dest, err := destination.New(id, destination.In, destination.Single, "mf", tr)
	if err != nil {
		log.Fatal(err)
	}

	// Announce your destination so others can find you
	if err := dest.Announce(false, nil, nil); err != nil {
		log.Printf("Warning: Failed to announce: %v", err)
	}

	// Create the Messenger
	messenger := mf.NewMessenger(tr, dest)

	// Create or recall a remote peer's identity
	remoteId, err := identity.NewIdentity()
	if err != nil {
		log.Fatal(err)
	}
	remoteHash := remoteId.Hash()

	// Remember the remote identity so it can be found when sending
	identity.Remember(nil, remoteHash, remoteId.GetPublicKey(), nil)

	// Send a message with retry logic (waits for path if needed)
	err = sendMessageWithRetry(messenger, tr, remoteHash, "Hello from reticulum-go-mf!")
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	log.Printf("Message sent to %s", hex.EncodeToString(remoteHash))
}

func sendMessageWithRetry(messenger *mf.Messenger, tr *transport.Transport, 
	destHash []byte, text string) error {
	maxRetries := 5
	retryDelay := 2 * time.Second
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if tr.HasPath(destHash) {
			if err := messenger.SendMessage(destHash, text); err == nil {
				return nil
			}
		}
		if attempt < maxRetries {
			time.Sleep(retryDelay)
		}
	}
	return messenger.SendMessage(destHash, text)
}
```

### Low-Level Formatting

You can also use the `Message` struct directly for manual formatting.

```go
import "git.quad4.io/RNS-Things/reticulum-go-mf/pkg/mf"

// Create a message
senderHash, _ := hex.DecodeString("0123456789abcdef0123456789abcdef")
msg := &mf.Message{
    SenderHash: senderHash,
    Text:       "Hello, Reticulum!",
}

// Pack the message into [16 bytes sender hash][text payload]
packed, err := msg.Pack()
if err != nil {
    log.Fatal(err)
}

// Unpack the message from raw bytes
unpacked, err := mf.Unpack(packed)
if err != nil {
    log.Fatal(err)
}
```

## LXMF

`pkg/lxmf` implements the LXMF message layer on top of Reticulum-Go: packing
and signing, opportunistic delivery, stamps (proof-of-work), paper message
URIs, optional container wrapping, and lxmd-style INI config parsing. The
`Messenger` type ties this to a standard `lxmf` / `lxmf.delivery` inbound
destination so the local hash matches what other stacks derive for the
same identity.

The implementation tracks upstream LXMF semantics (version constant in
`pkg/lxmf`); the wire layout matches the ecosystem format so a Go process
can exchange traffic with **lxmd**, the Python `lxmf` client, and other
RNS tools that speak LXMF on the same Reticulum network.

```go
package main

import (
    "log"

    "git.quad4.io/RNS-Things/reticulum-go-mf/pkg/lxmf"
    "git.quad4.io/Networks/Reticulum-Go/pkg/common"
    "git.quad4.io/Networks/Reticulum-Go/pkg/identity"
    "git.quad4.io/Networks/Reticulum-Go/pkg/transport"
)

func main() {
    cfg := common.DefaultConfig()
    tr := transport.NewTransport(cfg)
    if err := tr.Start(); err != nil {
        log.Fatal(err)
    }

    id, _ := identity.NewIdentity()
    messenger, err := lxmf.NewDeliveryMessenger(id, tr)
    if err != nil {
        log.Fatal(err)
    }

    messenger.SetMessageHandler(func(m *lxmf.LXMessage, _ common.NetworkInterface) {
        log.Printf("inbound %s: %q", m.FormatHash(), m.ContentString())
    })

    if err := messenger.Destination().Announce(false, nil, nil); err != nil {
        log.Printf("announce failed: %v", err)
    }
}
```

`NewDeliveryMessenger` builds an inbound `lxmf.delivery` destination so
that the resulting destination hash matches the hash other LXMF
implementations would compute for the same identity.

The package also exposes lower-level primitives:

- `lxmf.NewMessage` and `lxmf.LXMessage.Pack` for assembling and
  signing a message manually.
- `lxmf.Unpack` and `lxmf.UnpackFromBytes` for parsing packed bytes
  arriving outside of the `Messenger` flow.
- `lxmf.DisplayNameFromAppData`, `lxmf.StampCostFromAppData` and the
  `EncodeAnnounceAppData*` helpers for the announce metadata format.

**Examples** — A two-local-UDP hop demo: `task example:lxmf`. A terminal
chat (hub, split panes, optional remote): `task example:lxmf:tui`.

**Tests** — Run the full LXMF test suite (unit, pack/unpack, stamps,
messenger loopback) with `task test:lxmf` or `go test ./pkg/lxmf/...`.

**Interop** — The package is written for the same on-wire behaviour as
the reference Python stack, so a Go node can hand messages to and from
`lxmd` and other LXMF software on a shared Reticulum mesh. The repository
may also ship optional `TestInterop` checks that call into Python; those
are run with `task test:lxmf:interop` and need `uv` and the `lxmf`
package available when that test is present. If that test is not in your
tree, the task simply runs no interop-matching cases.

## Prerequisites

- Go 1.26.2 or later
- [Task](https://taskfile.dev/) for build automation
- `uv` (optional, only for optional LXMF Python interop tests when
  `TestInterop` exists)

You may set `alias task='go-task'` in your shell if you invoke Task as
`go-task` on your system.

## Development

### Code Quality

Format code:

```bash
task fmt
```

Run the combined check (vet, lint, short tests, gosec scan):

```bash
task check
```

### Testing

Run all tests:

```bash
task test
```

Run short tests only:

```bash
task test-short
```

Generate coverage report:

```bash
task coverage
```

## Tasks

The project uses [Task](https://taskfile.dev/) for all development and build operations.

| Task                 | Description                                         |
|----------------------|-----------------------------------------------------|
| default              | Show available tasks                                |
| all                  | Clean, download dependencies, and test              |
| fmt                  | Format Go code                                      |
| vet                  | Run go vet                                          |
| lint                 | Run revive linter                                   |
| scan                 | Run gosec security scanner                          |
| check                | Run vet, lint, test-short, and scan                 |
| clean                | Remove build artifacts                              |
| test                 | Run all tests                                       |
| test-short           | Run short tests only                                |
| test-race            | Run tests with race detector                        |
| test:lxmf            | Run all LXMF package tests                          |
| test:lxmf:interop    | Run `TestInterop` (optional; needs `uv` if used)     |
| test:messenger       | Run MF two-way messenger test                       |
| bench                | Run benchmarks (lxmf, mf)                           |
| stress               | Run stress tests                                    |
| fuzz                 | Run fuzz targets (see `scripts/run-fuzz.sh`)        |
| coverage             | Generate test coverage report                       |
| deps                 | Download, verify, and refresh `vendor/`             |
| vendor               | Regenerate `vendor/`                              |
| mod-tidy             | Tidy go.mod file                                    |
| mod-verify           | Verify dependencies                                 |
| example              | Run basic MF example                                |
| example:messenger    | Run MF messenger example                            |
| example:lxmf         | Run two-local-UDP LXMF example                      |
| example:lxmf:tui     | Run LXMF terminal chat example                    |
| tinygo:test / build  | TinyGo test or build of MF example                  |

## License

This project is licensed under the [0BSD](LICENSE) license.

