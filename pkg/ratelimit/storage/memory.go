package storage

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

var ValueNotFoundByKey = errors.New("value is not found by key")

type MemoryCache struct {
	dataMu sync.RWMutex
	data   map[string][]byte

	timerMu sync.RWMutex
	timers  map[string]*time.Timer
}

// NewMemoryCache create new storage in memory
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		data:   make(map[string][]byte),
		timers: make(map[string]*time.Timer),
	}
}

// Has check is set value by key
func (c *MemoryCache) Has(ctx context.Context, key []byte) bool {
	c.dataMu.RLock()
	defer c.dataMu.RUnlock()

	_, ok := c.data[string(key)]
	return ok
}

// Inc value by key
func (c *MemoryCache) Inc(ctx context.Context, key []byte, ttl *uint64) (int64, error) {
	c.dataMu.Lock()
	defer c.dataMu.Unlock()

	strKey := string(key)
	if _, ok := c.data[string(key)]; !ok {
		c.data[strKey] = []byte("1")
		return int64(1), ValueNotFoundByKey
	}

	valInt, err := c.sliceByteToInt64(key)
	if err != nil {
		return int64(0), err
	}
	valInt++
	c.data[strKey] = []byte(strconv.FormatInt(valInt, 10))

	c.initCancel(ctx, strKey, ttl)

	return valInt, nil
}

// Decr decrement value by key
func (c *MemoryCache) Decr(ctx context.Context, key []byte, ttl *uint64) (int64, error) {
	c.dataMu.Lock()
	defer c.dataMu.Unlock()
	strKey := string(key)

	if _, ok := c.data[strKey]; !ok {
		c.data[strKey] = []byte("-1")
		return int64(-1), ValueNotFoundByKey
	}

	valInt, err := c.sliceByteToInt64(key)
	if err != nil {
		return int64(0), err
	}
	valInt--
	c.data[strKey] = []byte(strconv.FormatInt(valInt, 10))

	c.initCancel(ctx, strKey, ttl)

	return valInt, nil
}

// Get value by key
func (c *MemoryCache) Get(ctx context.Context, key []byte) ([]byte, error) {
	c.dataMu.Lock()
	defer c.dataMu.Unlock()

	if _, ok := c.data[string(key)]; !ok {
		return nil, ValueNotFoundByKey
	}

	return c.data[string(key)], nil
}

// Set value by key
func (c *MemoryCache) Set(ctx context.Context, key []byte, value []byte, ttl *uint64) error {
	if len(key) == 0 {
		return fmt.Errorf("key is empty")
	}

	c.dataMu.Lock()
	defer c.dataMu.Unlock()

	strKey := string(key)
	c.data[strKey] = value

	c.initCancel(ctx, strKey, ttl)
	return nil
}

// Del value by key
func (c *MemoryCache) Del(ctx context.Context, list ...[]byte) error {
	c.dataMu.Lock()
	defer c.dataMu.Unlock()

	for _, key := range list {
		strKey := string(key)
		delete(c.data, strKey)
		err := c.callCancel(key)
		if err != nil {
			return err
		}
	}

	return nil
}

// Clear all
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.dataMu.Lock()
	defer c.dataMu.Unlock()

	c.timerMu.Lock()
	defer c.timerMu.Unlock()
	for _, t := range c.timers {
		if !t.Stop() {
			<-t.C
		}
	}

	c.data = make(map[string][]byte)
	c.timers = make(map[string]*time.Timer)

	return nil
}

// callCancel call cancel function that cancel removing by ttl
func (c *MemoryCache) callCancel(key []byte) error {
	c.timerMu.Lock()
	defer c.timerMu.Unlock()

	strKey := string(key)
	if t, ok := c.timers[strKey]; ok {
		if !t.Stop() {
			<-t.C
		}

		delete(c.timers, strKey)
	}

	return nil
}

// sliceByteToInt64 modify slice bytes to int64. Used in Inc and Decr
func (c *MemoryCache) sliceByteToInt64(key []byte) (int64, error) {
	val, ok := c.data[string(key)]
	if !ok {
		return int64(0), fmt.Errorf("value by key %s is not found", string(key))
	}

	valInt, err := strconv.ParseInt(string(val), 10, 64)
	if err != nil {
		return int64(0), fmt.Errorf("value by key %s is not integer", string(key))
	}

	return valInt, nil
}

func (c *MemoryCache) initCancel(ctx context.Context, strKey string, ttl *uint64) {
	if ttl == nil || *ttl <= 0 {
		return
	}

	c.timerMu.Lock()
	defer c.timerMu.Unlock()
	if _, ok := c.timers[strKey]; ok {
		return
	}
	t := time.NewTimer(time.Duration(*ttl) * time.Second)
	c.timers[strKey] = t

	go func() {
		select {
		case <-t.C:
			c.dataMu.Lock()
			delete(c.data, strKey)
			c.dataMu.Unlock()

			c.timerMu.Lock()
			delete(c.timers, strKey)
			c.timerMu.Unlock()
		case <-ctx.Done():
			if !t.Stop() {
				<-t.C
			}

			c.timerMu.Lock()
			delete(c.timers, strKey)
			c.timerMu.Unlock()
			return
		}
	}()
}
