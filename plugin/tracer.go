package plugin

// PluginTracer is an interface that can be implemented to add a tracer
type PluginTracer interface {
	Plugin
	InitGlobalTracer() error
}
