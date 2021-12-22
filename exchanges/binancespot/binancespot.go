package binancespot

import (
	"context"
	"fmt"
	"github.com/adshao/go-binance/v2"
	. "github.com/coinrust/crex"
	"github.com/coinrust/crex/utils"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type BinanceSpot struct {
	client *binance.Client
	symbol string
	ws     *BinanceSpotWebSocket
}

func (b *BinanceSpot) GetName() (name string) {
	return "binancespot"
}

func (b *BinanceSpot) GetTime() (tm int64, err error) {
	tm, err = b.client.NewServerTimeService().Do(context.Background())
	return
}

func (b *BinanceSpot) GetBalance(currency string) (result *Balance, err error) {
	var res *binance.Account
	res, err = b.client.NewGetAccountService().Do(context.Background())
	if err != nil {
		return
	}
	result = &Balance{}
	for _, v := range res.Balances {
		if v.Asset == currency { // USDT
			result.Equity = utils.ParseFloat64(v.Free) + utils.ParseFloat64(v.Locked)
			result.Available = utils.ParseFloat64(v.Free)
			break
		}
	}
	return
}

func (b *BinanceSpot) GetOrderBook(symbol string, depth int) (result *OrderBook, err error) {
	result = &OrderBook{}
	if depth <= 5 {
		depth = 5
	} else if depth <= 10 {
		depth = 10
	} else if depth <= 20 {
		depth = 20
	} else if depth <= 50 {
		depth = 50
	} else if depth <= 100 {
		depth = 100
	} else if depth <= 500 {
		depth = 500
	} else {
		depth = 1000
	}
	var res *binance.DepthResponse
	res, err = b.client.NewDepthService().
		Symbol(symbol).
		Limit(depth).
		Do(context.Background())
	if err != nil {
		return
	}
	for _, v := range res.Asks {
		result.Asks = append(result.Asks, Item{
			Price:  utils.ParseFloat64(v.Price),
			Amount: utils.ParseFloat64(v.Quantity),
		})
	}
	for _, v := range res.Bids {
		result.Bids = append(result.Bids, Item{
			Price:  utils.ParseFloat64(v.Price),
			Amount: utils.ParseFloat64(v.Quantity),
		})
	}
	result.Time = time.Now()
	return
}

func (b *BinanceSpot) IntervalKlinePeriod(period string) string {
	m := map[string]string{
		PERIOD_1WEEK: "7d",
	}
	if v, ok := m[period]; ok {
		return v
	}
	return period
}

func (b *BinanceSpot) GetRecords(symbol string, period string, from int64, end int64, limit int) (records []*Record, err error) {
	var res []*binance.Kline
	service := b.client.NewKlinesService().
		Symbol(symbol).
		Interval(b.IntervalKlinePeriod(period)).
		Limit(limit)
	if from > 0 {
		service = service.StartTime(from * 1000)
	}
	if end > 0 {
		service = service.EndTime(end * 1000)
	}
	res, err = service.Do(context.Background())
	if err != nil {
		return
	}
	for _, v := range res {
		records = append(records, &Record{
			Symbol:    symbol,
			Timestamp: time.Unix(0, v.OpenTime*int64(time.Millisecond)),
			Open:      utils.ParseFloat64(v.Open),
			High:      utils.ParseFloat64(v.High),
			Low:       utils.ParseFloat64(v.Low),
			Close:     utils.ParseFloat64(v.Close),
			Volume:    utils.ParseFloat64(v.Volume),
		})
	}
	return
}

func (b *BinanceSpot) SetContractType(currencyPair string, contractType string) (err error) {
	b.symbol = currencyPair
	return
}

func (b *BinanceSpot) GetContractID() (symbol string, err error) {
	return b.symbol, nil
}

func (b *BinanceSpot) SetLeverRate(value float64) (err error) {
	return
}

func (b *BinanceSpot) OpenLong(symbol string, orderType OrderType, price float64, size float64) (result *Order, err error) {
	return b.PlaceOrder(symbol, Buy, orderType, price, size)
}

func (b *BinanceSpot) OpenShort(symbol string, orderType OrderType, price float64, size float64) (result *Order, err error) {
	return b.PlaceOrder(symbol, Sell, orderType, price, size)
}

func (b *BinanceSpot) CloseLong(symbol string, orderType OrderType, price float64, size float64) (result *Order, err error) {
	return b.PlaceOrder(symbol, Sell, orderType, price, size, OrderReduceOnlyOption(true))
}

func (b *BinanceSpot) CloseShort(symbol string, orderType OrderType, price float64, size float64) (result *Order, err error) {
	return b.PlaceOrder(symbol, Buy, orderType, price, size, OrderReduceOnlyOption(true))
}

func (b *BinanceSpot) PlaceOrder(symbol string, direction Direction, orderType OrderType, price float64,
	size float64, opts ...PlaceOrderOption) (result *Order, err error) {
	params := ParsePlaceOrderParameter(opts...)
	service := b.client.NewCreateOrderService().
		Symbol(symbol).
		Quantity(fmt.Sprint(size))

	var side binance.SideType
	if direction == Buy {
		side = binance.SideTypeBuy
	} else if direction == Sell {
		side = binance.SideTypeSell
	}

	var _orderType binance.OrderType
	switch orderType {
	case OrderTypeLimit:
		_orderType = binance.OrderTypeLimit
	case OrderTypeMarket:
		_orderType = binance.OrderTypeMarket
	case OrderTypeStopMarket:
		_orderType = binance.OrderTypeStopLoss
		service = service.StopPrice(fmt.Sprint(params.StopPx))
	case OrderTypeStopLimit:
		_orderType = binance.OrderTypeStopLossLimit
		service = service.StopPrice(fmt.Sprint(params.StopPx))
	}

	if orderType != OrderTypeMarket {
		service = service.TimeInForce(resolveTimeInForce(params.TimeInForce))
		if price > 0 {
			service = service.Price(fmt.Sprint(price))
		}
	}

	service = service.Side(side).Type(_orderType)
	var res *binance.CreateOrderResponse
	res, err = service.Do(context.Background())
	if err != nil {
		return
	}
	result = b.convertOrder(res)
	return
}

func (b *BinanceSpot) GetOpenOrders(symbol string, opts ...OrderOption) (result []*Order, err error) {
	service := b.client.NewListOpenOrdersService().
		Symbol(symbol)
	var res []*binance.Order
	res, err = service.Do(context.Background())
	if err != nil {
		return
	}
	for _, v := range res {
		result = append(result, b.convertOrder1(v))
	}
	return
}

func (b *BinanceSpot) GetOrder(symbol string, id string, opts ...OrderOption) (result *Order, err error) {
	var orderID int64
	orderID, err = strconv.ParseInt(id, 10, 64)
	if err != nil {
		return
	}
	var res *binance.Order
	res, err = b.client.NewGetOrderService().
		Symbol(symbol).
		OrderID(orderID).
		Do(context.Background())
	if err != nil {
		return
	}
	result = b.convertOrder1(res)
	return
}

func (b *BinanceSpot) CancelOrder(symbol string, id string, opts ...OrderOption) (result *Order, err error) {
	var orderID int64
	orderID, err = strconv.ParseInt(id, 10, 64)
	if err != nil {
		return
	}
	var res *binance.CancelOrderResponse
	res, err = b.client.NewCancelOrderService().
		Symbol(symbol).
		OrderID(orderID).
		Do(context.Background())
	if err != nil {
		return
	}
	result = b.convertOrder2(res)
	return
}

func (b *BinanceSpot) CancelAllOrders(symbol string, opts ...OrderOption) (err error) {

	_, err = b.client.NewCancelOpenOrdersService().
		Symbol(symbol).
		Do(context.Background())

	return
}

func (b *BinanceSpot) AmendOrder(symbol string, id string, price float64, size float64, opts ...OrderOption) (result *Order, err error) {
	return
}

func (b *BinanceSpot) GetPositions(symbol string) (result []*Position, err error) {
	return
}

func (b *BinanceSpot) convertOrder2(order *binance.CancelOrderResponse) (result *Order) {
	result = &Order{}
	result.ID = fmt.Sprint(order.OrderID)
	result.ClientOId = order.ClientOrderID
	result.Symbol = order.Symbol
	result.Price = utils.ParseFloat64(order.Price)
	result.Amount = utils.ParseFloat64(order.OrigQuantity)
	result.Direction = b.convertDirection(order.Side)
	result.Type = b.convertOrderType(order.Type)
	result.AvgPrice = 0
	result.FilledAmount = utils.ParseFloat64(order.ExecutedQuantity)
	result.Status = b.orderStatus(order.Status)
	return
}

func (b *BinanceSpot) convertOrder(order *binance.CreateOrderResponse) (result *Order) {
	result = &Order{}
	result.ID = fmt.Sprint(order.OrderID)
	result.ClientOId = order.ClientOrderID
	result.Symbol = order.Symbol
	result.Price = utils.ParseFloat64(order.Price)
	result.Amount = utils.ParseFloat64(order.OrigQuantity)
	result.Direction = b.convertDirection(order.Side)
	result.Type = b.convertOrderType(order.Type)
	result.AvgPrice = utils.ParseFloat64(order.CummulativeQuoteQuantity) / utils.ParseFloat64(order.ExecutedQuantity)
	result.FilledAmount = utils.ParseFloat64(order.ExecutedQuantity)
	result.Time = time.Unix(order.TransactTime/int64(1e3), 0)

	result.Status = b.orderStatus(order.Status)
	return
}

func (b *BinanceSpot) convertOrder1(order *binance.Order) (result *Order) {
	result = &Order{}
	result.ID = fmt.Sprint(order.OrderID)
	result.ClientOId = order.ClientOrderID
	result.Symbol = order.Symbol
	result.Price = utils.ParseFloat64(order.Price)
	result.StopPx = utils.ParseFloat64(order.StopPrice)
	result.Amount = utils.ParseFloat64(order.OrigQuantity)
	result.Direction = b.convertDirection(order.Side)
	result.Type = b.convertOrderType(order.Type)
	result.AvgPrice = utils.ParseFloat64(order.CummulativeQuoteQuantity) / utils.ParseFloat64(order.ExecutedQuantity)
	result.FilledAmount = utils.ParseFloat64(order.ExecutedQuantity)

	result.Status = b.orderStatus(order.Status)
	result.Time = time.Unix(order.Time/int64(1e3), 0)
	result.UpdateTime = time.Unix(order.UpdateTime/int64(1e3), 0)
	return
}

func (b *BinanceSpot) orderStatus(status binance.OrderStatusType) OrderStatus {
	switch status {
	case binance.OrderStatusTypeNew:
		return OrderStatusNew
	case binance.OrderStatusTypePartiallyFilled:
		return OrderStatusPartiallyFilled
	case binance.OrderStatusTypeFilled:
		return OrderStatusFilled
	case binance.OrderStatusTypeCanceled:
		return OrderStatusCancelled
	case binance.OrderStatusTypeRejected:
		return OrderStatusRejected
	case binance.OrderStatusTypeExpired:
		return OrderStatusCancelled
	default:
		return OrderStatusCreated
	}
}

func (b *BinanceSpot) convertOrderType(orderType binance.OrderType) OrderType {
	/*
		OrderTypeTakeProfitMarket   OrderType = "TAKE_PROFIT_MARKET"
		OrderTypeTrailingStopMarket OrderType = "TRAILING_STOP_MARKET"
	*/
	switch orderType {
	case binance.OrderTypeLimit:
		return OrderTypeLimit
	case binance.OrderTypeMarket:
		return OrderTypeMarket
	case binance.OrderTypeStopLossLimit:
		return OrderTypeStopLimit
	case binance.OrderTypeStopLoss:
		return OrderTypeStopMarket
	default:
		return OrderTypeLimit
	}
}

func (b *BinanceSpot) convertDirection(side binance.SideType) Direction {
	switch side {
	case binance.SideTypeBuy:
		return Buy
	case binance.SideTypeSell:
		return Sell
	default:
		return Buy
	}
}

func resolveTimeInForce(timeInForce string) binance.TimeInForceType {
	var futuresTimeInForce binance.TimeInForceType

	switch timeInForce {
	case string(binance.TimeInForceTypeGTC):
		futuresTimeInForce = binance.TimeInForceTypeGTC
	case string(binance.TimeInForceTypeFOK):
		futuresTimeInForce = binance.TimeInForceTypeFOK
	case string(binance.TimeInForceTypeIOC):
		futuresTimeInForce = binance.TimeInForceTypeIOC
	default:
		futuresTimeInForce = binance.TimeInForceTypeGTC
	}

	return futuresTimeInForce
}

func (b *BinanceSpot) SetProxy(proxyURL string) error {
	proxyURL_, err := url.Parse(proxyURL)
	if err != nil {
		return err
	}

	//adding the proxy settings to the Transport object
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL_),
	}

	//adding the Transport object to the http Client
	b.client.HTTPClient.Transport = transport
	websocket.DefaultDialer = &websocket.Dialer{
		Proxy:            http.ProxyURL(proxyURL_),
		HandshakeTimeout: 45 * time.Second,
	}
	return nil
}

func (b *BinanceSpot) SubscribeTrades(market Market, callback func(trades []*Trade)) error {
	return b.ws.SubscribeTrades(market, callback)
}

func (b *BinanceSpot) SubscribeLevel2Snapshots(market Market, callback func(ob *OrderBook)) error {
	return ErrNotImplemented
}

func (b *BinanceSpot) SubscribeOrders(market Market, callback func(orders []*Order)) error {
	return ErrNotImplemented
}

func (b *BinanceSpot) SubscribePositions(market Market, callback func(positions []*Position)) error {
	return ErrNotImplemented
}

func (b *BinanceSpot) IO(name string, params string) (string, error) {
	return "", nil
}

func NewBinanceSpot(params *Parameters) *BinanceSpot {
	binance.UseTestnet = params.Testnet
	client := binance.NewClient(params.AccessKey, params.SecretKey)
	b := &BinanceSpot{
		client: client,
	}
	if params.ProxyURL != "" {
		b.SetProxy(params.ProxyURL)
	}

	return b
}
