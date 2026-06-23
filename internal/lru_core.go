package internal

// lruNode 双向链表节点
type lruNode[T any] struct {
	key  string
	val  T
	prev *lruNode[T]
	next *lruNode[T]
}

/*
lruCore 是不带锁的 LRU 核心实现（链表头 = 最近使用，链表尾 = 最久未使用）。
非线程安全，并发安全完全依赖外层 cacheShard 的 RWMutex 保证，
这样避免了"外层分片锁 + 内部 lru 包自带锁"的双重加锁开销。
*/
type lruCore[T any] struct {
	capacity int
	items    map[string]*lruNode[T]
	head     *lruNode[T]
	tail     *lruNode[T]
}

func newLRUCore[T any](capacity int) *lruCore[T] {
	if capacity < 1 {
		capacity = 1
	}
	return &lruCore[T]{
		capacity: capacity,
		items:    make(map[string]*lruNode[T], capacity),
	}
}

// Get 命中后移动到链表头部（标记为最近使用）
func (l *lruCore[T]) Get(key string) (T, bool) {
	n, ok := l.items[key]
	if !ok {
		var zero T
		return zero, false
	}
	l.moveToFront(n)
	return n.val, true
}

// Add 新增或更新；若超出容量，淘汰链表尾部（最久未使用）节点
func (l *lruCore[T]) Add(key string, val T) {
	if n, ok := l.items[key]; ok {
		n.val = val
		l.moveToFront(n)
		return
	}
	n := &lruNode[T]{key: key, val: val}
	l.items[key] = n
	l.pushFront(n)
	if len(l.items) > l.capacity {
		l.removeTail()
	}
}

// Remove 删除指定 key，key 不存在时为空操作
func (l *lruCore[T]) Remove(key string) {
	n, ok := l.items[key]
	if !ok {
		return
	}
	l.unlink(n)
	delete(l.items, key)
}

// Purge 清空所有数据
func (l *lruCore[T]) Purge() {
	l.items = make(map[string]*lruNode[T], l.capacity)
	l.head = nil
	l.tail = nil
}

// Len 当前缓存数量
func (l *lruCore[T]) Len() int {
	return len(l.items)
}

// ---- 链表基础操作，均为 O(1) ----

func (l *lruCore[T]) pushFront(n *lruNode[T]) {
	n.prev = nil
	n.next = l.head
	if l.head != nil {
		l.head.prev = n
	}
	l.head = n
	if l.tail == nil {
		l.tail = n
	}
}

func (l *lruCore[T]) unlink(n *lruNode[T]) {
	if n.prev != nil {
		n.prev.next = n.next
	} else {
		l.head = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	} else {
		l.tail = n.prev
	}
	n.prev = nil
	n.next = nil
}

func (l *lruCore[T]) moveToFront(n *lruNode[T]) {
	if l.head == n {
		return
	}
	l.unlink(n)
	l.pushFront(n)
}

func (l *lruCore[T]) removeTail() {
	if l.tail == nil {
		return
	}
	n := l.tail
	l.unlink(n)
	delete(l.items, n.key)
}
