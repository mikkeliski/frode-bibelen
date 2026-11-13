package main

import (
	"math"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"
)

// begrenser er en enkel token bucket per klient (IP-adresse, eller /64 for IPv6).
// Alt ligger bare i minnet og ryddes bort når klienten har vært stille en stund.
type begrenser struct {
	mu     sync.Mutex
	botter map[string]*botte
	rate   float64 // nye forespørsler per sekund
	burst  float64 // maks forespørsler i en kort topp
	maks   int     // maks antall klienter som spores samtidig
	na     func() time.Time
}

type botte struct {
	tokens float64
	sist   time.Time
}

func nyBegrenser(perMinutt, burst, maks int) *begrenser {
	return &begrenser{
		botter: make(map[string]*botte),
		rate:   float64(perMinutt) / 60,
		burst:  float64(burst),
		maks:   maks,
		na:     time.Now,
	}
}

// tillat sier om klienten kan få en forespørsel til nå.
func (b *begrenser) tillat(nokkel string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	na := b.na()
	bt, ok := b.botter[nokkel]
	if !ok {
		if len(b.botter) >= b.maks {
			b.ryddLaast(na)
			if len(b.botter) >= b.maks {
				return false
			}
		}
		bt = &botte{tokens: b.burst, sist: na}
		b.botter[nokkel] = bt
	}
	bt.tokens = math.Min(b.burst, bt.tokens+na.Sub(bt.sist).Seconds()*b.rate)
	bt.sist = na
	if bt.tokens < 1 {
		return false
	}
	bt.tokens--
	return true
}

// ryddLaast fjerner klienter som har fått fylt opp bøtta igjen (krever b.mu).
func (b *begrenser) ryddLaast(na time.Time) {
	full := time.Duration(b.burst / b.rate * float64(time.Second))
	for k, bt := range b.botter {
		if na.Sub(bt.sist) > full {
			delete(b.botter, k)
		}
	}
}

func (b *begrenser) rydd() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ryddLaast(b.na())
}

// klientNokkel finner klientens adresse. X-Forwarded-For brukes bare når
// serveren eksplisitt kjører bak en proxy, og da bare siste (proxyens) verdi.
func klientNokkel(r *http.Request, bakProxy bool) string {
	var ip string
	if bakProxy {
		if v := r.Header.Values("X-Forwarded-For"); len(v) > 0 {
			deler := strings.Split(v[len(v)-1], ",")
			ip = strings.TrimSpace(deler[len(deler)-1])
		}
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		host, _, splitErr := net.SplitHostPort(r.RemoteAddr)
		if splitErr != nil {
			host = r.RemoteAddr
		}
		if addr, err = netip.ParseAddr(host); err != nil {
			return "ukjent"
		}
	}
	addr = addr.Unmap()
	if addr.Is6() {
		p, _ := addr.Prefix(64)
		return p.String()
	}
	return addr.String()
}
