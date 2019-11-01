package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/lwsanty/cheap_flights/api_client"
	"github.com/lwsanty/cheap_flights/localization"
	tb "gopkg.in/tucnak/telebot.v2"
)

const (
	maxResults = 5

	defaultConfigPath = "config/localization.yml"
	separator         = ", "

	waitingGif = "https://media.giphy.com/media/tXL4FHPSnVJ0A/giphy.gif"
)

func main() {
	b, err := tb.NewBot(tb.Settings{
		Token:  os.Getenv("API_TOKEN"),
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Println("failed to acquire bot:", err)
		os.Exit(1)
	}

	data, err := ioutil.ReadFile(getEnv("CONFIG_PATH", defaultConfigPath))
	if err != nil {
		log.Println("failed to read languages file:", err)
		os.Exit(1)
	}

	regulars := make(map[string]localization.RegMsgs)
	if err := yaml.Unmarshal(data, regulars); err != nil {
		log.Println("failed to get languages:", err)
		os.Exit(1)
	}
	multiHelp := getMultiHelp(regulars)

	send := func(user *tb.User, text string) {
		if _, err := b.Send(user, text); err != nil {
			log.Println("failed to send message:", err)
		}
	}

	b.Handle("/start", func(m *tb.Message) {
		send(m.Sender, multiHelp)
	})

	b.Handle("/help", func(m *tb.Message) {
		send(m.Sender, multiHelp)
	})

	b.Handle(tb.OnUserJoined, func(m *tb.Message) {
		send(m.Sender, multiHelp)
	})

	b.Handle(tb.OnText, func(m *tb.Message) {
		loc, err := localization.New(regulars, m.Text)
		if err != nil {
			send(m.Sender, "🔴 could not parse text/не удалось распознать текст")
			log.Println("failed to prepareText:", err)
			return
		}

		text, err := prepareText(loc, m.Text)
		if err != nil {
			log.Println("failed to prepareText:", err)
			send(m.Sender, fmt.Sprintf("🔴 \n%s \n\n%s", loc.RegularMsgs.ParseError, loc.RegularMsgs.HelpInstructions))
			return
		}

		src, dst, err := api_client.GetSrcDstIATAs(text)
		if err != nil {
			log.Println("failed to get src and dst:", err)
			send(m.Sender, fmt.Sprintf("🔴🔴 \n%s \n\n%s", loc.RegularMsgs.AirportDataError, loc.RegularMsgs.HelpInstructions))
			return
		}

		//IATAResp := fmt.Sprintf("%s ➡️ %s", src.Name, dst.Name)
		//send(m.Sender, IATAResp)

		results, err := api_client.GetBestPrices(src, dst)
		if err != nil {
			log.Println("failed to get GetBestPrices:", err)
			send(m.Sender, "🔴🔴 \n"+loc.RegularMsgs.RequestError)
			return
		}

		optionsAmount := len(results)
		if optionsAmount == 0 {
			// TODO: pic
			send(m.Sender, loc.RegularMsgs.NothingFound)
			return
		}
		send(m.Sender, fmt.Sprintf(loc.RegularMsgs.Results, strconv.Itoa(optionsAmount), strconv.Itoa(maxResults)))

		// TODO: spinner
		waitMessage, err := b.Send(m.Sender, waitingGif)
		if err != nil {
			log.Println("failed to send wait message:", err)
		}

		optionsMessageText := optionsMessage(results)
		if waitMessage != nil {
			err := b.Delete(waitMessage)
			if err != nil {
				log.Println("failed to delete wait message:", err)
			}
		}

		send(m.Sender, optionsMessageText)
	})

	b.Start()
}

func optionsMessage(results []api_client.Result) string {
	var resOpt []string
	for i, res := range results {
		if i == maxResults {
			break
		}

		price := fmt.Sprintf(" ₽💶 %v", res.Option.Price)
		rate, err := api_client.GetCurrencyRateRubEur()
		if err == nil && rate != 0 {
			price = fmt.Sprintf("💶 € %.2f", rate*res.Option.Price)
		} else {
			log.Println("failed to get currency rate:", err)
		}

		depText := res.Option.DepartDate
		depDay, err := api_client.GetWeekdayFromDate(res.Option.DepartDate)
		if err == nil {
			depText = res.Option.DepartDate + " " + strings.ToLower(depDay.String())
		}

		retText := res.Option.ReturnDate
		retDay, err := api_client.GetWeekdayFromDate(res.Option.ReturnDate)
		if err == nil {
			retText = res.Option.ReturnDate + " " + strings.ToLower(retDay.String())
		}

		resOpt = append(resOpt, strings.Join([]string{
			price,
			fmt.Sprintf("🛫 %v", depText),
			fmt.Sprintf("🛬 %v", retText),
			// fmt.Sprintf("📏 %d km", opt.Distance),
			fmt.Sprintf("🔄 %d", res.Option.NumberOfChanges),
			//"🔎 " + res.Option.Site,
			"🔎 " + res.Link,
		}, "\n"))
	}

	return strings.Join(resOpt, "\n\n")
}

func prepareText(l *localization.Localization, text string) (string, error) {
	cities := strings.Split(text, separator)
	if len(cities) != 2 {
		return "", fmt.Errorf("amount of cities does not equal to 2: %v", cities)
	}

	var result []string
	for _, c := range cities {
		r, err := l.TranslateCity(c)
		if err != nil {
			return "", err
		}
		result = append(result, r)
	}

	return strings.Join(result, " "), nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func getMultiHelp(r map[string]localization.RegMsgs) string {
	var result []string
	for lang, text := range r {
		result = append(result, fmt.Sprintf("%s: %s %s", lang, text.Help, text.HelpInstructions))
	}
	return strings.Join(result, "\n\n")
}
