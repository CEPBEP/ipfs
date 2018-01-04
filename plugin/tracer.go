package plugin

import (
	"github.com/opentracing/opentracing-go"
)

// PluginTracer is an interface that can be implemented to add a tracer
type PluginTracer interface {
	Plugin
	InitGlobalTracer() (opentracing.Tracer, error)
}
