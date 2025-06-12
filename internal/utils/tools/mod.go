package tools

import (
	"fmt"
	"os"
	"reflect"
	"time"
)

// SliceOfAny преобразует слайс любого типа в слайс interface{}
func SliceOfAny[T any](s []T) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}

// GetField извлекает значение поля из структуры по имени
func GetField[T any](obj any, fieldName string) (T, error) {
	var zero T
	val := reflect.ValueOf(obj)
	if val.Kind() != reflect.Struct {
		return zero, fmt.Errorf("GetField: переданный объект не является структурой")
	}
	field := val.FieldByName(fieldName)
	if !field.IsValid() {
		return zero, fmt.Errorf("GetField: поле '%s' не найдено в структуре", fieldName)
	}
	fieldValue, ok := field.Interface().(T)
	if !ok {
		return zero, fmt.Errorf(
			"GetField: нельзя преобразовать поле '%s' (тип %T) в тип %T",
			fieldName, field.Interface(), zero,
		)
	}
	return fieldValue, nil
}

// TimestampToString преобразует Unix timestamp (в миллисекундах) в строку в формате RFC3339
func TimestampToString(ts int64) string {
	s, _ := time.Unix(ts/1000, 0).MarshalText()
	return string(s)
}

// PathExists проверяет существует ли путь
func PathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func Try(callback func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("unexpected error: %+v", r)
			}
		}
	}()

	err = callback()

	return
}
