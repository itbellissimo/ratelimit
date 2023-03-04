package storage

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

var (
	zero    = uint64(0)
	two     = uint64(2)
	three   = uint64(3)
	ten     = uint64(10)
	fifteen = uint64(15)
	twenty  = uint64(20)
)

// TestNewMemoryCache test NewMemoryCache function
func Test_NewMemoryCache(t *testing.T) {

	mem := NewMemoryCache()

	assert.Equal(t, fmt.Sprintf("%T", &MemoryCache{}), fmt.Sprintf("%T", mem))
	assert.NotNil(t, mem)
	assert.NotNil(t, mem.timers)
	assert.NotNil(t, mem.data)
	assert.Len(t, mem.timers, 0)
	assert.Len(t, mem.data, 0)
}

// TestMemoryCache_Set test Set function
func TestMemoryCache_Set(t *testing.T) {
	type setDataTest struct {
		memoryCache *MemoryCache
		ctx         context.Context
		key         []byte
		value       []byte
		ttl         uint64
		err         error
		timeout     uint64
	}

	var setDataTests = []setDataTest{
		{
			memoryCache: NewMemoryCache(),
			ctx:         context.Background(),
			key:         []byte("set_key1"),
			value:       []byte("value1"),
			ttl:         1,
			err:         nil,
		},
		{
			memoryCache: NewMemoryCache(),
			ctx:         context.Background(),
			key:         []byte(""),
			value:       []byte("value2"),
			ttl:         1,
			err:         fmt.Errorf("key is empty"),
		},
		{
			memoryCache: NewMemoryCache(),
			ctx:         context.Background(),
			key:         []byte("set_key2"),
			value:       []byte("value2"),
			ttl:         1,
			err:         nil,
			timeout:     2,
		},
		{
			memoryCache: NewMemoryCache(),
			ctx:         context.Background(),
			key:         []byte("set_key2"),
			value:       []byte("value2"),
			ttl:         7,
			err:         nil,
			timeout:     2,
		},
		{
			memoryCache: NewMemoryCache(),
			ctx:         context.Background(),
			key:         []byte("set_key3"),
			value:       []byte("value3"),
			ttl:         7,
			err:         nil,
			timeout:     2,
		},
	}

	for _, test := range setDataTests {
		//go func(test *setDataTest) {
		err := test.memoryCache.Set(test.ctx, test.key, test.value, &test.ttl)
		assert.Equal(t, test.err, err)

		if err != nil {
			test.memoryCache.dataMu.RLock()
			_, ok := test.memoryCache.data[string(test.key)]
			test.memoryCache.dataMu.RUnlock()
			assert.False(t, ok)
			test.memoryCache.timerMu.RLock()
			_, ok = test.memoryCache.timers[string(test.key)]
			test.memoryCache.timerMu.RUnlock()
			assert.False(t, ok)

			return
		} else {
			test.memoryCache.dataMu.RLock()
			assert.Equal(t, test.memoryCache.data[string(test.key)], test.value)
			test.memoryCache.dataMu.RUnlock()
			test.memoryCache.timerMu.RLock()
			assert.NotNil(t, test.memoryCache.timers[string(test.key)])
			test.memoryCache.timerMu.RUnlock()
		}

		if test.timeout > 0 {
			time.Sleep(time.Duration(test.timeout) * time.Second)
			if test.ttl > test.timeout { // long-lived cache
				test.memoryCache.dataMu.RLock()
				_, ok := test.memoryCache.data[string(test.key)]
				test.memoryCache.dataMu.RUnlock()
				assert.True(t, ok)
				test.memoryCache.timerMu.RLock()
				_, ok = test.memoryCache.timers[string(test.key)]
				test.memoryCache.timerMu.RUnlock()
				assert.True(t, ok)
			} else {
				test.memoryCache.dataMu.RLock()
				_, ok := test.memoryCache.data[string(test.key)]
				test.memoryCache.dataMu.RUnlock()
				assert.False(t, ok)
				test.memoryCache.timerMu.RLock()
				_, ok = test.memoryCache.timers[string(test.key)]
				test.memoryCache.timerMu.RUnlock()
				assert.False(t, ok)
			}
		}
		//}(&test)
	}
}

// TestMemoryCache_Get test Get function
func TestMemoryCache_Get(t *testing.T) {
	mem := NewMemoryCache()
	ctx := context.Background()

	err := mem.Set(ctx, []byte("key1"), []byte("value1"), &zero)
	assert.Equal(t, nil, err)
	k1Val, err := mem.Get(ctx, []byte("key1"))
	assert.Equal(t, nil, err)
	assert.Equal(t, "value1", string(k1Val))

	err = mem.Set(ctx, []byte("key2"), []byte("value2"), &two)
	assert.Equal(t, nil, err)
	k2Val, err := mem.Get(ctx, []byte("key2"))
	assert.Equal(t, nil, err)
	assert.Equal(t, "value2", string(k2Val))

	time.Sleep(time.Duration(3) * time.Second)
	k2Val, err = mem.Get(ctx, []byte("key2"))
	assert.NotNil(t, err)
	assert.Len(t, k2Val, 0)
	assert.Equal(t, "", string(k2Val))

	err = mem.Set(ctx, []byte("key1"), []byte("value_new1"), &three)
	assert.Equal(t, nil, err)
	k1Val, err = mem.Get(ctx, []byte("key1"))
	assert.Equal(t, nil, err)
	assert.Equal(t, "value_new1", string(k1Val))

	time.Sleep(time.Duration(1) * time.Second)

	err = mem.Set(ctx, []byte("key1"), []byte("value_new2"), &ten)
	assert.Equal(t, nil, err)
	k1Val, err = mem.Get(ctx, []byte("key1"))
	assert.Equal(t, nil, err)
	assert.Equal(t, "value_new2", string(k1Val))

	time.Sleep(time.Duration(3) * time.Second)
	k1Val, err = mem.Get(ctx, []byte("key1"))
	assert.NotNil(t, err)
	assert.Len(t, k1Val, 0)
	assert.Equal(t, "", string(k1Val))
}

// TestMemoryCache_Has test Has function
func TestMemoryCache_Has(t *testing.T) {
	mem := NewMemoryCache()
	ctx := context.Background()

	err := mem.Set(ctx, []byte("key1"), []byte("value1"), &zero)
	assert.Equal(t, nil, err)
	k1Val, err := mem.Get(ctx, []byte("key1"))
	assert.Equal(t, nil, err)
	assert.Equal(t, "value1", string(k1Val))

	mustTrue := mem.Has(ctx, []byte("key1"))
	assert.True(t, mustTrue)

	mustFalse := mem.Has(ctx, []byte("not_exists_key"))
	assert.False(t, mustFalse)
}

// TestMemoryCache_Inc_NotExist_Key test Inc with note exists key
func TestMemoryCache_Inc_NotExist_Key(t *testing.T) {
	mem := NewMemoryCache()
	ctx := context.Background()

	val, err := mem.Get(ctx, []byte("inc_key_not_exists"))
	assert.NotNil(t, err)
	assert.Equal(t, "", string(val))

	value, err := mem.Inc(ctx, []byte("inc_key_not_exists"), &fifteen)
	assert.NotNil(t, err)
	assert.Equal(t, ValueNotFoundByKey, err)
	assert.Equal(t, int64(1), value)
}

// TestMemoryCache_Inc test Inc function
func TestMemoryCache_Inc(t *testing.T) {
	mem := NewMemoryCache()
	ctx := context.Background()

	val, err := mem.Get(ctx, []byte("inc_key"))
	assert.NotNil(t, err)
	assert.Len(t, val, 0)
	assert.Equal(t, "", string(val))

	for i := 1; i <= 10; i++ {
		value, err := mem.Inc(ctx, []byte("inc_key"), &fifteen)
		if i == 1 {
			assert.NotNil(t, err)
			assert.Equal(t, ValueNotFoundByKey, err)
		} else {
			assert.Nil(t, err)
		}
		time.Sleep(time.Duration(1) * time.Second)
		assert.Equal(t, int64(i), value)
	}

	for i := 1; i <= 5; i++ {
		value, err := mem.Inc(ctx, []byte("inc_key2"), &three)
		if i == 5 {
			assert.NotNil(t, err)
			assert.Equal(t, ValueNotFoundByKey, err)
			assert.Equal(t, int64(1), value)
		} else {
			if i == 1 {
				assert.NotNil(t, err)
				assert.Equal(t, ValueNotFoundByKey, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, int64(i), value)
			}
		}

		val, err := mem.Get(ctx, []byte("inc_key2"))
		if i == 5 {
			assert.Nil(t, err)
			assert.Equal(t, "1", string(val))
		} else {
			assert.Nil(t, err)
			assert.Equal(t, strconv.Itoa(i), string(val))
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
}

// TestMemoryCache_Decr test Decr function
func TestMemoryCache_Decr(t *testing.T) {
	mem := NewMemoryCache()
	ctx := context.Background()

	val, err := mem.Get(ctx, []byte("decr_key"))
	assert.NotNil(t, err)
	assert.Len(t, val, 0)
	assert.Equal(t, "", string(val))

	err = mem.Set(ctx, []byte("decr_key"), []byte("2"), &ten)
	assert.Nil(t, err)

	for i := 1; i <= 3; i++ {
		k := 2 - i
		value, err := mem.Decr(ctx, []byte("decr_key"), &fifteen)
		assert.Nil(t, err)
		time.Sleep(time.Duration(1) * time.Second)
		assert.Equal(t, int64(k), value)
	}

	for i := -1; i >= -5; i-- {
		value, err := mem.Decr(ctx, []byte("decr_key2"), &three)
		if i == -5 {
			assert.NotNil(t, err)
			assert.Equal(t, int64(-1), value)
		} else {
			if i == -1 {
				assert.NotNil(t, err)
				assert.Equal(t, ValueNotFoundByKey, err)
				assert.Equal(t, int64(i), value)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, int64(i), value)
			}
		}

		val, err := mem.Get(ctx, []byte("decr_key2"))
		if i == -5 {
			assert.Nil(t, err)
			assert.Equal(t, "-1", string(val))
		} else {
			assert.Nil(t, err)
			assert.Equal(t, strconv.Itoa(i), string(val))
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
}

// TestMemoryCache_Del test Del function
func TestMemoryCache_Del(t *testing.T) {
	mem := NewMemoryCache()
	ctx := context.Background()

	val, err := mem.Get(ctx, []byte("del"))
	assert.NotNil(t, err)
	assert.Len(t, val, 0)
	assert.Equal(t, "", string(val))

	err = mem.Set(ctx, []byte("del"), []byte("exists"), &zero)
	assert.Nil(t, err)
	assert.Equal(t, string(mem.data["del"]), "exists")

	err = mem.Del(ctx, []byte("del"))
	assert.Nil(t, err)

	val, err = mem.Get(ctx, []byte("del"))
	assert.NotNil(t, err)
	assert.Len(t, val, 0)
	assert.Equal(t, "", string(val))

	err = mem.Del(ctx, []byte("del_not_exists"))
	assert.Nil(t, err)
}

// TestMemoryCache_Clear test Clear function
func TestMemoryCache_Clear(t *testing.T) {
	mem := NewMemoryCache()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		ttl := uint64((i % 2) * 20)
		key := "item_" + strconv.Itoa(i)
		err := mem.Set(ctx, []byte(key), []byte(strconv.Itoa(i)), &ttl)
		assert.Nil(t, err)
		assert.Equal(t, string(mem.data[key]), strconv.Itoa(i))
	}

	err := mem.Clear(ctx)
	assert.Nil(t, err)

	for i := 0; i < 10; i++ {
		key := "item_" + strconv.Itoa(i)
		val, err := mem.Get(ctx, []byte(key))
		assert.NotNil(t, err)
		assert.Len(t, val, 0)
		assert.Equal(t, "", string(val))
	}

	assert.Len(t, mem.data, 0)
	assert.Len(t, mem.timers, 0)
}

// TestMemoryCache_callCancel test callCancel function
func TestMemoryCache_callCancel(t *testing.T) {
	mem := NewMemoryCache()
	ctx := context.Background()

	val, err := mem.Get(ctx, []byte("callCancel"))
	assert.NotNil(t, err)
	assert.Len(t, val, 0)
	assert.Equal(t, "", string(val))

	err = mem.Set(ctx, []byte("callCancel"), []byte("exists"), &two)
	assert.Nil(t, err)
	assert.Equal(t, string(mem.data["callCancel"]), "exists")

	err = mem.callCancel([]byte("callCancel"))
	assert.Nil(t, err)

	_, ok := mem.timers["callCancel"]
	assert.False(t, ok)

	time.Sleep(time.Duration(5) * time.Second)

	val, err = mem.Get(ctx, []byte("callCancel"))
	assert.Nil(t, err)
	assert.Equal(t, "exists", string(val))
}

// TestMemoryCache_sliceByteToInt64 test sliceByteToInt64 function
func TestMemoryCache_sliceByteToInt64(t *testing.T) {
	mem := NewMemoryCache()
	ctx := context.Background()

	val123, err := mem.sliceByteToInt64([]byte("key1"))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "is not found")
	assert.Equal(t, int64(0), val123)

	err = mem.Set(ctx, []byte("key1"), []byte("123"), &twenty)
	assert.Nil(t, err)
	assert.Equal(t, string(mem.data["key1"]), "123")

	val123, err = mem.sliceByteToInt64([]byte("key1"))
	assert.Nil(t, err)
	assert.Equal(t, int64(123), val123)

	valNot123, err := mem.sliceByteToInt64([]byte("not123"))
	assert.NotNil(t, err)
	assert.Equal(t, int64(0), valNot123)
}

// TestMemoryCache_initCancel test initCancel function
func TestMemoryCache_initCancel(t *testing.T) {
	mem := NewMemoryCache()
	ctx := context.Background()

	mem.initCancel(ctx, "key1", &three)
	assert.NotNil(t, mem.timers["key1"])
}
