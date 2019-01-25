package trapi

import (
	"errors"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/mholt/caddy"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
)

// TODO: consider using the (existing?) caddy server instead of net/http if possible
// TODO: goroutines vs threads ?

var log = clog.NewWithPlugin("trapi")
var TRRs = RWLockableTRRMap{sync.RWMutex{}, make(map[string][]TemporaryResourceRecord), []string{}}

// init registers this plugin within the Caddy plugin framework.
func init() {
	caddy.RegisterPlugin("trapi", caddy.Plugin{
		ServerType: "dns",
		Action:     setupTrapi,
	})
}

// setup is the function that gets called when the config parser see the token "trap". Setup is responsible
// for parsing any extra options the trapi plugin may have. The first token this function sees is "trapi".
func setupTrapi(c *caddy.Controller) error {
	c.Next() // Ignore "trapi" and give us the next token.

	// parse plugin settings
	if !c.NextArg() {
		// argument(s) is/are required
		return plugin.Error("trapi", c.ArgErr())
	}
	// listen address
	// TODO: better way to specify a unix socket path
	listen := c.Val()

	// find API token if required
	var token = ""
	for c.NextBlock() {
		switch c.Val() {
		case "token":
			if c.NextArg() {
				token = c.Val()
			} else {
				return plugin.Error("trapi", c.ArgErr())
			}
			break
		}
	}

	if c.NextArg() { // no further arguments expected
		return plugin.Error("trapi", c.ArgErr())
	}

	if token == "" {
		return plugin.Error("trapi", errors.New("API token required"))
	} else {
		log.Infof("API Token %v", token)
	}

	api := API{Token: token}
	if strings.HasPrefix(listen, string(os.PathSeparator)) {
		log.Infof("listening on socket %s", listen)
		// socket is e.g. "/run/coredns.trapi.sock"
		os.Remove(listen) // TODO: check for existence first; possible foot-gun
		// TODO: make sure socket has correct (i.e. 600) permissions
		unixListener, err := net.Listen("unix", listen)
		if err != nil {
			log.Fatal("Listen (UNIX socket): ", err)
		}
		defer unixListener.Close()
		// Start the API server
		go http.Serve(unixListener, &api)
	} else {
		log.Infof("listening on %s", listen)
		// Start the API server
		// TODO: add error handling (e.g. invalid listen address)
		go http.ListenAndServe(listen, &api) // TODO: check how the grpc plugin does this
	}

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return Trapi{Next: next}
	})

	// All OK, return a nil error.
	return nil
}
