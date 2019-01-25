package trapi

import (
	"github.com/coredns/coredns/plugin"
	"sync"
	"testing"
	"time"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"

	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

func TestResponseWriter(t *testing.T) {
	// Create a new Trapi Plugin. Use the test.ErrorHandler as the next plugin.
	TRRs := RWLockableTRRMap{sync.RWMutex{}, make(map[string][]TemporaryResourceRecord), []string{}}
	x := Trapi{Next: test.ErrorHandler()}

	// setup (a) static test record(s)
	exampleOrgRR, err := dns.NewRR("example.org. 300 IN A 127.0.0.1")
	if exampleOrgRR == nil || err != nil {
		t.Fatalf("Failed creating static test RR: %v", err)
	}
	origin := plugin.Name(exampleOrgRR.Header().Name).Normalize()
	exampleOrgMap := make([]TemporaryResourceRecord, 0)
	exampleOrgMap = append(exampleOrgMap, TemporaryResourceRecord{RR: exampleOrgRR, Created: time.Now(), Ttl: 60})
	TRRs.Lock()
	TRRs.internal[origin] = exampleOrgMap
	TRRs.Names = append(TRRs.Names, origin)
	TRRs.Unlock()

	// TODO: add record(s) via trapi HTTP api and test their existence / serial numbers

	ctx := context.TODO()
	r := new(dns.Msg)
	r.SetQuestion("example.org.", dns.TypeA)
	// Create a new Recorder that captures the result
	rec := dnstest.NewRecorder(&test.ResponseWriter{})

	// Call the plugin directly
	_, _ = x.ServeDNS(ctx, rec, r)

	// TODO: check the result(s)
	if !rrInAnswer(exampleOrgRR, rec.Msg.Answer) {
		t.Fatalf("rec.Msg (%s) and exampleOrgRR (%s) do not match!", rec.Msg.Answer, exampleOrgRR)
	}
}

func rrInAnswer(rr dns.RR, list []dns.RR) bool {
	for _, rrr := range list {
		if rr == rrr {
			return true
		}
	}
	return false
}
