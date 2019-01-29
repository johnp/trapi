package trapi

import (
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"testing"
	"time"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"

	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

func TestResponseWriter(t *testing.T) {
	// setup (a) static test record(s)
	origin := plugin.Name("example.org.").Normalize()
	exampleOrgRR, _ := dns.NewRR("example.org. 3600 IN TXT injected")
	exampleOrgMap := make([]TemporaryResourceRecord, 0)
	exampleOrgMap = append(exampleOrgMap, TemporaryResourceRecord{RR: exampleOrgRR, Created: time.Now(), Ttl: 3600})
	TRRs.Lock()
	TRRs.internal[origin] = exampleOrgMap
	TRRs.Names = append(TRRs.Names, origin)
	TRRs.Unlock()

	ctx := context.TODO()
	req := new(dns.Msg)
	req.SetQuestion("example.org.", dns.TypeA)

	// Create a new Recorder that captures the result
	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	// Create dummy state from request
	state := request.Request{W: rec, Req: req, Context: ctx}
	trrrw := TRRResponseWriter{rec, state}
	// Create dummy response from downstream
	response := new(dns.Msg)
	response.SetReply(req)
	response.Authoritative = true
	downstreamAnswer, _ := dns.NewRR("example.org. 3600 IN TXT loremipsum")
	response.Answer = append(response.Answer, downstreamAnswer)
	// Call the plugin directly
	_ = trrrw.WriteMsg(response)

	if !rrsetContains(downstreamAnswer, rec.Msg.Answer) {
		t.Fatalf("downstreamAnswer (%s) not in rec.Msg (%s)!", downstreamAnswer, rec.Msg.Answer)
	}

	if !rrsetContains(exampleOrgRR, rec.Msg.Answer) {
		t.Fatalf("exampleOrgRR (%s) not in rec.Msg (%s)!", exampleOrgRR, rec.Msg.Answer)
	}
}

func rrsetContains(rr dns.RR, list []dns.RR) bool {
	for _, lrr := range list {
		if rr == lrr {
			return true
		}
	}
	return false
}
