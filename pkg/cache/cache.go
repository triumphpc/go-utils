package cache

// Cache представляет собой generic-кэш для хранения пар ключ-значение.
// K - тип ключа (должен быть comparable для использования в map)
// V - тип значения (может быть любым)
type Cache[K comparable, V any] struct {
	store map[K]V
}

// NewCache создает и возвращает новый экземпляр Cache.
// Возвращает указатель на инициализированный кэш с пустым хранилищем.
func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{store: make(map[K]V)}
}

// Set добавляет или обновляет значение в кэше по указанному ключу.
// key - ключ для сохранения значения
// value - значение, которое нужно сохранить в кэше
func (c *Cache[K, V]) Set(key K, value V) {
	c.store[key] = value
}

// Get возвращает значение из кэша по ключу и флаг наличия значения.
// key - ключ для поиска значения
// Возвращает:
//   - значение типа V, если ключ найден
//   - false, если ключ не найден в кэше
//
// Примечание: если ключ не найден, возвращается zero-value для типа V
func (c *Cache[K, V]) Get(key K) (V, bool) {
	v, ok := c.store[key]
	return v, ok
}
