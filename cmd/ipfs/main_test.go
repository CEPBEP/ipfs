package main

import (
	"testing"

	"gx/ipfs/QmVyK9pkXc5aPCtfxyvRTLrieon1CD31QmcmUxozBc32bh/go-ipfs-cmdkit"
)

func TestIsCientErr(t *testing.T) {
	t.Log("Only catch pointers")
	if !isClientError(&cmdkit.Error{Code: cmdkit.ErrClient}) {
		t.Errorf("misidentified error")
	}
}
