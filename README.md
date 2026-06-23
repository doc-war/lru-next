# TTLCacheNext

[![Go Reference](https://pkg.go.dev/badge/github.com/doc-war/TTLCacheNext.svg)](https://pkg.go.dev/github.com/doc-war/TTLCacheNext)
[![Go Report Card](https://goreportcard.com/badge/github.com/doc-war/TTLCacheNext)](https://goreportcard.com/report/github.com/doc-war/TTLCacheNext)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**TTLCacheNext** 是一个超高性能、泛型、带 TTL 过期和 LRU 淘汰的内存缓存库。专为读密集、高并发的场景设计，内置缓存雪崩防护和脏数据降级能力。

---

## 架构

```
+-----------+
|  Cache[T] |  ← 面向用户的泛型 API（package cache）
+-----+-----+
      |
+-----v------+------+------+------+
| Shard[0]   | ...  | ...  | [255]|  256 个分片，各有 sync.RWMutex
+-----+------+------+------+------+
      |
+-----v------+
|  lruCore   |  ← 无锁 LRU 双向链表
|  + hashmap |
+------------+
```

- **256 个分片**通过 FNV-1a 哈希路由，大幅降低锁竞争
- 每个分片拥有独立的 `sync.RWMutex` 和无锁 `lruCore`，无嵌套锁开销
- `lruCore` 使用 **哈希表 + 双向链表**（链表头 = 最近使用，链表尾 = 最久未使用）
- TTL 过期在读取时检查，过期数据由用户提供的 `loader` 回调刷新

### 快路径 / 慢路径

```
GetOrLoad(key, loader)
  │
  ├─ RLock → 命中且未过期？  ──✅ 直接返回
  │
  └─ Lock → 双重检查
       ├─ 未过期？            ──✅ 返回
       └─ loader(key)
            ├─ 成功  → 写入缓存并返回
            └─ 失败  → 返回旧值（降级）
```

慢路径的写锁在每个分片内串行化 `loader` 调用，提供了**内置的缓存雪崩防护**，无需额外的 singleflight 机制。

---

## 基准测试

硬件：Intel i5-8265U @ 1.60GHz, Windows, Go 1.25.3。
10K key 固定池，`RunParallel` 并发模式，取 5 次均值。

```
go test -bench=. -benchmem -count=5
```

#### 并发写入（Load+Set / Add / Set）

| 库 | ns/op | B/op | allocs/op |
|---|---|---|---|
| **TTLCacheNext** | **158** | 2 | 0 |
| golang-lru v2 | 670 | 0 | 0 |
| go-cache | 532 | 0 | 0 |

TTLCacheNext比golang-lru v2、go-cache分别快4.2倍和3.4倍

#### 并发读取（Get 命中）

| 库 | ns/op | B/op | allocs/op | vs |
|---|---|---|---|---|
| **TTLCacheNext** | **183** | 2 | 0 | — |
| golang-lru v2 | 762 | 0 | 0 | 快 4.2× |
| go-cache | 98 | 0 | 0 | 慢 1.9× * |

TTLCacheNext比golang-lru v2快4.2倍、比go-cache慢1.9倍

> \* go-cache 无 LRU 淘汰机制，内部仅为 `map` + `sync.RWMutex`，读性能更高但功能不对等。

#### 混合负载（90% 读 + 10% 写）

| 库 | ns/op | B/op | allocs/op | vs TTLCacheNext |
|---|---|---|---|---|
| **TTLCacheNext** | **405** | 45 | 0 | — |
| golang-lru v2 | 833 | 12 | 0 | **快 2.1×** |
| go-cache | 700 | 16 | 0 | **快 1.7×** |

TTLCacheNext比golang-lru v2、go-cache分别快2.1倍、1.7倍

#### 关键结论

- **写 & 混合场景全面领先**：分片设计大幅降低锁竞争，无锁 LRU 核心消除嵌套锁开销
- **读场景接近 go-cache 的 2 倍**：go-cache 没有 LRU 淘汰，仅为裸 map 封装（功能不对等）
- **相比 golang-lru v2（最流行 LRU 库）**：所有场景均有 2~4 倍性能优势

---

## 使用

#### 安装

```bash
go get github.com/doc-war/TTLCacheNext
```

#### 快速开始

```go
package main

import (
	"fmt"
	"time"
	"github.com/doc-war/TTLCacheNext"
)

func main() {
	c, err := cache.New[string](1000, 5*time.Minute)
	if err != nil {
		panic(err)
	}

	val, err := c.GetOrLoad("hello", func(key string) (string, error) {
		return "world", nil
	})
	fmt.Println(val) // world

	fmt.Println(c.Len()) // 1
	c.Delete("hello")
	c.Clear()
}
```

#### API

```go
// 创建一个最多缓存 ~1000 个 key、TTL 5 分钟的缓存实例。
c, err := cache.New[T](maxKeys int, ttl time.Duration)

// 获取或通过 loader 回源加载。
val, err := c.GetOrLoad(id string, loader func(string) (T, error)) (T, error)

// 删除指定 key。
c.Delete(id string)

// 清空所有缓存。
c.Clear()

// 当前缓存条目总数（近似值）。
c.Len() int
```

> **注意：** 当 `loader` 失败时，`GetOrLoad` 会返回**旧的过期值**（如果存在），确保优雅降级。

---

## License

MIT — 见 [LICENSE](LICENSE)。
