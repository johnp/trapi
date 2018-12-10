package trapi

import (
	"testing"
	"time"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"

	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

func TestTrapi(t *testing.T) {
	// Create a new Trapi Plugin. Use the test.ErrorHandler as the next plugin.
	x := Trapi{Next: test.ErrorHandler(), TRRs: make(map[string][]TransientResourceRecord)}

	// setup (a) static test record(s)
	exampleOrgRR, err := dns.NewRR("example.org. 300 IN A 127.0.0.1")
	if exampleOrgRR == nil || err != nil {
		t.Fatalf("Failed creating static test RR: %v", err)
	}
	exampleOrgMap := make([]TransientResourceRecord, 0)
	exampleOrgMap = append(exampleOrgMap, TransientResourceRecord{RR: exampleOrgRR, Created: time.Now()})
	x.TRRs[exampleOrgRR.Header().Name] = exampleOrgMap

	// TODO: add a record via trapi HTTP api

	ctx := context.TODO()
	r := new(dns.Msg)
	r.SetQuestion("example.org.", dns.TypeA)
	// Create a new Recorder that captures the result
	rec := dnstest.NewRecorder(&test.ResponseWriter{})

	// Call the plugin directly
	x.ServeDNS(ctx, rec, r)

	// TODO: check the result(s)
	if !rrInAnswer(exampleOrgRR, rec.Msg.Answer) {
		t.Fatalf("rec.Msg (%s) and exampleOrgRR (%s) do not match!", rec.Msg, exampleOrgRR)
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