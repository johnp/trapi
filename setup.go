package trapi

import (
	"errors"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/file"
	"github.com/coredns/coredns/plugin/kubernetes"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/mholt/caddy"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// TODO: consider using the (existing?) caddy server instead of net/http
//       (caddy doesn't seem to support unix socket transports)
// TODO: goroutines vs threads ?

var log = clog.NewWithPlugin("trapi")
var TRRs = RWLockableTRRMap{sync.RWMutex{}, make(map[string][]TemporaryResourceRecord), []string{}}
var transferToMap = make(map[string][]string, 0) // maps origin to transfer to targets TODO: deduplicate targets

// init registers this plugin within the Caddy plugin framework.
func init() {
	caddy.RegisterPlugin("trapi", caddy.Plugin{
		ServerType: "dns",
		Action:     setupTrapi,
	})
}

// setup is the function that gets called when the config parser see the token "trapi". Setup is responsible
// for parsing any extra options the trapi plugin may have. The first token this function sees is "trapi".
func setupTrapi(c *caddy.Controller) error {
	c.Next() // skip "trapi" token

	// parse plugin settings
	if !c.NextArg() {
		return plugin.Error("trapi", c.ArgErr()) // arguments are required
	}
	// listen address
	// TODO: better way to specify a unix socket path (http+unix: / unix [socket] ?)
	listen := c.Val()

	// find API token if required
	var token, certFile, keyFile string
	for c.NextBlock() {
		switch c.Val() {
		case "token":
			if c.NextArg() {
				token = c.Val()
				break
			}
			return plugin.Error("trapi", c.ArgErr())
		case "certFile":
			if c.NextArg() {
				certFile = c.Val()
				break
			}
			return plugin.Error("trapi", c.ArgErr())
		case "keyFile":
			if c.NextArg() {
				keyFile = c.Val()
				break
			}
			return plugin.Error("trapi", c.ArgErr())
		default:
			return plugin.Error("trapi", c.ArgErr())
		}
	}

	if c.NextArg() { // no further arguments expected
		return plugin.Error("trapi", c.ArgErr())
	}

	if token == "" {
		return plugin.Error("trapi", errors.New("API token required"))
	}

	tls := false
	if certFile != "" || keyFile != "" {
		if certFile == "" {
			return plugin.Error("trapi", errors.New("specified keyFile without certFile"))
		} else if keyFile == "" {
			return plugin.Error("trapi", errors.New("specified certFile without keyFile"))
		} else {
			tls = true
		}
	} else {
		tls = false
	}

	api := API{Token: token}
	if strings.HasPrefix(listen, string(os.PathSeparator)) {
		socketPath, err := filepath.Abs(listen)
		if err != nil {
			log.Fatalf("Unable to get absolute path: %v", err)
		}
		// socket is e.g. "/run/coredns.trapi.sock"
		_ = os.Remove(socketPath) // TODO: check for existence first; possible foot-gun
		// TODO: make sure socket has correct (i.e. 600) permissions
		unixListener, err := net.Listen("unix", socketPath)
		if err != nil {
			log.Fatalf("unixListener failed: %s", err)
		}
		defer unixListener.Close()
		go func() {
			log.Infof("listening on socket %s", socketPath)
			var err error
			if tls {
				err = http.ServeTLS(unixListener, &api, certFile, keyFile)
			} else {
				err = http.Serve(unixListener, &api)
			}
			if err != nil {
				log.Fatalf("failed setting up HTTP listener: %s", err)
			}
		}()
	} else {
		// TODO: check how the grpc plugin does this
		go func() {
			log.Infof("listening on %s", listen)
			var err error
			if tls {
				// yes, the order here is different to `http.ServeTLS` m(
				err = http.ListenAndServeTLS(listen, certFile, keyFile, &api)
			} else {
				err = http.ListenAndServe(listen, &api)
			}
			if err != nil {
				log.Fatalf("failed setting up HTTP listener: %s", err)
			}
		}()
	}

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	config := dnsserver.GetConfig(c)
	config.AddPlugin(func(next plugin.Handler) plugin.Handler {
		// extract NOTIFY targets from "transfer to" configuration of downstream plugin(s)
		// this must be done here so that the downstream plugins have had their setup routines executed
		for _, handler := range config.Handlers() {
			switch h := handler.(type) {
			case file.File:
				for _, origin := range h.Zones.Names {
					z := h.Zones.Z[origin]
					transferToMap[origin] = append(transferToMap[origin], z.TransferTo...)
				}
			case kubernetes.Kubernetes:
				transferToMap["."] = append(transferToMap["."], h.TransferTo...)
			}
		}

		if len(transferToMap) == 0 {
			log.Warning("no transfer to statements found; cannot send NOTIFY to slaves!")
		}
		return Trapi{Next: next}
	})

	// All OK, return a nil error.
	return nil
}
