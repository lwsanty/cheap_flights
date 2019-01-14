package main

import (
	"flights/api_client"
	"fmt"
	tb "gopkg.in/tucnak/telebot.v2"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	maxResults = 5

	helpText = "Вас приветствует пожилой бот для поиска дешевых билетов. " +
		"Чтобы начать поиск отправьте пожилое сообщение в виде \"Киев Таллин\" или \"Из Киева в Таллин\""
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

	send := func(user *tb.User, text string) {
		if _, err := b.Send(user, text); err != nil {
			log.Println("failed to send message:", err)
		}
	}

	b.Handle("/start", func(m *tb.Message) {
		send(m.Sender, helpText)
	})

	b.Handle("/help", func(m *tb.Message) {
		send(m.Sender, helpText)
	})

	b.Handle(tb.OnUserJoined, func(m *tb.Message) {
		send(m.Sender, helpText)
	})

	b.Handle(tb.OnText, func(m *tb.Message) {
		src, dst, err := api_client.GetSrcDstIATAs(m.Text)
		if err != nil {
			log.Println("failed to get src and dst:", err)
			send(m.Sender, "🔴 не смог получить данные об аэропортах")
			//send(m.Sender, "🔴 could not retrieve source and destination points")
			return
		}

		IATAResp := fmt.Sprintf("%s ➡️ %s", src.Name, dst.Name)
		send(m.Sender, IATAResp)

		options, err := api_client.GetBestPrices(src, dst)
		if err != nil {
			log.Println("failed to get GetBestPrices:", err)
			send(m.Sender, "🔴 произошла ошибка при отправке запроса")
			return
		}

		optionsAmount := len(options)
		if optionsAmount == 0 {
			send(m.Sender, "Ничего не нашел")
			return
		}
		send(m.Sender, "Всего результатов: "+strconv.Itoa(optionsAmount)+", покажу до "+strconv.Itoa(maxResults)+" лучших:")
		send(m.Sender, optionsMessage(options))
	})

	b.Start()
}

func optionsMessage(options []api_client.PriceOption) string {
	var resOpt []string
	for i, opt := range options {
		if i == maxResults {
			break
		}

		price := fmt.Sprintf("💶 %v ₽", opt.Price)
		rate, err := api_client.GetCurrencyRateRubEur()
		if err == nil && rate != 0 {
			price = fmt.Sprintf("💶 %.2f €", rate*opt.Price)
		} else {
			log.Println("failed to get currency rate:", err)
		}

		depText := opt.DepartDate
		depDay, err := api_client.GetWeekdayFromDate(opt.DepartDate)
		if err == nil {
			depText = opt.DepartDate + " " + strings.ToLower(depDay.String())
		}

		retText := opt.ReturnDate
		retDay, err := api_client.GetWeekdayFromDate(opt.ReturnDate)
		if err == nil {
			retText = opt.ReturnDate + " " + strings.ToLower(retDay.String())
		}

		resOpt = append(resOpt, strings.Join([]string{
			price,
			fmt.Sprintf("🛫 %v", depText),
			fmt.Sprintf("🛬 %v", retText),
			// fmt.Sprintf("📏 %d км", opt.Distance),
			fmt.Sprintf("🔄 %d", opt.NumberOfChanges),
			"🔎 " + opt.Site,
		}, "\n"))
	}

	return strings.Join(resOpt, "\n\n")
}
