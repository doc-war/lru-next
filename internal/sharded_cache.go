package internal

import (
	"hash/fnv"
	"sync"
	"time"
)

const shardCount = 256

// cacheItem 缓存数据和过期时间，仅 internal 包内部使用
type cacheItem[T any] struct {
	Data     T
	ExpireAt time.Time
}

// cacheShard 缓存分片
type cacheShard[T any] struct {
	mu    sync.RWMutex
	cache *lruCore[cacheItem[T]]
}

/*
ShardedCache 是带 TTL 过期、分片支持高并发访问的缓存核心实现。
对外暴露给上层 Cache[T] 包装使用，不直接面向最终用户。
*/
type ShardedCache[T any] struct {
	shards [shardCount]*cacheShard[T]
	ttl    time.Duration // 刷新间隔
}

/*
NewShardedCache 初始化:
1、maxKeys 防止内存泄露
2、底层自动保证最多缓存几个，自动剔除最少使用的
*/
func NewShardedCache[T any](maxKeys int, ttl time.Duration) (*ShardedCache[T], error) {
	sc := &ShardedCache[T]{ttl: ttl}
	// 初始化各分片，每个分片管理 1/shardCount 的数据
	shardSize := maxKeys/shardCount + 1
	for i := range shardCount {
		sc.shards[i] = &cacheShard[T]{
			cache: newLRUCore[cacheItem[T]](shardSize),
		}
	}
	return sc, nil
}

// getShard 获取 key 对应的分片
func (c *ShardedCache[T]) getShard(key string) *cacheShard[T] {
	return c.shards[fnv32a(key)%shardCount]
}

// fnv32a 计算字符串的 FNV-1a 32位哈希，用于分片路由
func fnv32a(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

/*
GetOrLoad 从缓存中获取，如果过期或不存在则通过 loader 加载
特别注意：返回的结构体 T 可能是零值
*/
func (c *ShardedCache[T]) GetOrLoad(
	id string, // 传入索引参数，比如 channelID
	loader func(string) (T, error), // 使用索引进行重新查询
) (T, error) {
	shard := c.getShard(id)
	var oldT T // 零值
	now := time.Now()
	// 快路径：读锁检查缓存
	shard.mu.RLock()
	if item, ok := shard.cache.Get(id); ok {
		if now.Before(item.ExpireAt) {
			// 缓存仍在有效期内
			shard.mu.RUnlock()
			return item.Data, nil
		}
		// 过期但保留旧值，万一重新加载失败可以降级使用
		oldT = item.Data
	}
	shard.mu.RUnlock()
	/**
	慢路径：写锁加载新数据。
	1、这可以确保 loader 方法的执行是互斥独占，避免并发访问
	2、因为只有一个请求可以拿到写锁，所以不需要单飞来消除重复请求
	*/
	shard.mu.Lock()
	defer shard.mu.Unlock()
	/**
	双重检查：
	1、可能在等待写锁期间已被其他 goroutine 更新
	2、如果被其他访问拿到了新值，直接返回
	*/
	if item, ok := shard.cache.Get(id); ok {
		if time.Now().Before(item.ExpireAt) {
			return item.Data, nil
		}
		oldT = item.Data
	}
	// 加载新数据
	data, err := loader(id)
	if err != nil {
		// 加载失败，返回旧值（如果有）
		return oldT, err
	}
	// 写入缓存，满 key 时自动剔除不活跃的
	shard.cache.Add(id, cacheItem[T]{
		Data:     data,
		ExpireAt: time.Now().Add(c.ttl),
	})
	return data, nil
}

// Delete 删除指定 key 的缓存
func (c *ShardedCache[T]) Delete(id string) {
	shard := c.getShard(id)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	shard.cache.Remove(id)
}

// Clear 清空所有缓存
func (c *ShardedCache[T]) Clear() {
	for i := range c.shards {
		shard := c.shards[i]
		shard.mu.Lock()
		shard.cache.Purge()
		shard.mu.Unlock()
	}
}

// Len 返回当前缓存的总数量（近似值，非原子操作）
func (c *ShardedCache[T]) Len() int {
	total := 0
	for i := range c.shards {
		shard := c.shards[i]
		shard.mu.RLock()
		total += shard.cache.Len()
		shard.mu.RUnlock()
	}
	return total
}
