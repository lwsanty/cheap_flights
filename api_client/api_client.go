package api_client

import (
	"encoding/json"
	"fmt"
	"github.com/vjeantet/jodaTime"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"time"
)

const (
	urlIATAPrefix = "https://www.travelpayouts.com/widgets_suggest_params?q="
	urlBestPrices = "http://min-prices.aviasales.ru/calendar_preload"

	urlCurrencyRateRubEur = "http://free.currencyconverterapi.com/api/v5/convert?q=RUB_EUR&compact=y"

	timeFormat = "YYYY-MM-dd"
)

type IATAPoint struct {
	IATA string `json:"iata"`
	Name string `json:"name"`
}

type IATAResponse struct {
	Src IATAPoint `json:"origin"`
	Dst IATAPoint `json:"destination"`
}

// TODO errors
type BestPricesResponse struct {
	Options []PriceOption `json:"best_prices"`
}

// TODO rubles to some sweet currency
type PriceOption struct {
	Price           float32 `json:"value"` // rubles
	ReturnDate      string  `json:"return_date"`
	NumberOfChanges int     `json:"number_of_changes"`
	Site            string  `json:"gate"`
	Distance        int     `json:"distance"`
	DepartDate      string  `json:"depart_date"`
}

type CurrencyResponse struct {
	CR CurrencyRate `json:"RUB_EUR"`
}

type CurrencyRate struct {
	Value float32 `json:"val"`
}

// TODO English support
func GetSrcDstIATAs(text string) (*IATAPoint, *IATAPoint, error) {
	encodedText := template.URLQueryEscaper(text)

	data, err := doReq(urlIATAPrefix + encodedText)
	if err != nil {
		return nil, nil, err
	}

	var resp IATAResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil, err
	}

	src, dst := &resp.Src, &resp.Dst
	if resp.Src.IATA == "" || resp.Dst.IATA == "" {
		return nil, nil, fmt.Errorf("IATAs were not parsed right, src: %s, dst: %s", src.IATA, dst.IATA)
	}
	return src, dst, nil
}

func GetBestPrices(src, dst *IATAPoint) ([]PriceOption, error) {
	// ?origin=BCN&destination=MOW&depart_date=2014-12-01&one_way=false

	var bpUrl *url.URL
	bpUrl, err := url.Parse(urlBestPrices)
	if err != nil {
		return nil, err
	}

	parameters := url.Values{}
	parameters.Add("origin", src.IATA)
	parameters.Add("destination", dst.IATA)
	parameters.Add("depart_date", jodaTime.Format(timeFormat, time.Now()))
	parameters.Add("one_way", "false")
	bpUrl.RawQuery = parameters.Encode()

	data, err := doReq(bpUrl.String())
	if err != nil {
		return nil, err
	}

	var resp BestPricesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	options := resp.Options
	sort.Slice(options, func(i, j int) bool { return options[i].Price < options[j].Price })
	return options, nil
}

func GetCurrencyRateRubEur() (float32, error) {
	data, err := doReq(urlCurrencyRateRubEur)
	if err != nil {
		return 0, err
	}

	var resp CurrencyResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return 0, err
	}

	return resp.CR.Value, nil
}

func doReq(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(res.Body)
	return data, err
}

func GetWeekdayFromDate(timeValue string) (time.Weekday, error) {
	t, err := jodaTime.Parse(timeFormat, timeValue)
	if err != nil {
		return 0, err
	}

	return t.Weekday(), nil
}
