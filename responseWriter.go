package trapi

import (
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

type TRRResponseWriter struct {
	dns.ResponseWriter
	state request.Request // needed to filter (again) on return-path
}

// TODO: figure out if/how to handle multiple question/answer/... sections

// WriteMsg implements the dns.ResponseWriter interface.
func (r *TRRResponseWriter) WriteMsg(res *dns.Msg) error {
	state := r.state
	log.Debugf("@TRRRW.WriteMsg: %s %s\n", state.Type(), state.Name())
	if res.Rcode != dns.RcodeSuccess { // only intercept successful (i.e. existing) zones
		return r.ResponseWriter.WriteMsg(res)
	}

	TRRs.RLock()
	zone := plugin.Zones(TRRs.Names).Matches(state.Name())
	if zone == "" { // not found
		return r.ResponseWriter.WriteMsg(res)
	}

	trrs, ok := TRRs.internal[zone]
	if !ok || trrs == nil {
		return r.ResponseWriter.WriteMsg(res)
	}

	// filter resource records to inject into response
	// and count the number of times we change the record
	var toInject []dns.RR
	var serialChangeCounter uint32 = 0
	for _, trr := range trrs {
		log.Infof("Filtering trr: %v", trr)
		if !trr.Expired() { // if RR valid, increment serial by 1 and inject
			log.Info("is not expired :)")
			serialChangeCounter += 1
			if trr.NameMatches(state) {
				log.Info("matches filter =D")
				toInject = append(toInject, trr)
			}
		} else { // if RR expired, increment serial by 2 and don't inject
			log.Info("is expired :(")
			serialChangeCounter += 2
		}
	}
	TRRs.RUnlock()

	// add the serial increment to the response serial
	incrementSerial(res.Answer, serialChangeCounter)

	log.Infof("Intercepting %s response...", state.Type())
	// inject only QType/Name filtered/matching TRRs
	switch state.QType() {
	case dns.TypeA:
	case dns.TypeAAAA:
	case dns.TypeTXT:
		res.Answer = injectTRRs(res.Answer, toInject, false)
		break
	case dns.TypeSOA:
		break // do nothing
	case dns.TypeIXFR: // TODO: handle IXFR correctly (upstream doesn't emit IXFR responses yet)
	case dns.TypeAXFR:
		res.Answer = injectTRRs(res.Answer, toInject, true)
		break
	default:
		// we don't support any other query types
		log.Warning("Unsupported response type")
	}

	return r.ResponseWriter.WriteMsg(res)
}

func injectTRRs(answer []dns.RR, inject []dns.RR, isXFR bool) []dns.RR {
	var endSOA dns.RR // only used for XFR
	// By default insert TRRs at the end (i.e. append)
	insertIdx := len(answer)
	if isXFR {
		// An XFR answer starts with an SOA and (usually) ends with an SOA.
		// This means we have to insert TRRs before the end SOA.
		// If there is no end SOA we just insert before the last RR.
		insertIdx -= 1
		if insertIdx == 0 { // edge case: empty zone and no end SOA
			insertIdx = 1 // need to preserve the initial SOA
		} else {
			endSOA = answer[len(answer)-1]
		}
	}

	// inject TRRs, possibly truncating the endSOA
	answer = append(answer[:insertIdx], inject...)
	// re-append endSOA if needed
	if endSOA != nil {
		answer = append(answer, endSOA)
	}

	return answer
}

func incrementSerial(in []dns.RR, amount uint32) {
	var targetSerial uint32 = 0
	for i, r := range in {
		switch r.Header().Rrtype {
		case dns.TypeSOA:
			// makes a copy of the RR because otherwise we end up modifying the underlying
			// backend's in-memory representation (e.g. of the file backend) and count up
			// again and again. Note that the file backend reloads every 5s by default, which
			// would lead to very strange errors here if we'd continuously increment that
			// very short living in-memory representation
			copied := dns.Copy(r)
			s, _ := copied.(*dns.SOA)
			if targetSerial == 0 {
				targetSerial = s.Serial + amount
			}
			s.Serial = targetSerial
			in[i] = copied
		}
	}
}

// Write implements the dns.ResponseWriter interface.
func (r *TRRResponseWriter) Write(buf []byte) (int, error) {
	// Should we pack and unpack here to fiddle with the packet... Not likely.
	// Always warn, since this is only injected if there exists a TRR for this zone
	log.Warning("TRRResponseWriter called with Write: not appending some temporary records")
	n, err := r.ResponseWriter.Write(buf)
	return n, err
}