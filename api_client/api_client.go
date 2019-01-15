package api_client

import (
	"encoding/json"
	"fmt"
	"github.com/vjeantet/jodaTime"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	urlIATAPrefix = "https://www.travelpayouts.com/widgets_suggest_params?q="
	urlBestPrices = "http://min-prices.aviasales.ru/calendar_preload"

	urlCurrencyRateRubEur = "http://free.currencyconverterapi.com/api/v5/convert?q=RUB_EUR&compact=y"

	timeFormat = "YYYY-MM-dd"

	defaultLink = "aviasales.ru"
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

type Result struct {
	Option PriceOption
	Link   string
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

// ?origin=BCN&destination=MOW&depart_date=2014-12-01&one_way=false
func GetBestPrices(src, dst *IATAPoint) ([]Result, error) {
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

	var results []Result
	for _, opt := range options {
		link, err := GetOptionLink(src, dst, opt)
		if err != nil {
			log.Println("failed to get link:", err)
			link = defaultLink
		}
		results = append(results, Result{
			Option: opt,
			Link:   link,
		})
	}
	return results, nil
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

// GetOptionLink returns link to search results of given cities on given dates
// date format in api 2020-01-09
// returned link example www.aviasales.ru/search/IEV2301TLL2401123 where
// IEV, TLL - src and dst IATAs
// 2301, 2401 - departure and return dates in ddmm format
// 123 - amounts of adult, child and baby tickets (1 adult ticket, 2 child tickets, 3 baby tickets)
func GetOptionLink(src, dst *IATAPoint, option PriceOption) (string, error) {
	dayMonth := func(date string) (string, error) {
		dt := strings.Split(date, "-")
		if len(dt) != 3 {
			return "", fmt.Errorf("wrong date format: %s", date)
		}
		return dt[2] + dt[1], nil
	}

	dep, err := dayMonth(option.DepartDate)
	if err != nil {
		return "", err
	}
	ret, err := dayMonth(option.ReturnDate)
	if err != nil {
		return "", err
	}

	return "aviasales.ru/search/" + src.IATA + dep + dst.IATA + ret + "1", nil
}
