package proxy

import (
	"bufio"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
)

// Pool manages a list of proxies with round-robin rotation
type Pool struct {
	proxies []*url.URL
	index   atomic.Uint64
}

// NewPool creates a proxy pool from a list of proxy URL strings
func NewPool(proxyURLs []string) (*Pool, error) {
	pool := &Pool{}
	for _, raw := range proxyURLs {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		parsed, err := url.Parse(raw)
		if err != nil {
			continue
		}
		pool.proxies = append(pool.proxies, parsed)
	}
	return pool, nil
}

// NewPoolFromFile reads proxy URLs from a file
func NewPoolFromFile(path string) (*Pool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return NewPool(lines)
}

// Next returns the next proxy in round-robin order
func (p *Pool) Next() *url.URL {
	if len(p.proxies) == 0 {
		return nil
	}
	idx := p.index.Add(1) - 1
	return p.proxies[idx%uint64(len(p.proxies))]
}

// Random returns a random proxy from the pool
func (p *Pool) Random() *url.URL {
	if len(p.proxies) == 0 {
		return nil
	}
	return p.proxies[rand.Intn(len(p.proxies))]
}

// Len returns the number of proxies in the pool
func (p *Pool) Len() int {
	return len(p.proxies)
}
