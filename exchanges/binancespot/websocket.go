package binancespot

import (
	"fmt"
	"github.com/adshao/go-binance/v2"
	"github.com/chuckpreslar/emission"
	. "github.com/coinrust/crex"
	"github.com/coinrust/crex/utils"
	"strconv"
)

type BinanceSpotWebSocket struct {
	emitter *emission.Emitter
}

func (s *BinanceSpotWebSocket) tradeCallback(event *binance.WsTradeEvent)  {

	var direction Direction
	if event.IsBuyerMaker {
		direction = Sell
	} else {
		direction = Buy
	}

	var result []*Trade
	t := Trade{
		ID: strconv.FormatInt(event.TradeID, 10),
		Direction: direction,
		Price: utils.ParseFloat64(event.Price),
		Amount: utils.ParseFloat64(event.Quantity),
		Ts: event.TradeTime,
		Symbol: event.Symbol,
	}
	result = append(result, &t)
	s.emitter.Emit(WSEventTrade, result)
}

func (s *BinanceSpotWebSocket) SubscribeTrades(market Market, callback func(trades []*Trade)) error {
	s.emitter.On(WSEventTrade, callback)
	_, _, err := binance.WsTradeServe(market.Symbol, s.tradeCallback, nil)

	if err != nil {
		fmt.Println(err)
		return err
	}
	//<-doneC
	return nil
}
