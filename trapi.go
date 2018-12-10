// package trapi is a CoreDNS plugin that provides a transient resource record API
//
package trapi

import (
	"context"
	"time"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metrics"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

var log = clog.NewWithPlugin("trapi")


type Trapi struct {
	Next plugin.Handler
	// TODO: make communication with API server thread-safe
	TRRs map[string][]TransientResourceRecord
}

type TransientResourceRecord struct {
	RR dns.RR
	Created time.Time // TODO: If time.Now() > Created+RR.TTL forget the TRR
}


// ServeDNS implements the plugin.Handler interface.
func (t Trapi) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r, Context: ctx}

	// debug print current list
	log.Infof("transientResourceRecords: %s", t.TRRs)

	// find RR by QName
	qname := state.Name()
	trrs := t.TRRs[qname]
	if trrs == nil {
		log.Infof("skipping trapi -- '%s' not found", qname)
		return plugin.NextOrFailure(t.Name(), t.Next, ctx, w, r)
	}
	log.Infof("found '%s' in trapi; checking QType...", qname)
	// filter RRs for QType
	qtype := state.QType()
	ans := make([]dns.RR, 0)
	log.Infof("trrs %s", trrs)
	for _, trr := range trrs {
		// TODO: this should probably also check Class/...(?)
		// TODO: check TTL and cleanup expired RRs
		log.Infof("looking at trr: %s", trr)
		if trr.RR != nil && qtype == trr.RR.Header().Rrtype {
			log.Infof("is QType %s", trr.RR.Header().Rrtype)
			ans = append(ans, trr.RR)
		}
	}
	log.Infof("Handling query for %s via trapi with ans %s", qname, ans)

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.Answer = ans
	state.SizeAndDo(m)
	err := w.WriteMsg(m)
	if err != nil {
		log.Warningf("failed writing trapi reply (err: %s); falling-through...", err)
		// fallthrough to next plugin
		return plugin.NextOrFailure(t.Name(), t.Next, ctx, w, r)
	}
	log.Info("Responded to query via trapi")
	// count the request as answered by the plugin
	requestCountAnswered.WithLabelValues(metrics.WithServer(ctx)).Inc()
	// finish
	return dns.RcodeSuccess, nil
}

// Name implements the Handler interface.
func (t Trapi) Name() string { return "trapi" }
