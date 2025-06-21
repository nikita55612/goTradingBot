package models

// AccountInfo содержит информацию об аккаунте
type AccountInfo struct {
	UnifiedMarginStatus int    `json:"unifiedMarginStatus"` // Статус унифицированной маржи (0: обычный аккаунт, 1: унифицированная маржа)
	TimeWindow          int    `json:"timeWindow"`          // Временное окно (устарело)
	SmpGroup            int    `json:"smpGroup"`            // SMP группа (устарело)
	IsMasterTrader      bool   `json:"isMasterTrader"`      // Является ли аккаунт мастер-аккаунтом
	MarginMode          string `json:"marginMode"`          // Режим маржи (ISOLATED_MARGIN, REGULAR_MARGIN и т.д.)
	SpotHedgingStatus   string `json:"spotHedgingStatus"`   // Статус хеджирования спот-позиций
	UpdatedTime         string `json:"updatedTime"`         // Время последнего обновления
	DcpStatus           string `json:"dcpStatus"`           // Статус DCP (устарело)
}
