package cdl

// import (
// 	"context"
// 	"goTradingBotDev/utils/seqs"
// 	"sync"
// 	"time"

// 	"github.com/google/uuid"
// )

// // CandleProvider определяет интерфейс для работы с поставщиком свечных данных
// type CandleProvider interface {
// 	CandleStream(ctx context.Context, symbol string, interval Interval) (<-chan *CandleStreamData, error)
// 	GetCandles(symbol string, interval Interval, limit int) ([]Candle, error)
// }

// // subscriber содержит каналы для подписчика свечных данных
// type subscriber struct {
// 	ch   chan<- *CandleStreamData // Канал для отправки данных подписчику
// 	done <-chan struct{}          // Канал для отмены подписки
// }

// // CandleSync синхронизирует свечные данные от провайдера и управляет подписками
// type CandleSync struct {
// 	provider    CandleProvider
// 	candles     *seqs.SyncBuffer[Candle]
// 	subscribers map[string]subscriber
// 	stream      <-chan *CandleStreamData
// 	lastCandle  Candle
// 	ctx         context.Context
// 	mu          sync.Mutex
// 	wg          sync.WaitGroup
// 	Symbol      string
// 	Interval    Interval
// 	bufferSize  int
// }

// // NewCandleSync создает новый экземпляр CandleSync
// func NewCandleSync(ctx context.Context, symbol string, interval Interval, bufferSize int, provider CandleProvider) *CandleSync {
// 	if bufferSize <= 1 {
// 		bufferSize = 2
// 	}
// 	return &CandleSync{
// 		Symbol:      symbol,
// 		Interval:    interval,
// 		provider:    provider,
// 		candles:     seqs.NewCircularBuffer[Candle](bufferSize),
// 		subscribers: make(map[string]subscriber),
// 		ctx:         ctx,
// 		bufferSize:  bufferSize,
// 	}
// }

// // StartSync начинает синхронизацию свечных данных
// func (s *CandleSync) StartSync() error {
// 	// Подключаемся к потоку свечей
// 	stream, err := s.provider.CandleStream(s.ctx, s.Symbol, s.Interval)
// 	if err != nil {
// 		return err
// 	}
// 	// Получаем исторические свечи
// 	candles, err := s.provider.GetCandles(s.Symbol, s.Interval, s.bufferSize)
// 	if err != nil {
// 		return err
// 	}

// 	s.candles.Write(candles[:len(candles)-1]...)
// 	s.stream = stream

// 	// Запускаем обработку в фоне
// 	go func() {
// 		defer s.close()

// 		s.wg.Add(2)
// 		go s.startStreamProcessor()       // Обработка потока свечей
// 		go s.startMissingCandlesChecker() // Проверка пропущенных свечей
// 		s.wg.Wait()
// 	}()

// 	go func() {
// 		time.Sleep(time.Second)
// 		candles, err := s.provider.GetCandles(s.Symbol, s.Interval, 2)
// 		if err != nil {
// 			return
// 		}
// 		s.tryAddNewCandle(candles[0])
// 	}()

// 	return nil
// }

// // Subscribe добавляет нового подписчика на свечные данные
// func (s *CandleSync) Subscribe(ch chan<- *CandleStreamData) chan<- struct{} {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	done := make(chan struct{}, 1)
// 	id := uuid.NewString()
// 	s.subscribers[id] = subscriber{ch: ch, done: done}
// 	return done
// }

// // tryAddNewCandle добавляет новую свечу в буфер, если она соответствует интервалу
// func (s *CandleSync) tryAddNewCandle(candle Candle) bool {
// 	lastCandle := s.candles.ReadIndex(-1)
// 	timeDiff := candle.Time - lastCandle.Time

// 	// Проверяем, что свеча соответствует ожидаемому интервалу
// 	if timeDiff > 10 && int(timeDiff) < s.Interval.AsMilli()+10 {
// 		s.candles.AsyncWrite(candle)
// 		return true
// 	}
// 	return false
// }

// // countMissingCandles подсчитывает количество пропущенных свечей
// func (s *CandleSync) countMissingCandles() int {
// 	var missingCount int
// 	maxAllowedGap := float64(s.Interval.AsSeconds() + 1)

// 	s.candles.WithLock(func(candles []Candle) {
// 		for i := len(candles) - 1; i > 0; i-- {
// 			endTime := time.Unix(candles[i].Time/1000, 0)
// 			secondsSince := time.Since(endTime).Seconds()
// 			if secondsSince < maxAllowedGap {
// 				break
// 			}
// 			missingCount++
// 		}
// 	})
// 	return missingCount
// }

// // removeSubscriber удаляет подписчика по ключу
// func (s *CandleSync) removeSubscriber(key string) {
// 	if sub, exists := s.subscribers[key]; exists {
// 		close(sub.ch) // Закрываем канал подписчика
// 		delete(s.subscribers, key)
// 	}
// }

// // broadcastToSubscribers рассылает данные всем подписчикам
// func (s *CandleSync) broadcastToSubscribers(data *CandleStreamData) {
// 	for key, sub := range s.subscribers {
// 		select {
// 		case <-sub.done:
// 			// Удаляем отписавшегося подписчика
// 			go func() {
// 				s.mu.Lock()
// 				defer s.mu.Unlock()
// 				s.removeSubscriber(key)
// 			}()
// 		case sub.ch <- data: // Отправляем данные подписчику
// 		default:
// 			// Пропускаем, если канал переполнен
// 		}
// 	}
// }

// // startMissingCandlesChecker периодически проверяет наличие пропущенных свечей
// func (s *CandleSync) startMissingCandlesChecker() {
// 	defer s.wg.Done()

// 	checkInterval := time.Duration(s.Interval.AsMilli()/10) * time.Millisecond
// 	ticker := time.NewTicker(checkInterval)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-s.ctx.Done():
// 			return
// 		case <-ticker.C:
// 			missingCount := s.countMissingCandles()
// 			if missingCount <= 0 {
// 				continue
// 			}
// 			// Запрашиваем пропущенные свечи
// 			missingCandles, err := s.provider.GetCandles(s.Symbol, s.Interval, missingCount+1)
// 			if err != nil || len(missingCandles) <= 1 {
// 				continue
// 			}
// 			// Обрабатываем полученные свечи
// 			missingCandles = missingCandles[:len(missingCandles)-1]
// 			for _, candle := range missingCandles {
// 				if !s.tryAddNewCandle(candle) {
// 					continue
// 				}
// 				s.mu.Lock()
// 				timeDiff := candle.Time - s.lastCandle.Time
// 				if timeDiff > 10 && int(timeDiff) < s.Interval.AsMilli()+10 {
// 					s.broadcastToSubscribers(&CandleStreamData{
// 						Interval: s.Interval,
// 						Confirm:  true,
// 						Candle:   candle,
// 					})
// 					s.lastCandle = candle
// 				}
// 				s.mu.Unlock()
// 			}
// 		}
// 	}
// }

// // startStreamProcessor обрабатывает входящий поток свечей
// func (s *CandleSync) startStreamProcessor() {
// 	defer s.wg.Done()

// 	for {
// 		select {
// 		case <-s.ctx.Done():
// 			return
// 		case data := <-s.stream:
// 			if data == nil || data.Interval != s.Interval {
// 				continue
// 			}
// 			s.mu.Lock()
// 			s.broadcastToSubscribers(data)
// 			if data.Confirm {
// 				s.tryAddNewCandle(data.Candle)
// 				s.lastCandle = data.Candle
// 			}
// 			s.mu.Unlock()
// 		}
// 	}
// }

// // GetCandles возвращает последние свечи
// func (s *CandleSync) GetCandles(limit int) []Candle {
// 	return s.candles.Read(limit)
// }

// // close завершает работу CandleSync и освобождает ресурсы
// func (s *CandleSync) close() {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	s.candles.Close()
// 	for key := range s.subscribers {
// 		s.removeSubscriber(key)
// 	}
// }
