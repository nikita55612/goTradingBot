package bybit

import (
	"fmt"

	"github.com/nikita55612/goTradingBot/internal/broker/bybit/models"
	"github.com/nikita55612/httpx"
)

// GetAccountInfo возвращает информацию об учетной записи.
// https://bybit-exchange.github.io/docs/v5/account/account-info
func (c *Client) GetAccountInfo() (*models.AccountInfo, error) {
	path := fmt.Sprintf(
		"%s%s",
		c.baseURL,
		"/v5/account/info",
	)
	req := httpx.Get(path)
	var accountInfo models.AccountInfo
	if err := c.callAPI(req, "", &accountInfo); err != nil {
		return nil, err.(*Error).SetEndpoint("GetAccountInfo")
	}

	return &accountInfo, nil
}
