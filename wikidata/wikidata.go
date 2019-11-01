package wikidata

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// v0.0.1
// TODO: customize source language
const translateQuery = `
SELECT distinct ?itemLabel
WHERE{  
  ?item ?label "%s"@%s.
  ?article schema:about ?item .
  ?article schema:inLanguage "ru" .
  ?article schema:isPartOf <https://ru.wikipedia.org/>. 
  SERVICE wikibase:label { bd:serviceParam wikibase:language "ru". }
}
LIMIT 1
`

var NotFound = fmt.Errorf("not found")

func TranslateCity(srcLang, city string) (string, error) {
	query := url.QueryEscape(translateCityQuery(srcLang, city))
	url := "https://query.wikidata.org/sparql?query=" + query

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Accept", "text/csv")

	resp, err := new(http.Client).Do(req)
	if err != nil {
		return "", err
	}
	log.Println("code:", resp.StatusCode)
	defer func() {
		_ = resp.Body.Close()
	}()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	result := processTranslationOutput(string(data))
	if result == "" {
		return "", NotFound
	}
	return result, nil
}

// TODO: Rostov-on-Don
func translateCityQuery(srcLang, city string) string {
	return fmt.Sprintf(translateQuery, strings.Title(city), strings.Title(srcLang))
}

func processTranslationOutput(s string) string {
	return strings.Replace(
		strings.Replace(s, "itemLabel", "", -1),
		"\r\n",
		"",
		-1)
}
