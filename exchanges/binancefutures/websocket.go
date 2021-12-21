package binancefutures

import (
	"fmt"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/chuckpreslar/emission"
	. "github.com/coinrust/crex"
	"github.com/coinrust/crex/utils"
	"strconv"
)

type BinanceFuturesWebSocket struct {
	emitter *emission.Emitter
}

func (s *BinanceFuturesWebSocket) aggTradeCallback(event *futures.WsAggTradeEvent)  {

	var direction Direction
	if event.Maker {
		direction = Sell
	} else {
		direction = Buy
	}

	var result []*Trade
	t := Trade{
		ID: strconv.FormatInt(event.LastTradeID, 10),
		Direction: direction,
		Price: utils.ParseFloat64(event.Price),
		Amount: utils.ParseFloat64(event.Quantity),
		Ts: event.TradeTime,
		Symbol: event.Symbol,
	}
	result = append(result, &t)
	s.emitter.Emit(WSEventTrade, result)
}

func (s *BinanceFuturesWebSocket) SubscribeTrades(market Market, callback func(trades []*Trade)) error {
	s.emitter.On(WSEventTrade, callback)

	_, _, err := futures.WsAggTradeServe(market.Symbol, s.aggTradeCallback, nil)
	if err != nil {
		fmt.Println(err)
		return err
	}
	//<-doneC
	return nil
}
