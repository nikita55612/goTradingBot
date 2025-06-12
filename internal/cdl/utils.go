package cdl

import (
	"encoding/csv"
	"errors"
	"os"
	"strconv"
)

var (
	// csvColumns определяет обязательные колонки в CSV файле с данными свечей
	csvColumns = [7]string{
		"time",     // Временная метка
		"open",     // Цена открытия
		"high",     // Максимальная цена
		"low",      // Минимальная цена
		"close",    // Цена закрытия
		"volume",   // Объем торгов
		"turnover", // Оборот
	}

	// Ошибки обработки свечных данных
	errParseCandle = errors.New("cdl: ошибка парсинга свечи: неверный формат данных")
	errEmptyData   = errors.New("cdl: файл не содержит данных")
	errInvalidData = errors.New("cdl: неверный формат файла или заголовков")
)

// CandlesAsMap преобразует массив свечей в map с массивами значений по ключам
func CandlesAsMap(candles []Candle) map[string][]float64 {
	return map[string][]float64{
		"time":     ListOfCandleArg(candles, Time),
		"open":     ListOfCandleArg(candles, Open),
		"high":     ListOfCandleArg(candles, High),
		"low":      ListOfCandleArg(candles, Low),
		"close":    ListOfCandleArg(candles, Close),
		"volume":   ListOfCandleArg(candles, Volume),
		"turnover": ListOfCandleArg(candles, Turnover),
	}
}

// ParseCandleFromRawData парсит данные свечи из строки CSV
// Возвращает ошибку если количество полей не соответствует ожидаемому
func ParseCandleFromRawData(data [7]string) (candle Candle, err error) {
	candle.Time, err = strconv.ParseInt(data[0], 10, 64)
	if err != nil {
		return candle, errParseCandle
	}
	candle.O, err = strconv.ParseFloat(data[1], 64)
	if err != nil {
		return candle, errParseCandle
	}
	candle.H, err = strconv.ParseFloat(data[2], 64)
	if err != nil {
		return candle, errParseCandle
	}
	candle.L, err = strconv.ParseFloat(data[3], 64)
	if err != nil {
		return candle, errParseCandle
	}
	candle.C, err = strconv.ParseFloat(data[4], 64)
	if err != nil {
		return candle, errParseCandle
	}
	candle.Volume, err = strconv.ParseFloat(data[5], 64)
	if err != nil {
		return candle, errParseCandle
	}
	candle.Turnover, err = strconv.ParseFloat(data[6], 64)
	if err != nil {
		return candle, errParseCandle
	}
	return candle, nil
}

// CandlesFromCsv загружает свечи из CSV файла
// Проверяет соответствие структуры файла ожидаемому формату
func CandlesFromCsv(path string) ([]Candle, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) <= 1 {
		return nil, errEmptyData
	}
	if len(records[0]) != len(csvColumns) {
		return nil, errInvalidData
	}
	for i, col := range records[0] {
		if col != csvColumns[i] {
			return nil, errInvalidData
		}
	}
	candles := make([]Candle, 0, len(records)-1)
	for _, record := range records[1:] {
		candle, err := ParseCandleFromRawData([7]string(record))
		if err != nil {
			return candles, err
		}
		candles = append(candles, candle)
	}
	return candles, nil
}

// SaveCandlesToCsv сохраняет свечи в CSV файл
// Перезаписывает файл если он уже существует
func SaveCandlesToCsv(path string, candles []Candle) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write(csvColumns[:]); err != nil {
		return err
	}
	for _, candle := range candles {
		if err := writer.Write(candle.AsArr()[:]); err != nil {
			return err
		}
	}
	return nil
}

// CandlesFromRawData преобразует двумерный массив строк в массив свечей
func CandlesFromRawData(data [][7]string) ([]Candle, error) {
	candles := make([]Candle, len(data))
	for i, row := range data {
		candle, err := ParseCandleFromRawData(row)
		if err != nil {
			return candles, err
		}
		candles[i] = candle
	}
	return candles, nil
}

// CandlesToRawData преобразует массив свечей в двумерный массив строк
func CandlesToRawData(candles []Candle) [][7]string {
	rawData := make([][7]string, len(candles))
	for i, candle := range candles {
		rawData[i] = *candle.AsArr()
	}
	return rawData
}
