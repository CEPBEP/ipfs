package jaeger

import (
	"github.com/ipfs/go-ipfs/plugin"
	config "github.com/uber/jaeger-client-go/config"
	opentrace "gx/ipfs/QmbWpD2QmLEmmoXhYpnc2E4g3jWB5TawSZmYavJopcRk44/opentracing-go"
)

// Plugins is exported list of plugins that will be loaded
var Plugins = []plugin.Plugin{
	&jaegerPlugin{},
}

type jaegerPlugin struct{}

var _ plugin.PluginTracer = (*jaegerPlugin)(nil)

func (*jaegerPlugin) Name() string {
	return "jaeger"
}

func (*jaegerPlugin) Version() string {
	return "0.0.1"
}

func (*jaegerPlugin) Init() error {
	return nil
}

//Initalize a Jaeger tracer and set it as the global tracer in opentracing api
func (*jaegerPlugin) InitGlobalTracer() error {
	tracerCfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans: false,
		},
	}
	//we are ignoring the closer for now
	tracer, _, err := tracerCfg.New("IPFS-NODE-ID")
	if err != nil {
		//probably failed to init the tracer
		return err
	}
	opentrace.SetGlobalTracer(tracer.(opentrace.Tracer))
	return nil
}
