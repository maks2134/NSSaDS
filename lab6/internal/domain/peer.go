package domain

import (
	"net"
	"sync"
	"time"
)

type Peer struct {
	IP       net.IP
	Nick     string
	Mode     Mode
	LastSeen time.Time
	Ignored  bool
}

type PeerRegistry struct {
	mu      sync.RWMutex
	peers   map[string]*Peer
	ignored map[string]bool
}

func NewPeerRegistry() *PeerRegistry {
	return &PeerRegistry{
		peers:   make(map[string]*Peer),
		ignored: make(map[string]bool),
	}
}

func (r *PeerRegistry) Upsert(ip net.IP, nick string, mode Mode, now time.Time) {
	key := ip.String()
	r.mu.Lock()
	defer r.mu.Unlock()
	ignored := r.ignored[key]
	p, ok := r.peers[key]
	if !ok {
		r.peers[key] = &Peer{IP: ip, Nick: nick, Mode: mode, LastSeen: now, Ignored: ignored}
		return
	}
	p.Nick = nick
	p.Mode = mode
	p.LastSeen = now
	p.Ignored = ignored
}

func (r *PeerRegistry) Remove(ip net.IP) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.peers, ip.String())
}

func (r *PeerRegistry) SetIgnored(ip net.IP, ignored bool) {
	key := ip.String()
	r.mu.Lock()
	defer r.mu.Unlock()
	if ignored {
		r.ignored[key] = true
	} else {
		delete(r.ignored, key)
	}
	if p, ok := r.peers[key]; ok {
		p.Ignored = ignored
	}
}

func (r *PeerRegistry) IsIgnored(ip net.IP) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ignored[ip.String()]
}

func (r *PeerRegistry) Expire(before time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, p := range r.peers {
		if p.LastSeen.Before(before) {
			delete(r.peers, key)
		}
	}
}

func (r *PeerRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.peers = make(map[string]*Peer)
}

func (r *PeerRegistry) List(mode Mode) []Peer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Peer, 0, len(r.peers))
	for _, p := range r.peers {
		if p.Mode != mode {
			continue
		}
		out = append(out, *p)
	}
	return out
}
