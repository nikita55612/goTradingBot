package models

// OrderResult содержит ответ API на создание ордера
type PlaceOrderResult struct {
	OrderId     string `json:"orderId"`     // ID ордера в системе Bybit
	OrderLinkId string `json:"orderLinkId"` // Пользовательский ID ордера (если был указан)
}

// CancelOrderResult содержит ответ API на отмену ордера
type CancelOrderResult struct {
	OrderId     string `json:"orderId"`     // ID ордера в системе Bybit
	OrderLinkId string `json:"orderLinkId"` // Пользовательский ID ордера (если был указан)
}

// OrderHistoryResult представляет ответ API со списком ордеров
type OrderHistoryResult struct {
	List           []OrderHistoryDetail `json:"list"`           // Список ордеров
	NextPageCursor string               `json:"nextPageCursor"` // Курсор для пагинации (токен следующей страницы)
	Category       string               `json:"category"`       // Тип продукта (категория)
}

// OrderHistoryDetail содержит детальную информацию об ордере
type OrderHistoryDetail struct {
	PositionIdx           int    `json:"positionIdx"`           // Индекс позиции (для идентификации в разных режимах)
	SmpGroup              int    `json:"smpGroup"`              // ID SMP-группы
	TriggerDirection      int    `json:"triggerDirection"`      // Направление триггера: 1 - рост, 2 - падение
	ReduceOnly            bool   `json:"reduceOnly"`            // Флаг уменьшения позиции (только уменьшение)
	CloseOnTrigger        bool   `json:"closeOnTrigger"`        // Флаг закрытия при срабатывании
	OrderId               string `json:"orderId"`               // ID ордера в системе Bybit
	OrderLinkId           string `json:"orderLinkId"`           // Пользовательский ID ордера
	BlockTradeId          string `json:"blockTradeId"`          // ID блок-сделки
	Symbol                string `json:"symbol"`                // Название символа (торговая пара)
	Price                 string `json:"price"`                 // Цена ордера
	Qty                   string `json:"qty"`                   // Количество
	Side                  string `json:"side"`                  // Направление сделки: Buy/Sell
	IsLeverage            string `json:"isLeverage"`            // Флаг использования кредитного плеча
	OrderStatus           string `json:"orderStatus"`           // Статус ордера
	CancelType            string `json:"cancelType"`            // Тип отмены ордера
	RejectReason          string `json:"rejectReason"`          // Причина отклонения ордера
	AvgPrice              string `json:"avgPrice"`              // Средняя цена исполнения
	LeavesQty             string `json:"leavesQty"`             // Оставшееся количество для исполнения
	LeavesValue           string `json:"leavesValue"`           // Оставшаяся стоимость для исполнения
	CumExecQty            string `json:"cumExecQty"`            // Кумулятивное исполненное количество
	CumExecValue          string `json:"cumExecValue"`          // Кумулятивная исполненная стоимость
	CumExecFee            string `json:"cumExecFee"`            // Кумулятивная комиссия за исполнение
	TimeInForce           string `json:"timeInForce"`           // Условие времени действия ордера
	OrderType             string `json:"orderType"`             // Тип ордера: Market/Limit
	StopOrderType         string `json:"stopOrderType"`         // Тип стоп-ордера
	OrderIv               string `json:"orderIv"`               // Подразумеваемая волатильность (для опционов)
	TriggerPrice          string `json:"triggerPrice"`          // Цена триггера
	TakeProfit            string `json:"takeProfit"`            // Цена тейк-профита
	StopLoss              string `json:"stopLoss"`              // Цена стоп-лосса
	TpTriggerBy           string `json:"tpTriggerBy"`           // Тип триггера для тейк-профита
	SlTriggerBy           string `json:"slTriggerBy"`           // Тип триггера для стоп-лосса
	TriggerBy             string `json:"triggerBy"`             // Тип цены триггера
	LastPriceOnCreated    string `json:"lastPriceOnCreated"`    // Последняя цена при создании ордера
	SmpType               string `json:"smpType"`               // Тип SMP-исполнения
	SmpOrderId            string `json:"smpOrderId"`            // ID SMP-ордера
	TpslMode              string `json:"tpslMode"`              // Режим TP/SL: Full/Partial
	TpLimitPrice          string `json:"tpLimitPrice"`          // Лимитная цена для TP
	SlLimitPrice          string `json:"slLimitPrice"`          // Лимитная цена для SL
	PlaceType             string `json:"placeType"`             // Тип размещения (для опционов)
	SlippageToleranceType string `json:"slippageToleranceType"` // Тип допуска проскальзывания
	SlippageTolerance     string `json:"slippageTolerance"`     // Значение допуска проскальзывания
	CreatedTime           string `json:"createdTime"`           // Время создания ордера (мс)
	UpdatedTime           string `json:"updatedTime"`           // Время обновления ордера (мс)
}
