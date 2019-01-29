package trapi

import (
	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
	"net/http"
	"strconv"
	"time"
)

// TODO: maybe make API json
// TODO: define what happens to collisions (shadow, append, not allowed?; currently always appends)
// TODO: maybe add permission system (allows TypeA/AAA/TXT right now)
// TODO: add tests
type API struct {
	Token string
}

func (f *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(2048); err != nil {
		log.Error("Received invalid API call.")
	} else {
		log.Debugf("Received API call from %s", r.UserAgent())
	}

	if r.PostFormValue("token") != f.Token {
		log.Warning("Invalid API call token.")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	origin := plugin.Name(r.PostFormValue("origin")).Normalize()
	if origin == "" {
		log.Warning("Invalid API call origin.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	now := time.Now()
	ttl := uint32(0)
	if val := r.PostFormValue("ttl"); val != "" {
		if aTtl, err := strconv.ParseUint(val, 10, 32); err == nil {
			ttl = uint32(aTtl)
		}
	}

	var trrs []TemporaryResourceRecord
	var maxTtl = ttl
	for key, values := range r.PostForm {
		if len(values) == 0 {
			continue
		}
		switch key {
		case "rr":
			for _, rrstr := range values {
				rr, err := dns.NewRR(rrstr)
				if err != nil || rr == nil {
					log.Warningf("Invalid rr in API call: %s", rrstr)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				log.Infof("Received new temporary RR for '%s': %s", origin, rr.String())
				var trr TemporaryResourceRecord
				if ttl > 0 {
					trr = TemporaryResourceRecord{RR: rr, Created: now, Ttl: ttl}
				} else {
					trr = TemporaryResourceRecord{RR: rr, Created: now, Ttl: rr.Header().Ttl}
					if rr.Header().Ttl > maxTtl {
						maxTtl = rr.Header().Ttl
					}
				}
				trrs = append(trrs, trr)
			}
		case "token": // already handled
		case "origin": // already handled
		case "ttl": // already handled
		default:
			log.Warningf("Received unknown POST data: %s => %v", key, values)
		}
	}

	if origin == "" || trrs == nil || ttl == 0 {
		log.Warning("Invalid API call.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	TRRs.Lock()
	if TRRs.internal[origin] == nil {
		TRRs.internal[origin] = trrs
	} else { // append new RRs to list
		TRRs.internal[origin] = append(TRRs.internal[origin], trrs...)
	}
	TRRs.Names = append(TRRs.Names, origin)
	TRRs.Unlock()

	// confirm to api client
	w.WriteHeader(http.StatusCreated)

	// trigger NOTIFY to slaves
	asyncNotify(origin)

	// schedule expiry NOTIFY after maxTtl + 1 minute
	_ = time.AfterFunc(time.Duration(time.Second*time.Duration(maxTtl)+time.Minute), func() {
		asyncNotify(origin)
	})
}
