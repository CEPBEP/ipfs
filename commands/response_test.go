package commands

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	cmdkit "gx/ipfs/QmVD1W3MC8Hk1WZgFQPWWmBECJ3X72BgUYf9eCQ4PGzPps/go-ipfs-cmdkit"
	cmds "gx/ipfs/QmYopJAcV7R9SbxiPBCvqhnt8EusQpWPHewoZakCMt8hps/go-ipfs-cmds"
)

type TestOutput struct {
	Foo, Bar string
	Baz      int
}

func TestMarshalling(t *testing.T) {
	cmd := &Command{}
	opts, _ := cmd.GetOptions(nil)

	req, _ := NewRequest(nil, nil, nil, nil, nil, opts)

	res := NewResponse(req)
	res.SetOutput(TestOutput{"beep", "boop", 1337})

	_, err := res.Marshal()
	if err == nil {
		t.Error("Should have failed (no encoding type specified in request)")
	}

	req.SetOption(cmds.EncLong, JSON)

	reader, err := res.Marshal()
	if err != nil {
		t.Error(err, "Should have passed")
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	output := buf.String()
	if removeWhitespace(output) != "{\"Foo\":\"beep\",\"Bar\":\"boop\",\"Baz\":1337}" {
		t.Error("Incorrect JSON output")
	}

	res.SetError(fmt.Errorf("Oops!"), cmdkit.ErrClient)
	reader, err = res.Marshal()
	if err != nil {
		t.Error("Should have passed")
	}
	buf.Reset()
	buf.ReadFrom(reader)
	output = buf.String()
	fmt.Println(removeWhitespace(output))
	if removeWhitespace(output) != `{"Message":"Oops!","Code":1,"Type":"error"}` {
		t.Error("Incorrect JSON output")
	}
}

func TestErrTypeOrder(t *testing.T) {
	if cmdkit.ErrNormal != 0 || cmdkit.ErrClient != 1 || cmdkit.ErrImplementation != 2 || cmdkit.ErrNotFound != 3 {
		t.Fatal("ErrType order is wrong")
	}
}

func removeWhitespace(input string) string {
	input = strings.Replace(input, " ", "", -1)
	input = strings.Replace(input, "\t", "", -1)
	input = strings.Replace(input, "\n", "", -1)
	return strings.Replace(input, "\r", "", -1)
}
