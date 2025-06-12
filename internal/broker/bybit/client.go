package bybit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/nikita55612/httpx"
)

const (
	PUBLICWS   = "wss://stream.bybit.com/v5/public"
	MAINNET    = "https://api.bybit.com"
	MAINNETALT = "https://api.bytick.com"
	TESTNET    = "https://api-testnet.bybit.com"
)

// ServerResponse представляет структуру стандартного ответа от API Bybit
type ServerResponse struct {
	RetCode    int      `json:"retCode"`    // RetCode - код возврата, где 0 означает успешный запрос
	RetMsg     string   `json:"retMsg"`     // RetMsg - сообщение от сервера ("OK", "SUCCESS" или описание ошибки)
	Result     any      `json:"result"`     // Result - основные данные ответа, тип зависит от конкретного запроса
	RetExtInfo struct{} `json:"retExtInfo"` // RetExtInfo - дополнительная информация (обычно пустой объект)
	Time       int64    `json:"time"`       // Time - временная метка сервера в миллисекундах
}

// Client представляет клиент для работы с REST API Bybit
type Client struct {
	baseURL    string          // базовый URL API (тестовая или основная сеть)
	apiKey     string          // публичный API-ключ для аутентификации
	apiSecret  string          // секретный ключ для подписи запросов (HMAC)
	recvWindow int             // временное окно валидности запроса в миллисекундах
	category   string          // spot/linear/inverse
	ctx        context.Context // контекст для выполнения запросов
	timeout    time.Duration   // таймаут HTTP-запросов
}

// NewClient создает новый экземпляр клиента для работы с API Bybit
// Загружает учетные данные из .env файла (API_KEY и API_SECRET)
// Принимает опциональные параметры конфигурации через Option функции
func NewClient(apiKey, apiSecret string, opts ...Option) *Client {
	client := &Client{
		baseURL:    MAINNET,
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		recvWindow: 5000,
		category:   "spot",
		timeout:    5 * time.Second,
	}
	for _, option := range opts {
		option(client)
	}
	return client
}

func NewClientFromEnv(opts ...Option) *Client {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("%s: NewClientFromEnv: ошибка загрузки .env файла", errorTitel)
	}
	apiKey := os.Getenv("BYBIT_API_KEY")
	if apiKey == "" {
		log.Fatalf("%s: NewClientFromEnv: не указан BYBIT_API_KEY", errorTitel)
	}
	apiSecret := os.Getenv("BYBIT_API_SECRET")
	if apiSecret == "" {
		log.Fatalf("%s: NewClientFromEnv: не указан BYBIT_API_SECRET", errorTitel)
	}
	return NewClient(apiKey, apiSecret, opts...)
}

// Option определяет тип функции для настройки Client
type Option func(*Client)

// WithContext устанавливает контекст для выполнения запросов
func WithContext(ctx context.Context) Option {
	return func(c *Client) {
		c.ctx = ctx
	}
}

// WithTimeout устанавливает таймаут для HTTP-запросов
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithRecvWindow устанавливает пользовательское значение recvWindow
// recvWindow - временное окно валидности запроса в миллисекундах
func WithRecvWindow(recvWindow int) Option {
	return func(c *Client) {
		c.recvWindow = recvWindow
	}
}

// WithBaseURL устанавливает пользовательский базовый URL API
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithCategory устанавливает категорию (spot, linear, inverse)
func WithCategory(category string) Option {
	return func(c *Client) {
		c.category = category
	}
}

func (c *Client) callAPI(req httpx.RequestBuilder, queryString string, result any) error {
	timestamp := strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	signature := fmt.Sprintf("%s%s%d%s", timestamp, c.apiKey, c.recvWindow, queryString)
	mac := hmac.New(sha256.New, []byte(c.apiSecret))
	if _, err := mac.Write([]byte(signature)); err != nil {
		err := fmt.Errorf("error when creating the request signature: %w", err)
		return NewError(UnknownErrorT, err)
	}
	signature = hex.EncodeToString(mac.Sum(nil))
	req = req.WithHeader(
		"X-BAPI-API-KEY", c.apiKey,
		"X-BAPI-TIMESTAMP", timestamp,
		"X-BAPI-SIGN", signature,
		"X-BAPI-RECV-WINDOW", c.recvWindow,
		"Content-Type", "application/json",
		"Accept", "application/json",
	)
	if c.ctx != nil {
		req = req.WithContext(c.ctx)
	}
	if c.timeout > 0 {
		req = req.WithTimeout(c.timeout)
	}
	res, err := req.Build().Do()
	if err != nil {
		return NewError(RequestErrorT, err)
	}
	defer res.Close()

	var serverResponse ServerResponse
	if err := res.UnmarshalBody(&serverResponse); err != nil {
		return NewError(SerDeErrorT, err)
	}
	if err := ErrorFromServerResponse(&serverResponse); err.ServerResponseCode() != 0 {
		return err
	}
	data, err := json.Marshal(serverResponse.Result)
	if err != nil {
		return NewError(SerDeErrorT, err)
	}
	if result != nil {
		if err := json.Unmarshal(data, result); err != nil {
			return NewError(SerDeErrorT, err)
		}
	}
	return nil
}
