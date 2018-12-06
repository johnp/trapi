package trapi

import (
	"testing"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"

	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

func TestTrapi(t *testing.T) {
	// Create a new Trapi Plugin. Use the test.ErrorHandler as the next plugin.
	x := Trapi{Next: test.ErrorHandler()}

	// TODO: add a record via trapi

	ctx := context.TODO()
	r := new(dns.Msg)
	r.SetQuestion("example.org.", dns.TypeA)
	// Create a new Recorder that captures the result, this isn't actually used in this test
	// as it just serves as something that implements the dns.ResponseWriter interface.
	rec := dnstest.NewRecorder(&test.ResponseWriter{})

	// Call the plugin directly
	x.ServeDNS(ctx, rec, r)

	// TODO: check the result
}
