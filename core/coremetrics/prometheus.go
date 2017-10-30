package coremetrics

import (
	"net/http"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	prometheus "gx/ipfs/QmX3QZ5jHEPidwUrymXV1iSCSUhdGxj15sm2gP4jKMef7B/client_golang/prometheus"
	p2phost "gx/ipfs/Qmc1XhrFEiSeBNn3mpfg6gEuYCt5im2gYmNVmncsvmpeAk/go-libp2p-host"
)

func MustRegister(h p2phost.Host, ps pstore.Peerstore) {
	c := &IpfsNodeCollector{PeerHost: h, Peerstore: ps}
	prometheus.MustRegister(c)
}

// This adds the scraping endpoint which Prometheus uses to fetch metrics.
func ScrapingHandler() http.Handler {
	return prometheus.UninstrumentedHandler()
}

// This adds collection of net/http-related metrics
func CollectorHandler(handlerName string, mux *http.ServeMux) http.HandlerFunc {
	return prometheus.InstrumentHandler(handlerName, mux)
}

var (
	peersTotalMetric = prometheus.NewDesc(
		prometheus.BuildFQName("ipfs", "p2p", "peers_total"),
		"Number of connected peers", []string{"transport"}, nil)
)

type IpfsNodeCollector struct {
	PeerHost p2phost.Host
}

func (_ IpfsNodeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- peersTotalMetric
}

func (c IpfsNodeCollector) Collect(ch chan<- prometheus.Metric) {
	for tr, val := range c.PeersTotalValues() {
		ch <- prometheus.MustNewConstMetric(
			peersTotalMetric,
			prometheus.GaugeValue,
			val,
			tr,
		)
	}
}

func (c IpfsNodeCollector) PeersTotalValues() map[string]float64 {
	vals := make(map[string]float64)
	if c.PeerHost == nil {
		return vals
	}
	for _, conn := range c.PeerHost.Network().Conns() {
		tr := ""
		for _, proto := range conn.RemoteMultiaddr().Protocols() {
			tr = tr + "/" + proto.Name
		}
		vals[tr] = vals[tr] + 1
	}
	return vals
}
