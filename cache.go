package cache

import (
	"container/list"
	"container/ring"
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
	LFU
	CLOCK
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
	case LFU:
		return NewLFUPolicy()
	case CLOCK:
		return NewCLOCKPolicy()
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

// FIFO
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

// LRU
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

// CLOCK
type ClockPolicy struct {
	list      *CircularList
	keyNode   map[CacheKey]*ring.Ring
	clockHand *ring.Ring
}
type ClockItem struct {
	key CacheKey
	bit bool
}

func NewCLOCKPolicy() CachePolicy {
	policy := &ClockPolicy{}
	policy.keyNode = make(map[CacheKey]*ring.Ring)
	policy.list = &CircularList{}
	policy.clockHand = nil
	return policy
}

func (p *ClockPolicy) Victim() CacheKey {
	var victimKey CacheKey
	var nodeItem *ClockItem
	for {
		currentNode := (*p.clockHand)
		nodeItem = currentNode.Value.(*ClockItem)
		if nodeItem.bit {
			nodeItem.bit = false
			currentNode.Value = nodeItem
			p.clockHand = currentNode.Next()
		} else {
			victimKey = nodeItem.key
			p.list.Move(p.clockHand.Prev())
			p.clockHand = nil
			p.list.Remove(&currentNode)
			delete(p.keyNode, victimKey)
			return victimKey
		}
	}
}

func (p *ClockPolicy) Add(key CacheKey) {
	node := p.list.Append(&ClockItem{key, true})
	if p.clockHand == nil {
		p.clockHand = node
	}
	p.keyNode[key] = node
}

func (p *ClockPolicy) Remove(key CacheKey) {
	node, ok := p.keyNode[key]
	if !ok {
		return
	}

	if p.clockHand == node {
		p.clockHand = p.clockHand.Prev()
	}
	p.list.Remove(node)
	delete(p.keyNode, key)
}

func (p *ClockPolicy) Access(key CacheKey) {
	node, ok := p.keyNode[key]
	if !ok {
		return
	}
	node.Value = &ClockItem{key, true}
}

// LFU

type Frequency int

type LFUItem struct {
	frequency Frequency
	key       CacheKey
}

type LFUPolicy struct {
	freqList     map[Frequency]*list.List
	keyNode      map[CacheKey]*list.Element
	minFrequency Frequency
}

func NewLFUPolicy() CachePolicy {
	policy := &LFUPolicy{}
	policy.keyNode = make(map[CacheKey]*list.Element)
	policy.freqList = make(map[Frequency]*list.List)
	policy.minFrequency = 1
	return policy
}

func (p *LFUPolicy) Victim() CacheKey {
	fList := p.freqList[p.minFrequency]
	element := fList.Back()
	fList.Remove(element)
	delete(p.keyNode, element.Value.(LFUItem).key)
	return element.Value.(LFUItem).key
}

func (p *LFUPolicy) Add(key CacheKey) {
	_, ok := p.freqList[1]
	if !ok {
		p.freqList[1] = list.New()
	}

	node := p.freqList[1].PushFront(LFUItem{1, key})
	p.keyNode[key] = node
	p.minFrequency = 1
}

func (p *LFUPolicy) Remove(key CacheKey) {
	p.remove(key)
}

func (p *LFUPolicy) Access(key CacheKey) {
	node := p.remove(key)

	frequency := node.Value.(LFUItem).frequency
	_, ok := p.freqList[frequency+1]
	if !ok {
		p.freqList[frequency+1] = list.New()
	}

	node = p.freqList[frequency+1].PushFront(LFUItem{frequency + 1, key})
	p.keyNode[key] = node
}

func (p *LFUPolicy) remove(key CacheKey) *list.Element {
	node := p.keyNode[key]
	frequency := node.Value.(LFUItem).frequency

	p.freqList[frequency].Remove(node)
	delete(p.keyNode, key)

	if p.freqList[frequency].Len() == 0 {
		delete(p.freqList, frequency)
		if p.minFrequency == frequency {
			p.minFrequency++
		}
	}

	return node
}
