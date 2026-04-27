// SPDX-License-Identifier: 0BSD
package lxmf

import (
	"bytes"
	"errors"
	"sync"
	"testing"

	"git.quad4.io/Networks/Reticulum-Go/pkg/identity"
)

var errStampWorkblockEmpty = errors.New("lxmf stress: empty workblock")

func TestStress_LXMF_ParallelPackUnpack(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	const workers = 48
	const iters = 150

	src := mustNewIdentity(t)
	dst := mustNewIdentity(t)
	dh, sh := dst.Hash(), src.Hash()
	identity.Remember(nil, sh, src.GetPublicKey(), nil)

	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				msg, err := NewMessage(dh, sh, []byte("t"), []byte("content-stress"), nil)
				if err != nil {
					errs <- err
					return
				}
				msg.Timestamp = 1_800_000_000.0 + float64(seed*iters+i)
				packed, err := msg.Pack(src)
				if err != nil {
					errs <- err
					return
				}
				if _, err := Unpack(packed, RecallSource); err != nil {
					errs <- err
					return
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestStress_LXMF_StampWorkblockParallel(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	const workers = 32
	const iters = 80
	material := bytes.Repeat([]byte{0x31}, 32)

	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				wb, err := StampWorkblock(material, 8+i%5)
				if err != nil {
					errs <- err
					return
				}
				if len(wb) == 0 {
					errs <- errStampWorkblockEmpty
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}
