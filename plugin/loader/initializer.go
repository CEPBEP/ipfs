package loader

import (
	"github.com/ipfs/go-ipfs/core/coredag"
	"github.com/ipfs/go-ipfs/plugin"

	format "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
)

func initialize(plugins []plugin.Plugin) error {
	for _, p := range plugins {
		err := p.Init()
		if err != nil {
			return err
		}
	}

	return nil
}

func run(plugins []plugin.Plugin) error {
	for _, pl := range plugins {
		switch pl.(type) {
		case plugin.PluginIPLD:
			err := runIPLDPlugin(pl.(plugin.PluginIPLD))
			if err != nil {
				return err
			}
		case plugin.PluginTracer:
			err := runTracerPlugin(pl.(plugin.PluginTracer))
			if err != nil {
				return err
			}
		default:
			panic(pl)
		}
	}
	return nil
}

func runIPLDPlugin(pl plugin.PluginIPLD) error {
	err := pl.RegisterBlockDecoders(format.DefaultBlockDecoder)
	if err != nil {
		return err
	}

	return pl.RegisterInputEncParsers(coredag.DefaultInputEncParsers)
}

func runTracerPlugin(pl plugin.PluginTracer) error {
	err := pl.InitGlobalTracer()
	if err != nil {
		return err
	}
	return nil
}
