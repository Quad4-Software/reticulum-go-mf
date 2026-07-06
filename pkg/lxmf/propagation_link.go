// SPDX-License-Identifier: 0BSD
package lxmf

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"quad4/reticulum-go/pkg/destination"
	"quad4/reticulum-go/pkg/identity"
	"quad4/reticulum-go/pkg/link"
	"quad4/reticulum-go/pkg/resource"
)

const propagationLinkWait = 90 * time.Second

func (m *Messenger) ensurePropagationLink(propNodeHash []byte) (*link.Link, error) {
	if m == nil || m.transport == nil {
		return nil, errors.New("lxmf: messenger not initialized")
	}
	if len(propNodeHash) != DestinationLength {
		return nil, fmt.Errorf("propagation node: %w", ErrInvalidHashLength)
	}

	m.propLinkMu.Lock()
	if m.propLink != nil && m.propLink.IsActive() && bytes.Equal(m.propLinkNode, propNodeHash) {
		lnk := m.propLink
		m.propLinkMu.Unlock()
		return lnk, nil
	}
	if m.propLink != nil {
		m.propLink.Teardown()
		m.propLink = nil
		m.propLinkNode = nil
	}
	m.propLinkMu.Unlock()

	if !m.transport.HasPath(propNodeHash) {
		if err := m.transport.RequestPath(propNodeHash, "", nil, true); err != nil {
			return nil, fmt.Errorf("propagation path request: %w", err)
		}
	}

	deadline := time.Now().Add(pathWaitForPropagation)
	for !m.transport.HasPath(propNodeHash) {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("propagation node %x: no path", propNodeHash)
		}
		time.Sleep(pathPollInterval)
	}

	pnIdentity, err := identity.Recall(propNodeHash)
	if err != nil {
		return nil, fmt.Errorf("propagation node identity: %w", err)
	}
	if pnIdentity == nil {
		return nil, fmt.Errorf("propagation node %x: %w", propNodeHash, ErrDestinationUnknown)
	}

	destOut, err := destination.FromHash(propNodeHash, pnIdentity, destination.Single, m.transport)
	if err != nil {
		return nil, fmt.Errorf("propagation destination: %w", err)
	}

	lnk := link.NewLink(destOut, m.transport, nil, nil, func(closed *link.Link) {
		m.propLinkMu.Lock()
		if m.propLink == closed {
			m.propLink = nil
			m.propLinkNode = nil
		}
		m.propLinkMu.Unlock()
	})
	if err := lnk.Establish(); err != nil {
		return nil, fmt.Errorf("propagation link request: %w", err)
	}
	lnk.Start()

	hops := m.transport.HopsTo(propNodeHash)
	timeout := time.Duration(link.EstablishmentTimeoutPerHop)*time.Second*time.Duration(maxU8(hops, 1)) + 10*time.Second
	if timeout > propagationLinkWait {
		timeout = propagationLinkWait
	}

	waitDeadline := time.Now().Add(timeout)
	for !lnk.IsActive() {
		if time.Now().After(waitDeadline) {
			lnk.Teardown()
			return nil, errors.New("propagation link establishment timeout")
		}
		time.Sleep(pathPollInterval)
	}

	m.propLinkMu.Lock()
	m.propLink = lnk
	m.propLinkNode = append([]byte(nil), propNodeHash...)
	m.propLinkMu.Unlock()
	return lnk, nil
}

func (m *Messenger) sendPropagationPayload(lnk *link.Link, payload []byte) error {
	if lnk == nil {
		return errors.New("lxmf: nil propagation link")
	}
	if len(payload) == 0 {
		return errors.New("lxmf: empty propagation payload")
	}

	if len(payload) <= LinkPacketMaxContent {
		if err := lnk.SendPacket(payload); err != nil {
			return fmt.Errorf("propagation link packet: %w", err)
		}
		return nil
	}

	res, err := resource.New(payload, true)
	if err != nil {
		return fmt.Errorf("propagation resource: %w", err)
	}
	if err := lnk.SendResource(res); err != nil {
		return fmt.Errorf("propagation link resource: %w", err)
	}

	deadline := time.Now().Add(5 * time.Minute)
	for {
		switch res.GetStatus() {
		case resource.StatusComplete:
			return nil
		case resource.StatusFailed, resource.StatusCancelled:
			return errors.New("propagation resource transfer failed")
		}
		if time.Now().After(deadline) {
			return errors.New("propagation resource transfer timeout")
		}
		time.Sleep(pathPollInterval)
	}
}

const pathWaitForPropagation = 60 * time.Second

func maxU8(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}
