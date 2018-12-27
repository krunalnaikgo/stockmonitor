package stocksearch

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"git/repos/stock/utils"
	"strconv"
)

type StockPriceDetails struct {
	StockName string
	APIKey    string
}

type StockValues struct {
	Token map[string]interface{} `json:"Time Series (1min)"`
}

type StockValuesFound struct {
	Open map[string]interface{} `json:"open"`
}

func (stock *StockPriceDetails) getStockData() map[string]interface{} {
	stockUrl := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_INTRADAY&"+
		"symbol=%s&interval=1min&outputsize=compact&apikey=%s",
		stock.StockName,
		stock.APIKey)

	client := &http.Client{}
	req, err := http.NewRequest("GET", stockUrl, nil)
	if err != nil {
		log.Fatal("Found Error", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if resp.StatusCode == 200 || resp.StatusCode == 429 {
		getStockData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal("Found Error", err)
		}
		var stockToken StockValues
		err1 := json.Unmarshal(getStockData, &stockToken)
		if err1 != nil {
			log.Fatal("Unmarshal Error", err1)
		}
		log.Println("Stock Price Found")
		return stockToken.Token
	} else {
		log.Println("Stock Price not Found", resp)
	}

	return nil
}

func (stock *StockPriceDetails) queryStock(timeT string) []map[string]interface{} {
	stockData := stock.getStockData()
	var stockList []map[string]interface{}
	for _, value := range stockData {
		stockList = append(stockList, value.(map[string]interface{}))
	}
	return stockList
}

func (stock *StockPriceDetails) getSumStockValues() (float64, float64, int, []float64) {
	currTime := utils.GetCurrentTime()
	stringMap := stock.queryStock(currTime)
	var foundHighVal string
	var foundLowVal string
	count := 0
	var floatHighValSum float64
	var floatLowValSum float64
	var maxHigh []float64
	if stringMap != nil {
		for _, mapValue := range stringMap {
			for key, value := range mapValue {
				foundHigh := strings.Contains(key, "high")
				foundlow := strings.Contains(key, "low")
				if foundHigh {
					foundHighVal = fmt.Sprintf("%v", value)
				}
				if foundlow {
					foundLowVal = fmt.Sprintf("%v", value)
				}
				count = count + 1
				foundFloatHighVal, _ := strconv.ParseFloat(strings.TrimSpace(foundHighVal), 64)
				maxHigh = append(maxHigh, foundFloatHighVal)
				floatHighValSum = floatHighValSum + foundFloatHighVal
				foundFloatLowVal, _ := strconv.ParseFloat(strings.TrimSpace(foundLowVal), 64)
				floatLowValSum = floatLowValSum + foundFloatLowVal

			}
		}
	}
	return floatHighValSum, floatLowValSum, count, maxHigh
}

func (stock *StockPriceDetails) GetStockValues() (float64, float64, float64) {
	foundHighVal, foundLowVal, foundCount, listHighVal := stock.getSumStockValues()
	avgHighVal := foundHighVal / float64(foundCount)
	avgLowVal := foundLowVal / float64(foundCount)
	var maxHighValue float64
	var firstValue bool
	for _, v1 := range listHighVal {
		if !firstValue {
			maxHighValue = v1
			firstValue = true
		} else {
			if v1 > maxHighValue {
				maxHighValue = v1
			}
		}
	}
	return avgHighVal, avgLowVal, maxHighValue
}
