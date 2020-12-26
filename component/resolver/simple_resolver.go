package resolver

import (
	"net"
	"sync"
	"time"
)

var timeout = time.Second * 60

type dnsResolver struct {
	recTime time.Time
	ipv4    net.IP
	ipv6    net.IP
	ip      net.IP
	host    string
	mu      sync.Mutex
	blocked bool
}

func (r *dnsResolver) resolve() error {
	r.mu.Lock()
	if r.blocked || r.recTime.Add(time.Second*3).After(time.Now()) {
		r.mu.Unlock()
		return nil
	}
	r.blocked = true
	r.recTime = time.Now()
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.blocked = false
		r.recTime = time.Now()
		r.mu.Unlock()
	}()

	ipAddrs, err := net.LookupIP(r.host)
	if err != nil {
		return err
	}
	ipAddr, err := net.ResolveIPAddr("ip", r.host)
	if err != nil {
		return err
	}
	r.ip = ipAddr.IP
	for _, ip := range ipAddrs {
		if ip := ip.To4(); ip != nil {
			r.ipv4 = ip
			break
		}
	}
	for _, ip := range ipAddrs {
		if ip.To4() == nil {
			r.ipv6 = ip
			break
		}
	}
	return nil
}

func (r *dnsResolver) ResolveIPv4() (net.IP, error) {
	if r.recTime.Add(timeout).Before(time.Now()) {
		err := r.resolve()
		if err != nil {
			return nil, err
		}
		return r.ipv4, nil
	}
	go r.resolve()
	return r.ipv4, nil
}

func (r *dnsResolver) ResolveIPv6() (net.IP, error) {
	if r.recTime.Add(timeout).Before(time.Now()) {
		err := r.resolve()
		if err != nil {
			return nil, err
		}
		return r.ipv6, nil
	}
	go r.resolve()
	return r.ipv6, nil
}

func (r *dnsResolver) ResolveIP() (net.IP, error) {
	if r.recTime.Add(timeout).Before(time.Now()) {
		err := r.resolve()
		if err != nil {
			return nil, err
		}
		return r.ip, nil
	}
	go r.resolve()
	return r.ip, nil
}

type DnsResolver map[string]*dnsResolver

func (r DnsResolver) getResolver(host string) *dnsResolver {
	resolver, ok := r[host]
	if !ok {
		resolver = &dnsResolver{host: host}
		r[host] = resolver
	}
	return resolver
}

func (r DnsResolver) ResolveIPv4(host string) (net.IP, error) {
	ip := net.ParseIP(host)
	if ip != nil {
		return ip, nil
	}
	return r.getResolver(host).ResolveIPv4()
}

func (r DnsResolver) ResolveIPv6(host string) (net.IP, error) {
	ip := net.ParseIP(host)
	if ip != nil {
		return ip, nil
	}
	return r.getResolver(host).ResolveIPv6()
}

func (r DnsResolver) ResolveIP(host string) (net.IP, error) {
	ip := net.ParseIP(host)
	if ip != nil {
		return ip, nil
	}
	return r.getResolver(host).ResolveIP()
}

var simpleResolver = make(DnsResolver)
