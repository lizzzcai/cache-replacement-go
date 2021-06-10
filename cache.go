package cache

import (
	"container/list"
	"errors"
)

type CacheKey string

type CacheData map[CacheKey]string

type Cache struct {
	maxSize int
	size    int
	policy  CachePolicy
	data    CacheData
}

type PolicyType int

const (
	FIFO PolicyType = 1 << iota
	LRU
)

// Victim runs the policy algorithm and elects a CacheKey , called victim, for removal;
// Add makes a cache key eligible for eviction
// Remove makes a cache key no longer eligible for eviction
// Access indicates to the cache policy that a cache key was accessed. This provides additional information to the cache replacement algorithm to make its decision
type CachePolicy interface {
	Victim() CacheKey
	Add(CacheKey)
	Remove(CacheKey)
	Access(CacheKey)
}

func GetCachePolicy(policy PolicyType) CachePolicy {
	switch policy {
	case FIFO:
		return NewFIFOPolicy()
	case LRU:
		return NewLRUPolicy()
	default:
		return NewFIFOPolicy()
	}
}

func (c *Cache) Put(key CacheKey, value string) {
	if c.size == c.maxSize {
		victimKey := c.policy.Victim()
		delete(c.data, victimKey)
		c.size -= 1
	}
	c.policy.Add(key)
	c.data[key] = value
	c.size += 1
}

func (c *Cache) Get(key CacheKey) (*string, error) {
	if value, ok := c.data[key]; ok {
		c.policy.Access(key)
		return &value, nil
	}

	return nil, errors.New("key not found")
}

func NewCache(maxSize int, policy PolicyType) *Cache {
	cache := &Cache{}
	cache.maxSize = maxSize
	cache.policy = GetCachePolicy(policy)
	cache.data = make(CacheData, maxSize)
	return cache
}

// Replacement Policies

type FIFOPolicy struct {
	list    *list.List
	keyNode map[CacheKey]*list.Element
}

func NewFIFOPolicy() CachePolicy {
	policy := &FIFOPolicy{}
	policy.list = list.New()
	policy.keyNode = make(map[CacheKey]*list.Element)
	return policy
}

func (p *FIFOPolicy) Victim() CacheKey {
	element := p.list.Back()
	p.list.Remove(element)
	delete(p.keyNode, element.Value.(CacheKey))
	return element.Value.(CacheKey)
}

func (p *FIFOPolicy) Add(key CacheKey) {
	node := p.list.PushFront(key)
	p.keyNode[key] = node
}

func (p *FIFOPolicy) Remove(key CacheKey) {
	node, ok := p.keyNode[key]
	if !ok {
		return
	}
	p.list.Remove(node)
	delete(p.keyNode, key)
}

func (p *FIFOPolicy) Access(key CacheKey) {}

type LRUPolicy struct {
	list    *list.List
	keyNode map[CacheKey]*list.Element
}

func NewLRUPolicy() CachePolicy {
	policy := &LRUPolicy{}
	policy.list = list.New()
	policy.keyNode = make(map[CacheKey]*list.Element)
	return policy
}

func (p *LRUPolicy) Victim() CacheKey {
	element := p.list.Back()
	p.list.Remove(element)
	delete(p.keyNode, element.Value.(CacheKey))
	return element.Value.(CacheKey)
}

func (p *LRUPolicy) Add(key CacheKey) {
	node := p.list.PushFront(key)
	p.keyNode[key] = node
}

func (p *LRUPolicy) Remove(key CacheKey) {
	node, ok := p.keyNode[key]
	if !ok {
		return
	}
	p.list.Remove(node)
	delete(p.keyNode, key)
}

func (p *LRUPolicy) Access(key CacheKey) {
	p.Remove(key)
	p.Add(key)
}
