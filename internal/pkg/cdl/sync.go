package cdl

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nikita55612/goTradingBot/internal/utils/seqs"
)

// CandleProvider определяет интерфейс для работы с поставщиком свечных данных
type CandleProvider interface {
	CandleStream(ctx context.Context, symbol string, interval Interval) (<-chan *CandleStreamData, error)
	GetCandles(symbol string, interval Interval, limit int) ([]Candle, error)
}

// subscriber содержит каналы для подписчика свечных данных
type subscriber struct {
	ch   chan<- *CandleStreamData
	done <-chan struct{}
}

// CandleSync синхронизирует свечные данные от провайдера и управляет подписками
type CandleSync struct {
	Symbol       string
	Interval     Interval
	provider     CandleProvider
	ctx          context.Context
	candles      *seqs.SyncBuffer[Candle]
	bufferSize   int
	confirmTime  int64
	writeConfirm chan *Candle
	confirmWg    sync.WaitGroup
	sendToSubs   chan *CandleStreamData
	subscribers  map[string]subscriber
	subRWMu      sync.RWMutex
	stream       <-chan *CandleStreamData
}

// NewCandleSync создает новый экземпляр CandleSync
func NewCandleSync(ctx context.Context, symbol string, interval Interval, bufferSize int, provider CandleProvider) *CandleSync {
	if bufferSize <= 1 {
		bufferSize = 2
	}

	return &CandleSync{
		Symbol:       symbol,
		Interval:     interval,
		provider:     provider,
		ctx:          ctx,
		candles:      seqs.NewSyncBuffer[Candle](bufferSize),
		bufferSize:   bufferSize,
		writeConfirm: make(chan *Candle),
		sendToSubs:   make(chan *CandleStreamData, 2),
		subscribers:  make(map[string]subscriber),
	}
}

// StartSync начинает синхронизацию свечных данных
func (s *CandleSync) StartSync() error {
	// Подключаемся к потоку свечей
	stream, err := s.provider.CandleStream(s.ctx, s.Symbol, s.Interval)
	if err != nil {
		return err
	}
	// Получаем исторические свечи
	candles, err := s.provider.GetCandles(s.Symbol, s.Interval, s.bufferSize)
	if err != nil {
		return err
	}

	s.candles.Write(candles[:len(candles)-1]...)
	s.confirmTime = candles[len(candles)-1].Time
	s.stream = stream

	// Запускаем обработку в фоне
	go s.confirmWriter()
	go s.subMessenger()
	go s.streamProcessor()

	go func() {
		time.Sleep(time.Second)
		candles, err := s.provider.GetCandles(s.Symbol, s.Interval, 2)
		if err != nil {
			return
		}
		s.confirmWg.Add(1)
		s.writeConfirm <- &candles[0]
	}()

	return nil
}

// Subscribe добавляет нового подписчика на свечные данные
func (s *CandleSync) Subscribe(ch chan<- *CandleStreamData) chan<- struct{} {
	s.subRWMu.Lock()
	defer s.subRWMu.Unlock()

	done := make(chan struct{}, 1)
	id := uuid.NewString()
	s.subscribers[id] = subscriber{ch: ch, done: done}

	return done
}

// removeSubscriber удаляет подписчика по ключу
func (s *CandleSync) removeSubscriber(key string) {
	s.subRWMu.Lock()
	defer s.subRWMu.Unlock()

	if sub, exists := s.subscribers[key]; exists {
		close(sub.ch) // Закрываем канал подписчика
		delete(s.subscribers, key)
	}
}

// confirmWriter добавляет новую свечу в буфер, если она соответствует интервалу
func (s *CandleSync) confirmWriter() {
	intervalMs := s.Interval.AsMilli()

	for candle := range s.writeConfirm {
		missCount := int(candle.Time-s.confirmTime+5) / intervalMs
		if missCount <= 0 {
			s.confirmWg.Done()
			continue
		}

		if missCount == 1 {
			s.confirmTime = candle.Time
			s.candles.Write(*candle)
		} else {
			missCandles, err := s.provider.GetCandles(
				s.Symbol, s.Interval, missCount+2,
			)
			n := len(missCandles)
			if err != nil || n <= 1 {
				s.confirmWg.Done()
				continue
			}

			for i, c := range missCandles[:n-1] {
				if int(c.Time-s.confirmTime+2) >= 0 {
					s.candles.Write(missCandles[i : n-1]...)
					break
				}
			}
			s.confirmTime = missCandles[n-1].Time
		}
		s.confirmWg.Done()
	}
}

// subMessenger рассылает данные всем подписчикам
func (s *CandleSync) subMessenger() {
	for data := range s.sendToSubs {
		s.subRWMu.RLock()
		for key, sub := range s.subscribers {
			select {
			case <-sub.done:
				// Удаляем отписавшегося подписчика
				go s.removeSubscriber(key)
			case sub.ch <- data: // Отправляем данные подписчику
			default:
			}
		}
		s.subRWMu.RUnlock()
	}

}

// streamProcessor обрабатывает входящий поток свечей
func (s *CandleSync) streamProcessor() {
	defer s.close()

	for data := range s.stream {
		if data == nil {
			continue
		}
		if data.Confirm {
			s.confirmWg.Add(1)
			s.writeConfirm <- &data.Candle
		}
		s.sendToSubs <- data
	}
}

// ReadConfirmCandles возвращает последние свечи
func (s *CandleSync) ReadConfirmCandles(limit int) []Candle {
	s.confirmWg.Wait()
	return s.candles.Read(limit)
}

// close завершает работу CandleSync и освобождает ресурсы
func (s *CandleSync) close() {
	s.subRWMu.Lock()
	defer s.subRWMu.Unlock()

	s.candles.Close()
	close(s.writeConfirm)
	close(s.sendToSubs)

	for key := range s.subscribers {
		go s.removeSubscriber(key)
	}
}
