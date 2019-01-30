// package trapi is a plugin that provides a temporary resource record API
package trapi

import (
	"context"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"sync"
	"time"
)

type Trapi struct {
	Next plugin.Handler
}

type RWLockableTRRMap struct {
	sync.RWMutex
	// structure similar to the `file` plugin
	internal map[string][]TemporaryResourceRecord // A map mapping zone (origin) to the the zones TRRs
	Names    []string                             // All the keys from the map Z as a string slice.
}

// TODO: maybe use the file plugins Zone type as well
//type TemporaryZone struct {
//	file.Zone
//	Created time.Time
//}

type TemporaryResourceRecord struct {
	dns.RR
	Created time.Time // by default lives until Created+TTl
	Ttl     uint32
}

// ServeDNS implements the plugin.Handler interface.
func (t Trapi) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r, Context: ctx}
	qname := state.Name()
	// only inject custom response writer if requested name/origin exists in list of TRRs
	TRRs.RLock()
	exists := plugin.Zones(TRRs.Names).Matches(qname) != ""
	TRRs.RUnlock()
	log.Debugf("ServeDNS(): exists: %v qname: %s qtype: %s", exists, qname, state.Type())
	if exists { // TODO(enhancement): filter by Message OpCode/QType here as well already
		w = &TRRResponseWriter{w, state}
	}
	return plugin.NextOrFailure(t.Name(), t.Next, ctx, w, r)
}

// Name implements the Handler interface.
func (t Trapi) Name() string { return "trapi" }

func (trr TemporaryResourceRecord) Expired() bool {
	expires := trr.Created.Add(time.Second * time.Duration(trr.Ttl))
	return time.Now().After(expires)
}

func (trr TemporaryResourceRecord) Matches(state request.Request) bool {
	return (trr.Header().Rrtype == state.QType() || dns.TypeANY == state.QType()) &&
		trr.Header().Class == state.QClass() && trr.Header().Name == state.Name()
}

func (trr TemporaryResourceRecord) NameMatches(state request.Request) bool {
	isXFR := state.QType() == dns.TypeAXFR || state.QType() == dns.TypeIXFR
	requestName := state.Name()
	trrName := trr.Header().Name
	return (!isXFR && requestName == trrName) || (isXFR && plugin.Name(requestName).Matches(trrName))
}