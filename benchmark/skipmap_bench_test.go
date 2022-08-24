package benchmark

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/cornelk/hashmap"
	"github.com/google/btree"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/zhangyunhao116/fastrand"
	"github.com/zhangyunhao116/skipmap"
)

const (
	initsize         = 1 << 10 // for `Load` `1Delete9Store90Load` `1Range9Delete90Store900Load`
	randN            = math.MaxUint32
	mockvalue uint64 = 100
)

func BenchmarkInt64(b *testing.B) {
	all := []benchInt64Task{{
		name: "skipmap", New: func() int64Map {
			return skipmap.NewInt64[any]()
		}}}
	all = append(all, benchInt64Task{
		name: "sync.Map", New: func() int64Map {
			return new(int64SyncMap)
		}})
	all = append(all, benchInt64Task{
		name: "btree.BTree", New: func() int64Map {
			return &int64BTreeSyncMap{
				data: btree.NewG(2, btree.LessFunc[kvInt64Pair](func(a, b kvInt64Pair) bool {
					return a.key < b.key
				})),
			}
		}})
	all = append(all, benchInt64Task{
		name: "cornelk/hashmap", New: func() int64Map {
			return &int64CornelkHashmap{
				data: hashmap.New[int64, interface{}](),
			}
		}})
	benchStore(b, all)
	benchLoad50Hits(b, all)
	bench30Store70Load(b, all)
	bench50Store50Load(b, all)
	bench1Delete9Store90Load(b, all)
	bench1Range9Delete90Store900Load(b, all)
}

func BenchmarkString(b *testing.B) {
	all := []benchStringTask{{
		name: "skipmap", New: func() stringMap {
			return skipmap.NewString[any]()
		}}}
	all = append(all, benchStringTask{
		name: "sync.Map", New: func() stringMap {
			return new(stringSyncMap)
		}})
	all = append(all, benchStringTask{
		name: "btree.BTree", New: func() stringMap {
			return &stringBTreeSyncMap{
				data: btree.NewG(2, btree.LessFunc[kvStringPair](func(a, b kvStringPair) bool {
					return strings.Compare(a.key, b.key) < 0
				})),
			}
		}})
	all = append(all, benchStringTask{
		name: "concurrent-map", New: func() stringMap {
			return &stringCMap{
				data: cmap.New[interface{}](),
			}
		}})
	benchStringStore(b, all)
	benchStringLoad50Hits(b, all)
	benchString30Store70Load(b, all)
	benchString50Store50Load(b, all)
	benchString1Delete9Store90Load(b, all)
	benchString1Range9Delete90Store900Load(b, all)
}

func benchStore(b *testing.B, benchTasks []benchInt64Task) {
	for _, v := range benchTasks {
		b.Run("Store/"+v.name, func(b *testing.B) {
			s := v.New()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					s.Store(int64(fastrand.Uint32n(randN)), mockvalue)
				}
			})
		})
	}
}

func benchLoad50Hits(b *testing.B, benchTasks []benchInt64Task) {
	for _, v := range benchTasks {
		b.Run("Load50Hits/"+v.name, func(b *testing.B) {
			const rate = 2
			s := v.New()
			for i := 0; i < initsize*rate; i++ {
				if fastrand.Uint32n(rate) == 0 {
					s.Store(int64(i), mockvalue)
				}
			}
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					s.Load(int64(fastrand.Uint32n(initsize * rate)))
				}
			})
		})
	}
}

func bench30Store70Load(b *testing.B, benchTasks []benchInt64Task) {
	for _, v := range benchTasks {
		b.Run("30Store70Load/"+v.name, func(b *testing.B) {
			s := v.New()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					u := fastrand.Uint32n(10)
					if u < 3 {
						s.Store(int64(fastrand.Uint32n(randN)), mockvalue)
					} else {
						s.Load(int64(fastrand.Uint32n(randN)))
					}
				}
			})
		})
	}
}

func bench50Store50Load(b *testing.B, benchTasks []benchInt64Task) {
	for _, v := range benchTasks {
		b.Run("50Store50Load/"+v.name, func(b *testing.B) {
			s := v.New()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					u := fastrand.Uint32n(10)
					if u < 5 {
						s.Store(int64(fastrand.Uint32n(randN)), mockvalue)
					} else {
						s.Load(int64(fastrand.Uint32n(randN)))
					}
				}
			})
		})
	}
}

func bench1Delete9Store90Load(b *testing.B, benchTasks []benchInt64Task) {
	for _, v := range benchTasks {
		b.Run("1Delete9Store90Load/"+v.name, func(b *testing.B) {
			s := v.New()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					u := fastrand.Uint32n(100)
					if u < 9 {
						s.Store(int64(fastrand.Uint32n(randN)), mockvalue)
					} else if u == 10 {
						s.Delete(int64(fastrand.Uint32n(randN)))
					} else {
						s.Load(int64(fastrand.Uint32n(randN)))
					}
				}
			})
		})
	}
}

func bench1Range9Delete90Store900Load(b *testing.B, benchTasks []benchInt64Task) {
	for _, v := range benchTasks {
		b.Run("1Range9Delete90Store900Load/"+v.name, func(b *testing.B) {
			s := v.New()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					u := fastrand.Uint32n(1000)
					if u == 0 {
						s.Range(func(key int64, value interface{}) bool {
							return true
						})
					} else if u > 10 && u < 20 {
						s.Delete(int64(fastrand.Uint32n(randN)))
					} else if u >= 100 && u < 190 {
						s.Store(int64(fastrand.Uint32n(randN)), mockvalue)
					} else {
						s.Load(int64(fastrand.Uint32n(randN)))
					}
				}
			})
		})
	}
}

func benchStringStore(b *testing.B, benchTasks []benchStringTask) {
	for _, v := range benchTasks {
		b.Run("Store/"+v.name, func(b *testing.B) {
			s := v.New()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					s.Store(strconv.Itoa(int(fastrand.Uint32n(randN))), mockvalue)
				}
			})
		})
	}
}

func benchStringLoad50Hits(b *testing.B, benchTasks []benchStringTask) {
	for _, v := range benchTasks {
		b.Run("Load50Hits/"+v.name, func(b *testing.B) {
			const rate = 2
			s := v.New()
			for i := 0; i < initsize*rate; i++ {
				if fastrand.Uint32n(rate) == 0 {
					s.Store(strconv.Itoa(int(fastrand.Uint32n(randN))), mockvalue)
				}
			}
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					s.Load(strconv.Itoa(int(fastrand.Uint32n(randN))))
				}
			})
		})
	}
}

func benchString30Store70Load(b *testing.B, benchTasks []benchStringTask) {
	for _, v := range benchTasks {
		b.Run("30Store70Load/"+v.name, func(b *testing.B) {
			s := v.New()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					u := fastrand.Uint32n(10)
					if u < 3 {
						s.Store(strconv.Itoa(int(fastrand.Uint32n(randN))), mockvalue)
					} else {
						s.Load(strconv.Itoa(int(fastrand.Uint32n(randN))))
					}
				}
			})
		})
	}
}

func benchString50Store50Load(b *testing.B, benchTasks []benchStringTask) {
	for _, v := range benchTasks {
		b.Run("50Store50Load/"+v.name, func(b *testing.B) {
			s := v.New()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					u := fastrand.Uint32n(10)
					if u < 3 {
						s.Store(strconv.Itoa(int(fastrand.Uint32n(randN))), mockvalue)
					} else {
						s.Load(strconv.Itoa(int(fastrand.Uint32n(randN))))
					}
				}
			})
		})
	}
}

func benchString1Delete9Store90Load(b *testing.B, benchTasks []benchStringTask) {
	for _, v := range benchTasks {
		b.Run("1Delete9Store90Load/"+v.name, func(b *testing.B) {
			s := v.New()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					u := fastrand.Uint32n(100)
					if u < 9 {
						s.Store(strconv.Itoa(int(fastrand.Uint32n(randN))), mockvalue)
					} else if u == 10 {
						s.Delete(strconv.Itoa(int(fastrand.Uint32n(randN))))
					} else {
						s.Load(strconv.Itoa(int(fastrand.Uint32n(randN))))
					}
				}
			})
		})
	}
}

func benchString1Range9Delete90Store900Load(b *testing.B, benchTasks []benchStringTask) {
	for _, v := range benchTasks {
		b.Run("1Range9Delete90Store900Load/"+v.name, func(b *testing.B) {
			s := v.New()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					u := fastrand.Uint32n(1000)
					if u == 0 {
						s.Range(func(_ string, _ interface{}) bool {
							return true
						})
					} else if u > 10 && u < 20 {
						s.Delete(strconv.Itoa(int(fastrand.Uint32n(randN))))
					} else if u >= 100 && u < 190 {
						s.Store(strconv.Itoa(int(fastrand.Uint32n(randN))), mockvalue)
					} else {
						s.Load(strconv.Itoa(int(fastrand.Uint32n(randN))))
					}
				}
			})
		})
	}
}

type benchInt64Task struct {
	New  func() int64Map
	name string
}

type int64Map interface {
	Store(x int64, v interface{})
	Load(x int64) (interface{}, bool)
	Delete(x int64) bool
	Range(f func(key int64, value interface{}) bool)
}

type int64SyncMap struct {
	data sync.Map
}

func (m *int64SyncMap) Store(x int64, v interface{}) {
	m.data.Store(x, v)
}

func (m *int64SyncMap) Load(x int64) (interface{}, bool) {
	return m.data.Load(x)
}

func (m *int64SyncMap) Delete(x int64) bool {
	m.data.Delete(x)
	return true
}

func (m *int64SyncMap) Range(f func(key int64, value interface{}) bool) {
	m.data.Range(func(key, value interface{}) bool {
		return !f(key.(int64), value)
	})
}

type kvInt64Pair struct {
	val interface{}
	key int64
}

type int64BTreeSyncMap struct {
	mu   sync.RWMutex
	data *btree.BTreeG[kvInt64Pair]
}

func (m *int64BTreeSyncMap) Store(x int64, v interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data.ReplaceOrInsert(kvInt64Pair{val: v, key: x})
}

func (m *int64BTreeSyncMap) Load(x int64) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data.Get(kvInt64Pair{val: nil, key: x})
	if ok {
		return v.val, true
	}
	return nil, false
}

func (m *int64BTreeSyncMap) Delete(x int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.data.Delete(kvInt64Pair{val: nil, key: x})
	return exists
}

func (m *int64BTreeSyncMap) Range(f func(key int64, value interface{}) bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data.Ascend(btree.ItemIteratorG[kvInt64Pair](func(i kvInt64Pair) bool {
		return !f(i.key, i.val)
	}))
}

type int64CornelkHashmap struct {
	data *hashmap.HashMap[int64, interface{}]
	name string
}

func (m *int64CornelkHashmap) Store(x int64, v interface{}) {
	m.data.Set(x, v)
}
func (m *int64CornelkHashmap) Load(x int64) (interface{}, bool) {
	return m.data.Get(x)
}
func (m *int64CornelkHashmap) Delete(x int64) bool {
	return m.data.Del(x)
}
func (m *int64CornelkHashmap) Range(f func(key int64, value interface{}) bool) {
	m.data.Range(func(key int64, value interface{}) bool {
		return !f(key, value)
	})
}

type benchStringTask struct {
	New  func() stringMap
	name string
}

type stringMap interface {
	Store(x string, v interface{})
	Load(x string) (interface{}, bool)
	Delete(x string) bool
	Range(f func(key string, value interface{}) bool)
}

type stringSyncMap struct {
	data sync.Map
}

func (m *stringSyncMap) Store(x string, v interface{}) {
	m.data.Store(x, v)
}

func (m *stringSyncMap) Load(x string) (interface{}, bool) {
	return m.data.Load(x)
}

func (m *stringSyncMap) Delete(x string) bool {
	m.data.Delete(x)
	return true
}

func (m *stringSyncMap) Range(f func(key string, value interface{}) bool) {
	m.data.Range(func(key, value interface{}) bool {
		return !f(key.(string), value)
	})
}

type kvStringPair struct {
	val interface{}
	key string
}

type stringBTreeSyncMap struct {
	mu   sync.RWMutex
	data *btree.BTreeG[kvStringPair]
}

func (m *stringBTreeSyncMap) Store(x string, v interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data.ReplaceOrInsert(kvStringPair{val: v, key: x})
}

func (m *stringBTreeSyncMap) Load(x string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data.Get(kvStringPair{val: nil, key: x})
	if ok {
		return v.val, true
	}
	return nil, false
}

func (m *stringBTreeSyncMap) Delete(x string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.data.Delete(kvStringPair{val: nil, key: x})
	return exists
}

func (m *stringBTreeSyncMap) Range(f func(key string, value interface{}) bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data.Ascend(btree.ItemIteratorG[kvStringPair](func(i kvStringPair) bool {
		return !f(i.key, i.val)
	}))
}

type stringCMap struct {
	data cmap.ConcurrentMap[interface{}]
}

func (m *stringCMap) Store(x string, v interface{}) {
	m.data.Set(x, v)
}

func (m *stringCMap) Load(x string) (interface{}, bool) {
	v, ok := m.data.Get(x)
	return v, ok
}

func (m *stringCMap) Delete(x string) bool {
	m.data.Remove(x)
	return true
}

func (m *stringCMap) Range(f func(key string, value interface{}) bool) {
	m.data.IterCb(cmap.IterCb[interface{}](func(key string, value interface{}) {
		f(key, value)
	}))
}
