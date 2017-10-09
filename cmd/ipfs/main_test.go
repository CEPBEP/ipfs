package main

import (
	"testing"

	"gx/ipfs/QmPMeikDc7tQEDvaS66j1bVPQ2jBkvFwz3Qom5eA5i4xip/go-ipfs-cmds"
)

func TestIsCientErr(t *testing.T) {
	t.Log("Only catch pointers")
	if !isClientError(&cmds.Error{Code: cmdkit.ErrClient}) {
		t.Errorf("misidentified error")
	}
}
