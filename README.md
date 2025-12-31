# reticulum-go-mf

A message format for Reticulum-Go.

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

	dest, err := destination.New(id, destination.IN, destination.SINGLE, "mf", tr)
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

## Prerequisites

- Go 1.24 or later
- [Task](https://taskfile.dev/) for build automation

Note: You may need to set `alias task='go-task'` in your shell configuration to use `task` instead of `go-task`.

### Nix

If you have Nix installed, you can use the development shell which automatically provides all dependencies including Task:

```bash
nix develop
```

This will enter a development environment with Go and Task pre-configured.

## Development

### Code Quality

Format code:

```bash
task fmt
```

Run static analysis checks (formatting, vet, linting):

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

| Task                | Description                                          |
|---------------------|------------------------------------------------------|
| default             | Show available tasks                                 |
| all                 | Clean, download dependencies, and test              |
| fmt                 | Format Go code                                       |
| fmt-check           | Check if code is formatted (CI-friendly)             |
| vet                 | Run go vet                                           |
| lint                | Run revive linter                                    |
| scan                | Run gosec security scanner                           |
| check               | Run fmt-check, vet, and lint                         |
| clean               | Remove build artifacts                               |
| test                | Run all tests                                        |
| test-short          | Run short tests only                                 |
| test-race           | Run tests with race detector                         |
| coverage            | Generate test coverage report                        |
| deps                | Download and verify dependencies                     |
| mod-tidy            | Tidy go.mod file                                     |
| mod-verify          | Verify dependencies                                  |

example: task test

## License

This project is licensed under the [0BSD](LICENSE) license.

