package trapi

import (
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metrics"
	"github.com/mholt/caddy"
	"github.com/miekg/dns"
	"io/ioutil"
	"net/http"
)

// TODO: design a proper API (json?)
// TODO: support HTTP API over unix socket
// TODO: consider using the (existing?) caddy server instead of net/http

// init registers this plugin within the Caddy plugin framework. It uses "trapi" as the
// name, and couples it to the Action "setup".
func init() {
	caddy.RegisterPlugin("trapi", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

var transientResourceRecords = make(map[string][]dns.RR)

// setup is the function that gets called when the config parser see the token "trap". Setup is responsible
// for parsing any extra options the trapi plugin may have. The first token this function sees is "trapi".
func setup(c *caddy.Controller) error {
	c.Next() // Ignore "trapi" and give us the next token.

	// parse plugin settings
	if !c.NextArg() {
		// argument(s) is/are required
		return plugin.Error("trapi", c.ArgErr())
	}
	// try to start HTTP API server with listen address and port
	http.HandleFunc("/trapi", serveAPI)
	err := http.ListenAndServe(c.Val(), nil)

	//input := caddy.CaddyfileInput{Contents: []byte(c.Val()), Filepath: "trapi generated", ServerTypeName: "http"}
	//_, err := caddy.Start(input)
	if err != nil {
		return plugin.Error("trapi", c.Errf("failed starting http server for trapi %s", err))
	}

	if c.NextArg() {
		// no further arguments are expected
		return plugin.Error("trapi", c.ArgErr())
	}

	// Add a startup function that will -- after all plugins have been loaded -- check if the
	// prometheus plugin has been used - if so we will export metrics. We can only register
	// this metric once, hence the "once.Do".
	c.OnStartup(func() error {
		once.Do(func() { metrics.MustRegister(c, requestCountAnswered) })
		return nil
	})

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return Trapi{Next: next, TRRs: transientResourceRecords}
	})

	// All OK, return a nil error.
	return nil
}

func serveAPI(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil || body == nil {
		log.Warning("received invalid api request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Debugf("Received trapi API call %s", body)

	rr, err := dns.NewRR(string(body))
	if err != nil || rr == nil {
		log.Warning("received invalid resource record in api request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Debug("Creating new transient RR")

	// lookup existing record(s)
	trrs := transientResourceRecords[rr.Header().Name]
	//rr = TransientResourceRecord{Name: rrh.Name, Class: rrh.Class, Rrtype: rrh.Rrtype, Ttl: rrh.Ttl }
	if trrs == nil { // or add new list
		trrs = make([]dns.RR, 1)
	}
	// append new RR to list
	trrs = append(trrs, rr)

	// confirm to api client
	w.WriteHeader(http.StatusCreated)
}