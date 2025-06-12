package saveform

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"reflect"
	"slices"
	"strconv"

	"golang.org/x/exp/constraints"
)

type Number interface {
	constraints.Integer | constraints.Float
}

// ToCSV сохраняет данные из map в CSV файл.
// Ключи map становятся заголовками столбцов.
// Все столбцы должны иметь одинаковую длину (по длине первого столбца).
func ToCSV[V Number](path string, cols map[string][]V) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	keys := slices.Collect(maps.Keys(cols))
	if len(keys) == 0 {
		return fmt.Errorf("SaveToCSV: отсутсвуют колонки для сохранения")
	}
	err = writer.Write(keys)
	if err != nil {
		return err
	}
	for i := 0; i < len(cols[keys[0]]); i++ {
		row := make([]string, len(keys))
		for n := 0; n < len(keys); n++ {
			col := cols[keys[n]]
			if len(col)-1 < i {
				row[n] = ""
				continue
			}
			v := cols[keys[n]][i]
			row[n] = formatValue(v)
		}
		err = writer.Write(row)
		if err != nil {
			return err
		}
	}
	return nil
}

// ColumnsToCSV сохраняет несколько столбцов данных в CSV файл.
// cols - слайс слайсов с данными, columnNames - названия столбцов.
// Возвращает ошибку если количество названий не совпадает с количеством столбцов.
func ColumnsToCSV[V Number](path string, cols [][]V, headers []string) error {
	if len(cols) == 0 {
		return fmt.Errorf("SaveColumnsToCSV: отсутсвуют колонки для сохранения")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if headers != nil {
		if len(headers) != len(cols) {
			return fmt.Errorf(
				"SaveColumnsToCSV: количество заголовков (%d) не соответствует количеству столбцов (%d)",
				len(headers),
				len(cols),
			)
		}
		err = writer.Write(headers)
		if err != nil {
			return err
		}
	}
	maxRows := 0
	for _, col := range cols {
		if len(col) > maxRows {
			maxRows = len(col)
		}
	}
	for i := 0; i < maxRows; i++ {
		row := make([]string, len(cols))
		for n := 0; n < len(cols); n++ {
			if len(cols[n])-1 < i {
				row[n] = ""
				continue
			}
			v := cols[n][i]
			row[n] = formatValue(v)
		}
		err = writer.Write(row)
		if err != nil {
			return err
		}
	}
	return nil
}

// formatValue преобразует числовое значение в строку.
func formatValue[V Number](v V) string {
	value := reflect.ValueOf(v)
	switch value.Kind() {
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(value.Float(), 'f', -1, 64)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(value.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(value.Uint(), 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ToJSON сохраняет данные в JSON файл с отступами (pretty print).
func ToJSON(path string, data any) error {
	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	err = os.WriteFile(path, jsonData, 0644)
	if err != nil {
		return err
	}
	return nil
}
