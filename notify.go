package trapi

import (
	"fmt"
	"github.com/coredns/coredns/plugin/pkg/rcode"
	"github.com/miekg/dns"
)

func asyncNotify(origin string) {
	targets := make([]string, len(transferToMap["."])+len(transferToMap[origin]))
	copy(targets, transferToMap["."])
	for _, target := range transferToMap[origin] {
		if !contains(targets, target) {
			targets = append(targets, target)
		}
	}
	go func() {
		if err := notify(origin, targets); err != nil {
			log.Error(err)
		}
	}()
}

func contains(slice []string, e string) bool {
	for _, se := range slice {
		if se == e {
			return true
		}
	}
	return false
}

////////// ////////// ////////// //////////
//////////   the following code  //////////
//////////    is copied from     //////////
////////// plugin/file/notify.go //////////
////////// ////////// ////////// //////////
// TODO: consider using the public (*Zone).Notify instead

// notify sends notifies to the configured remote servers. It will try up to three times
// before giving up on a specific remote. We will sequentially loop through "to"
// until they all have replied (or have 3 failed attempts).
func notify(zone string, to []string) error {
	m := new(dns.Msg)
	m.SetNotify(zone)
	c := new(dns.Client)

	for _, t := range to {
		if t == "*" {
			continue
		}
		if err := notifyAddr(c, m, t); err != nil {
			log.Error(err.Error())
		} else {
			log.Infof("Sent notify for zone %q to %q", zone, t)
		}
	}
	return nil
}

func notifyAddr(c *dns.Client, m *dns.Msg, s string) error {
	var err error

	code := dns.RcodeServerFailure
	for i := 0; i < 3; i++ {
		ret, _, err := c.Exchange(m, s)
		if err != nil {
			continue
		}
		code = ret.Rcode
		if code == dns.RcodeSuccess {
			return nil
		}
	}
	if err != nil {
		return fmt.Errorf("notify for zone %q was not accepted by %q: %q", m.Question[0].Name, s, err)
	}
	return fmt.Errorf("notify for zone %q was not accepted by %q: rcode was %q", m.Question[0].Name, s, rcode.ToString(code))
}
