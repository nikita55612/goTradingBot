package seqs

import (
	"slices"
	"sync"
)

// OrderedMap - потокобезопасная мапа с сохранением порядка элементов
type OrderedMap[K comparable, V any] struct {
	m   map[K]V
	s   []K
	cap int
	mu  sync.RWMutex
}

// NewOrderedMap создает новую OrderedMap с начальной емкостью
func NewOrderedMap[K comparable, V any](cap int) *OrderedMap[K, V] {
	return &OrderedMap[K, V]{
		m: make(map[K]V, cap),
		s: make([]K, 0, cap),
	}
}

// Set устанавливает значение для ключа, сохраняя исходную позицию при обновлении.
// Возвращает:
// - текущий индекс ключа, если он уже существовал (позиция не меняется)
// - последний индекс, если ключ добавлен впервые
func (om *OrderedMap[K, V]) Set(k K, v V) int {
	om.mu.Lock()
	defer om.mu.Unlock()

	if idx := slices.Index(om.s, k); idx != -1 {
		om.m[k] = v
		return idx
	}
	om.m[k] = v
	om.s = append(om.s, k)
	return len(om.s) - 1
}

// Add устанавливает значение для ключа, перемещая его в конец при обновлении.
// Всегда возвращает последний индекс
func (om *OrderedMap[K, V]) Add(k K, v V) int {
	om.mu.Lock()
	defer om.mu.Unlock()

	if idx := slices.Index(om.s, k); idx != -1 {
		if idx != len(om.s)-1 {
			om.s = slices.Delete(om.s, idx, idx+1)
			om.shrinkIfNeeded()
			om.s = append(om.s, k)
		}
	} else {
		om.s = append(om.s, k)
	}
	om.m[k] = v
	return len(om.s) - 1
}

// Get возвращает значение по ключу
func (om *OrderedMap[K, V]) Get(k K) (V, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	v, ok := om.m[k]
	return v, ok
}

// KeyByIndex возвращает ключ по индексу (отрицательный - с конца)
func (om *OrderedMap[K, V]) KeyByIndex(i int) (K, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	if i = normalizeIndex(i, len(om.s)); i == -1 {
		var zero K
		return zero, false
	}
	return om.s[i], true
}

// GetByIndex возвращает значение по индексу (отрицательный - с конца)
func (om *OrderedMap[K, V]) GetByIndex(i int) (V, bool) {
	if k, ok := om.KeyByIndex(i); ok {
		return om.Get(k)
	}
	var zero V
	return zero, false
}

// Index возвращает индекс ключа или -1 если не найден
func (om *OrderedMap[K, V]) Index(k K) int {
	om.mu.RLock()
	defer om.mu.RUnlock()

	return slices.Index(om.s, k)
}

// Delete удаляет элемент по ключу, возвращает true если удален
func (om *OrderedMap[K, V]) Delete(k K) bool {
	om.mu.Lock()
	defer om.mu.Unlock()

	if _, ok := om.m[k]; !ok {
		return false
	}
	delete(om.m, k)
	if i := slices.Index(om.s, k); i >= 0 {
		om.s = slices.Delete(om.s, i, i+1)
		om.shrinkIfNeeded()
	}
	return true
}

// DeleteByIndex удаляет элемент по индексу (отрицательный - с конца)
func (om *OrderedMap[K, V]) DeleteByIndex(i int) bool {
	om.mu.Lock()
	defer om.mu.Unlock()

	if i = normalizeIndex(i, len(om.s)); i == -1 {
		return false
	}
	delete(om.m, om.s[i])
	om.s = slices.Delete(om.s, i, i+1)
	om.shrinkIfNeeded()
	return true
}

// shrinkIfNeeded уменьшает емкость среза если он заполнен меньше чем наполовину
func (om *OrderedMap[K, V]) shrinkIfNeeded() {
	if len(om.s) < cap(om.s)/2 {
		newSlice := make([]K, len(om.s))
		copy(newSlice, om.s)
		om.s = newSlice
	}
}

// Keys возвращает все ключи в порядке добавления
func (om *OrderedMap[K, V]) Keys() []K {
	om.mu.RLock()
	defer om.mu.RUnlock()

	return slices.Clone(om.s)
}

// Values возвращает все значения в порядке добавления
func (om *OrderedMap[K, V]) Values() []V {
	om.mu.RLock()
	defer om.mu.RUnlock()

	values := make([]V, len(om.s))
	for i, k := range om.s {
		values[i] = om.m[k]
	}
	return values
}

// Len возвращает количество элементов
func (om *OrderedMap[K, V]) Len() int {
	om.mu.RLock()
	defer om.mu.RUnlock()

	return len(om.s)
}

// Clear очищает мапу, сохраняя текущую емкость
func (om *OrderedMap[K, V]) Clear() {
	om.mu.Lock()
	defer om.mu.Unlock()

	clear(om.m)
	om.s = om.s[:0]
}

// Clone создает полную копию OrderedMap
func (om *OrderedMap[K, V]) Clone() *OrderedMap[K, V] {
	om.mu.RLock()
	defer om.mu.RUnlock()

	newMap := NewOrderedMap[K, V](om.cap)
	for k, v := range om.m {
		newMap.m[k] = v
	}
	newMap.s = slices.Clone(om.s)
	return newMap
}

// Range итерируется по элементам, вызывая f для каждой пары ключ-значение.
// Если f возвращает false, итерация прекращается
func (om *OrderedMap[K, V]) Range(f func(K, V) bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, k := range om.s {
		if !f(k, om.m[k]) {
			break
		}
	}
}

// normalizeIndex корректирует отрицательные индексы и проверяет границы
func normalizeIndex(i, length int) int {
	if i < 0 {
		i = length + i
	}
	if i < 0 || i >= length {
		return -1
	}
	return i
}
