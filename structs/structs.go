package structs

type StockPriceDetails struct {
	StockName string
	APIKey    string
}

type StockValues struct {
	Token map[string]interface{} `json:"Time Series (Daily)"`
}

type StockValuesFound struct {
	Open map[string]interface{} `json:"open"`
}

type MyEvent struct {
	Name string `json:"name"`
}

type MyResponse struct {
	Message string `json:"Answer:"`
	Ok      bool   `json:"ok"`
}
