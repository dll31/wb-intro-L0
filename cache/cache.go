package cache

import (
	"sync"
)

type Cache struct {
	sync.RWMutex
	// defaultExpiration time.Duration
	// cleanupInterval   time.Duration
	items map[string]Item
}

type Item struct {
	Value interface{}
	// Created    time.Time
	// Expiration int64
}

func New() *Cache {

	its := make(map[string]Item)

	cache := Cache{
		items: its,
	}

	return &cache
}

func (c *Cache) Set(key string, value interface{}) {

	c.Lock()
	defer c.Unlock()

	c.items[key] = Item{
		Value: value,
	}
}

func (c *Cache) Get(key string) interface{} {
	c.RLock()
	defer c.RUnlock()

	item := c.items[key]

	return item.Value
}
