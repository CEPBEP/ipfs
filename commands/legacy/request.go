package legacy

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"

	"gx/ipfs/QmVD1W3MC8Hk1WZgFQPWWmBECJ3X72BgUYf9eCQ4PGzPps/go-ipfs-cmdkit"
	"gx/ipfs/QmVD1W3MC8Hk1WZgFQPWWmBECJ3X72BgUYf9eCQ4PGzPps/go-ipfs-cmdkit/files"
	"gx/ipfs/QmYopJAcV7R9SbxiPBCvqhnt8EusQpWPHewoZakCMt8hps/go-ipfs-cmds"

	oldcmds "github.com/ipfs/go-ipfs/commands"
)

// requestWrapper implements a oldcmds.Request from an Request
type requestWrapper struct {
	req *cmds.Request
	ctx *oldcmds.Context
}

// InvocContext retuns the invocation context of the oldcmds.Request.
// It is faked using OldContext().
func (r *requestWrapper) InvocContext() *oldcmds.Context {
	return r.ctx
}

// SetInvocContext sets the invocation context. First the context is converted
// to a Context using NewContext().
func (r *requestWrapper) SetInvocContext(ctx oldcmds.Context) {
	r.ctx = &ctx
}

// Command is an empty stub.
func (r *requestWrapper) Command() *oldcmds.Command { return nil }

func (r *requestWrapper) Arguments() []string {
	return r.req.Arguments
}

func (r *requestWrapper) Context() context.Context {
	return r.req.Context
}

func (r *requestWrapper) ConvertOptions() error {
	return convertOptions(r.req)
}

func (r *requestWrapper) Files() files.File {
	return r.req.Files
}

func (r *requestWrapper) Option(name string) *cmdkit.OptionValue {
	var option cmdkit.Option

	for _, def := range r.req.Command.Options {
		for _, optName := range def.Names() {
			if name == optName {
				option = def
				break
			}
		}
	}
	if option == nil {
		return nil
	}

	// try all the possible names, break if we find a value
	for _, n := range option.Names() {
		val, found := r.req.Options[n]
		if found {
			return &cmdkit.OptionValue{val, found, option}
		}
	}

	return &cmdkit.OptionValue{option.Default(), false, option}
}

func (r *requestWrapper) Options() cmdkit.OptMap {
	return r.req.Options
}

func (r *requestWrapper) Path() []string {
	return r.req.Path
}

func (r *requestWrapper) SetArguments(args []string) {
	r.req.Arguments = args
}

func (r *requestWrapper) SetFiles(f files.File) {
	r.req.Files = f
}

func (r *requestWrapper) SetOption(name string, v interface{}) {
	r.req.SetOption(name, v)
}

func (r *requestWrapper) SetOptions(om cmdkit.OptMap) error {
	r.req.Options = om
	return convertOptions(r.req)
}

func (r *requestWrapper) SetRootContext(ctx context.Context) error {
	r.req.Context = ctx
	return nil
}

func (r *requestWrapper) Stdin() io.Reader {
	return os.Stdin
}

func (r *requestWrapper) StringArguments() []string {
	return r.req.Arguments
}

func (r *requestWrapper) Values() map[string]interface{} {
	return nil
}

func (r *requestWrapper) VarArgs(f func(string) error) error {
	if len(r.req.Arguments) >= len(r.req.Command.Arguments) {
		for _, arg := range r.req.Arguments {
			err := f(arg)
			if err != nil {
				return err
			}
		}
		return nil
	}

	s, err := r.req.BodyArgs()
	if err != nil {
		return err
	}

	for s.Scan() {
		err = f(s.Text())
		if err != nil {
			return err
		}
	}

	return nil
}

// copied from go-ipfs-cmds/request.go
func convertOptions(req *cmds.Request) error {
	optDefSlice := req.Command.Options

	optDefs := make(map[string]cmdkit.Option)
	for _, def := range optDefSlice {
		for _, name := range def.Names() {
			optDefs[name] = def
		}
	}

	for k, v := range req.Options {
		opt, ok := optDefs[k]
		if !ok {
			continue
		}

		kind := reflect.TypeOf(v).Kind()
		if kind != opt.Type() {
			if str, ok := v.(string); ok {
				val, err := opt.Parse(str)
				if err != nil {
					value := fmt.Sprintf("value %q", v)
					if len(str) == 0 {
						value = "empty value"
					}
					return fmt.Errorf("Could not convert %q to type %q (for option %q)",
						value, opt.Type().String(), "-"+k)
				}
				req.Options[k] = val

			} else {
				return fmt.Errorf("Option %q should be type %q, but got type %q",
					k, opt.Type().String(), kind.String())
			}
		}

		for _, name := range opt.Names() {
			if _, ok := req.Options[name]; name != k && ok {
				return fmt.Errorf("Duplicate command options were provided (%q and %q)",
					k, name)
			}
		}
	}

	return nil
}
