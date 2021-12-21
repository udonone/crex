package binancefutures

import (
	"encoding/json"
	"log"
	"testing"
	"time"

	. "github.com/coinrust/crex"
	"github.com/coinrust/crex/configtest"
)

func testExchange() Exchange {
	testConfig := configtest.LoadTestConfig("binancefutures")
	params := &Parameters{
		AccessKey: testConfig.AccessKey,
		SecretKey: testConfig.SecretKey,
		Testnet:   testConfig.Testnet,
		ProxyURL:  testConfig.ProxyURL,
	}
	ex := NewBinanceFutures(params)
	return ex
}

func TestBinanceFutures_GetTime(t *testing.T) {
	ex := testExchange()
	tm, err := ex.GetTime()
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("%v", tm)
}

func TestBinanceFutures_GetBalance(t *testing.T) {
	ex := testExchange()
	balance, err := ex.GetBalance("USDT")
	if err != nil {
		t.Error(err)
		return
	}
	//https://fapi.binance.com/fapi/v2/balance
	t.Logf("%#v", balance)
}

func TestBinanceFutures_GetOrderBook(t *testing.T) {
	ex := testExchange()
	ob, err := ex.GetOrderBook("BNBUSDT", 10)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("%#v", ob)
}

func TestBinanceFutures_GetRecords(t *testing.T) {
	ex := testExchange()
	now := time.Now()
	start := now.Add(-10 * time.Minute)
	end := now
	records, err := ex.GetRecords("BNBUSDT",
		PERIOD_1MIN, start.Unix(), end.Unix(), 10)
	if err != nil {
		t.Error(err)
		return
	}
	for _, v := range records {
		t.Logf("Timestamp: %v %#v", v.Timestamp, v)
	}
}

func TestBinanceFutures_GetOpenOrders(t *testing.T) {
	ex := testExchange()
	orders, err := ex.GetOpenOrders("BNBUSDT")
	if err != nil {
		t.Error(err)
		return
	}
	b, _ := json.Marshal(orders)
	t.Logf("%#v", string(b))
}

func TestBinanceFutures_ChangeLeverage(t *testing.T) {
	ex := testExchange()
	binance := ex.(*BinanceFutures)
	err := binance.ChangeLeverage("BTCUSDT", 3)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestWsTrade(t *testing.T)  {
	ex := testExchange()
	binance := ex.(*BinanceFutures)
	binance.ws.SubscribeTrades(Market{
		Symbol: "BTCUSDT",
	}, func(ob []*Trade) {
		log.Printf("%s-%f-%f",ob[0].Symbol, ob[0].Price, ob[0].Amount)
	})

	select {}
}
