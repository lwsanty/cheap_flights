package localization

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/abadojack/whatlanggo"
)

const langDef = "en"

var whiteListOptions = whatlanggo.Options{
	Whitelist: map[whatlanggo.Lang]bool{
		whatlanggo.Eng: true,
		whatlanggo.Rus: true,
		whatlanggo.Spa: true,
		whatlanggo.Deu: true,
	},
}

type Localization struct {
	lang string
}

func New(s string) *Localization {
	info := whatlanggo.DetectWithOptions(s, whiteListOptions)
	lang := whatlanggo.LangToStringShort(info.Lang)
	if lang == "" {
		lang = langDef
	}
	return &Localization{
		lang: lang,
	}
}

func (l *Localization) Text(s string) string {
	text, err := Translate(s, "auto", l.lang)
	if err != nil {
		fmt.Printf("failed to translate %s to %s: %v\n", s, l.lang, err)
		return s
	}
	return text
}

// fixed github.com/bas24/googletranslatefree
func Translate(source, sourceLang, targetLang string) (string, error) {
	var text []string
	var result []interface{}

	url := "https://translate.googleapis.com/translate_a/single?client=gtx&sl=" +
		sourceLang + "&tl=" + targetLang + "&dt=t&q=" + url.QueryEscape(source) +
		"&ie=UTF-8&oe=UTF-8"

	r, err := http.Get(url)
	if err != nil {
		return "err", errors.New("error getting translate.googleapis.com")
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "err", errors.New("error reading response body")
	}

	bReq := strings.Contains(string(body), `<title>Error 400 (Bad Request)`)
	if bReq {
		return "err", errors.New("error 400 (Bad Request)")
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return "err", fmt.Errorf("error unmarshaling data: %v\n", string(body))
	}

	if len(result) > 0 {
		inner := result[0]
		for _, slice := range inner.([]interface{}) {
			for _, translatedText := range slice.([]interface{}) {
				text = append(text, fmt.Sprintf("%v", translatedText))
				break
			}
		}
		cText := strings.Join(text, "")

		return cText, nil
	} else {
		return "err", errors.New("no translated data in response")
	}
}
