// package trapi is a plugin that provides a temporary resource record API
package trapi

import (
	"context"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/file"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"sync"
	"time"
)

type Trapi struct {
	Next plugin.Handler
}

// TODO: maybe use the same struct layout (and types, e.g. `Zone`) as the file plugin
type RWLockableTRRMap struct {
	sync.RWMutex
	internal map[string][]TemporaryResourceRecord // A map mapping zone (origin) to the the zones TRRs
	Names    []string                             // // All the keys from the map Z as a string slice.
}

type TemporaryZone struct {
	file.Zone
	Created time.Time
}

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
	log.Infof("@trapi.ServeDNS(%v): type: %s qname: %s r: %s", exists, state.Type(), qname, r.String())
	if exists { // TODO(performance): filter by Message OpCode/QType here as well already
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
	return trr.Header().Name == state.Name()
}