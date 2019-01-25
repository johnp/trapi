package trapi

import (
	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
	"net/http"
	"strconv"
	"time"
)

// TODO: maybe make API json
// TODO: improve security (HTTPS/caddy?)
// TODO: define what happens to collisions (shadow, append, not allowed?; currently always appends)
// TODO: maybe add permission system (allows TypeA/AAA/TXT right now)
type API struct {
	Token string
}

func (f *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Error("Received invalid API call.")
	} else {
		log.Infof("Received API call from %s", r.UserAgent())
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
				}
				trrs = append(trrs, trr)
			}
			break
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

	TRRs.RLock()
	for _, trrs := range TRRs.internal[origin] {
		log.Infof("RR: %+v", trrs.String())
		log.Infof("Created: %s", trrs.Created)
		log.Infof("Ttl: %v", trrs.Ttl)
	}
	TRRs.RUnlock()

	// TODO: trigger NOTIFY to slaves
	// Options:
	//  (1) somehow call the file plugins notify function
	//  (2) require a "transfer to" config option for trapi
	//  (3) ?
}
