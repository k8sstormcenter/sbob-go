package profilediff

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strconv"
	"sync"

	"github.com/k8sstormcenter/sbob-go/profile"
)

type Fingerprint struct {
	UID             string
	ResourceVersion string
}

func (f Fingerprint) complete() bool {
	return f.UID != "" && f.ResourceVersion != ""
}

type Stamp struct {
	shadow  Fingerprint
	bound   []Fingerprint
	matcher string
}

func NewStamp(shadow Fingerprint, bound []Fingerprint, matcher string) Stamp {
	s := Stamp{shadow: shadow, matcher: matcher}
	if len(bound) > 0 {
		s.bound = make([]Fingerprint, len(bound))
		copy(s.bound, bound)
	}
	return s
}

func (s Stamp) WithMatcher(matcher string) Stamp {
	s.matcher = matcher
	return s
}

func (s Stamp) Complete() bool {
	if !s.shadow.complete() || s.matcher == "" {
		return false
	}
	for _, b := range s.bound {
		if !b.complete() {
			return false
		}
	}
	return true
}

func (s Stamp) Digest() string {
	h := sha256.New()
	field := func(v string) {
		io.WriteString(h, strconv.Itoa(len(v)))
		io.WriteString(h, ":")
		io.WriteString(h, v)
	}
	field(s.shadow.UID)
	field(s.shadow.ResourceVersion)
	field(strconv.Itoa(len(s.bound)))
	for _, b := range s.bound {
		field(b.UID)
		field(b.ResourceVersion)
	}
	field(s.matcher)
	return hex.EncodeToString(h.Sum(nil))
}

func (s Stamp) Equal(other Stamp) bool {
	if !s.Complete() || !other.Complete() {
		return false
	}
	return s.Digest() == other.Digest()
}

func MustRecompute(prev, next Stamp) bool {
	return !prev.Equal(next)
}

type Cache struct {
	mu    sync.RWMutex
	stamp Stamp
	value *profile.Spec
	set   bool
}

func NewCache() *Cache {
	return &Cache{}
}

func (c *Cache) Store(s Stamp, residual *profile.Spec) {
	if !s.Complete() {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stamp, c.value, c.set = s, residual, true
}

func (c *Cache) Load(s Stamp) (*profile.Spec, bool) {
	if !s.Complete() {
		return nil, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.set || !c.stamp.Equal(s) {
		return nil, false
	}
	return c.value, true
}

func (c *Cache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stamp, c.value, c.set = Stamp{}, nil, false
}

func (c *Cache) ResidualAt(s Stamp, shadow, bound *profile.Spec) *profile.Spec {
	if cached, ok := c.Load(s); ok {
		return cached
	}
	residual := Residual(shadow, bound)
	c.Store(s, residual)
	return residual
}
