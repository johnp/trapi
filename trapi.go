// package trapi is a CoreDNS plugin that provides a transient resource record API
//
package trapi

import (
	"context"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metrics"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

var log = clog.NewWithPlugin("trapi")


type Trapi struct {
	Next plugin.Handler
	//Names []string
	TRRs map[string][]dns.RR
}

type TransientResourceRecord struct {
	Name     string
	Rrtype   uint16
	Class    uint16
	Ttl      uint32
	data     string
}


// ServeDNS implements the plugin.Handler interface.
func (t Trapi) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r, Context: ctx}

	// find RR by QName
	qname := state.Name()
	trrs, ok := t.TRRs[qname]
	if !ok || trrs == nil {
		return plugin.NextOrFailure(t.Name(), t.Next, ctx, w, r)
	}

	// filter RRs for QType
	ftrrs := make([]dns.RR, len(trrs))
	for _, rr := range ftrrs {
		// TODO: this should probably also check Class/...(?)
		if state.QType() == rr.Header().Rrtype {
			ftrrs = append(ftrrs, rr)
		}
	}
	// TODO: check TTL and cleanup expired RRs
	log.Debug("Handling query via trapi")

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative, m.RecursionAvailable = true, false
	m.Answer = ftrrs
	err := w.WriteMsg(m)
	if err != nil {
		log.Warning("failed writing trapi reply; falling-through...")
		// fallthrough to next plugin
		return plugin.NextOrFailure(t.Name(), t.Next, ctx, w, r)
	}
	log.Debug("Responded to query via trapi")
	// count the request as answered by the plugin
	requestCountAnswered.WithLabelValues(metrics.WithServer(ctx)).Inc()
	// finish
	return dns.RcodeSuccess, nil
}

// Name implements the Handler interface.
func (t Trapi) Name() string { return "trapi" }
