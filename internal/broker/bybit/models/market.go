package models

// CandleResult представляет данные свечи за определенный период времени.
type CandleResult struct {
	Category string      `json:"category"` // тип продукта (например, "inverse" - обратный контракт)
	Symbol   string      `json:"symbol"`   // название символа (например, "BTCUSD")
	List     [][7]string `json:"list"`     // массив данных свечей, отсортированный в обратном порядке по времени
}

// InstrumentInfoResult представляет ответ API с информацией об инструментах
type InstrumentInfoResult struct {
	Category string           `json:"category"` // Категория инструментов (spot, linear, inverse)
	List     []InstrumentInfo `json:"list"`     // Список инструментов
}

// InstrumentInfo представляет информацию о торговой паре (спот, фьючерсы, перпетуальные контракты)
type InstrumentInfo struct {
	Symbol             string `json:"symbol"`             // Название торговой пары (например BTCUSDT)
	ContractType       string `json:"contractType"`       // Тип контракта (LinearPerpetual, InversePerpetual и т.д.)
	Status             string `json:"status"`             // Статус инструмента (Trading, PreLaunch и т.д.)
	BaseCoin           string `json:"baseCoin"`           // Базовая монета (например BTC)
	QuoteCoin          string `json:"quoteCoin"`          // Котируемая монета (например USDT)
	LaunchTime         string `json:"launchTime"`         // Время запуска в timestamp (мс)
	DeliveryTime       string `json:"deliveryTime"`       // Время доставки для фьючерсов (0 для перпетуальных)
	DeliveryFeeRate    string `json:"deliveryFeeRate"`    // Ставка комиссии за доставку
	PriceScale         string `json:"priceScale"`         // Масштаб цены (количество знаков после запятой)
	Innovation         string `json:"innovation"`         // Является ли инструментом зоны инноваций (0: нет, 1: да)
	MarginTrading      string `json:"marginTrading"`      // Доступна ли маржинальная торговля
	StTag              string `json:"stTag"`              // Специальный тег (0: нет, 1: да)
	SettleCoin         string `json:"settleCoin"`         // Монета расчетов
	CopyTrading        string `json:"copyTrading"`        // Доступность копи-трейдинга (none, both и т.д.)
	UpperFundingRate   string `json:"upperFundingRate"`   // Верхний предел ставки фандинга
	LowerFundingRate   string `json:"lowerFundingRate"`   // Нижний предел ставки фандинга
	DisplayName        string `json:"displayName"`        // Отображаемое имя в UI
	FundingInterval    int    `json:"fundingInterval"`    // Интервал фандинга (в минутах)
	IsPreListing       bool   `json:"isPreListing"`       // Является ли премаркет-контрактом
	UnifiedMarginTrade bool   `json:"unifiedMarginTrade"` // Поддержка унифицированной маржи

	LeverageFilter struct {
		MinLeverage  string `json:"minLeverage"`  // Минимальное плечо
		MaxLeverage  string `json:"maxLeverage"`  // Максимальное плечо
		LeverageStep string `json:"leverageStep"` // Шаг изменения плеча
	} `json:"leverageFilter"`

	LotSizeFilter struct {
		BasePrecision       string `json:"basePrecision"`       // Точность базовой монеты
		QuotePrecision      string `json:"quotePrecision"`      // Точность котируемой монеты
		MinOrderQty         string `json:"minOrderQty"`         // Минимальное количество для ордера
		MaxOrderQty         string `json:"maxOrderQty"`         // Максимальное количество для Limit и PostOnly ордера
		MaxMktOrderQty      string `json:"maxMktOrderQty"`      // Максимальное количество для Market ордера
		MinOrderAmt         string `json:"minOrderAmt"`         // Минимальная сумма ордера
		MaxOrderAmt         string `json:"maxOrderAmt"`         // Максимальная сумма ордера
		QtyStep             string `json:"qtyStep"`             // Шаг изменения количества
		MinNotionalValue    string `json:"minNotionalValue"`    // Минимальная номинальная стоимость
		PostOnlyMaxOrderQty string `json:"postOnlyMaxOrderQty"` // Устарело, использовать maxOrderQty
	} `json:"lotSizeFilter"`

	PriceFilter struct {
		MinPrice string `json:"minPrice"` // Минимальная цена ордера
		MaxPrice string `json:"maxPrice"` // Максимальная цена ордера
		TickSize string `json:"tickSize"` // Шаг изменения цены (tick size)
	} `json:"priceFilter"`

	RiskParameters struct {
		PriceLimitRatioX string `json:"priceLimitRatioX"` // Коэффициент лимита цены X
		PriceLimitRatioY string `json:"priceLimitRatioY"` // Коэффициент лимита цены Y
	} `json:"riskParameters"`

	PreListingInfo *struct {
		CurAuctionPhase string `json:"curAuctionPhase"` // Текущая фаза аукциона
		Phases          []struct {
			Phase     string `json:"phase"`     // Фаза премаркет-трейдинга
			StartTime string `json:"startTime"` // Время начала фазы (timestamp мс)
			EndTime   string `json:"endTime"`   // Время окончания фазы (timestamp мс)
		} `json:"phases"` // Информация о фазах
		AuctionFeeInfo struct {
			AuctionFeeRate string `json:"auctionFeeRate"` // Ставка комиссии во время аукциона
			TakerFeeRate   string `json:"takerFeeRate"`   // Ставка тейкера в фазе непрерывного трейдинга
			MakerFeeRate   string `json:"makerFeeRate"`   // Ставка мейкера в фазе непрерывного трейдинга
		} `json:"auctionFeeInfo"` // Информация о комиссиях
	} `json:"preListingInfo"` // Информация о премаркете (если isPreListing=true)
}

// CandleStreamRawData представляет потоковые данные свечи
type CandleStreamRawData struct {
	Ts    int64  `json:"ts"`    // Временная метка
	Type  string `json:"type"`  // Тип сообщения
	Topic string `json:"topic"` // Топик подписки

	Data []struct {
		Start     int64  `json:"start"`     // Начальное время
		End       int64  `json:"end"`       // Конечное время
		Timestamp int64  `json:"timestamp"` // Временная метка
		Interval  string `json:"interval"`  // Интервал
		Open      string `json:"open"`      // Цена открытия
		Close     string `json:"close"`     // Цена закрытия
		High      string `json:"high"`      // Максимальная цена
		Low       string `json:"low"`       // Минимальная цена
		Volume    string `json:"volume"`    // Объем
		Turnover  string `json:"turnover"`  // Оборот
		Confirm   bool   `json:"confirm"`   // Подтверждение
	} `json:"data"`
}
