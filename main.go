package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/PuerkitoBio/goquery"
	"gopkg.in/urfave/cli.v2"
)

type flags struct {
	URL         string
	World       string
	BotAPIToken string
	RecipientID Recipient
}

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "url",
				Value: "https://na.finalfantasyxiv.com/lodestone/news/category/1",
			},
			&cli.StringFlag{
				Name:  "world",
				Value: "tonberry",
			},
			&cli.StringFlag{
				Name:  "bot-api-token",
				Value: "TOKEN",
			},
			&cli.StringFlag{
				Name:  "recipient-id",
				Value: "USERID",
			},
		},
		Action: checkList,
	}

	app.Run(os.Args)

	t := time.NewTicker(10 * time.Minute)
	for {
		select {
		case <-t.C:
			err := app.Run(os.Args)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
}

func checkList(c *cli.Context) error {
	flags := &flags{
		URL:         c.String("url"),
		World:       c.String("world"),
		BotAPIToken: c.String("bot-api-token"),
		RecipientID: Recipient(c.String("recipient-id")),
	}
	doc, err := goquery.NewDocument(flags.URL)
	if err != nil {
		return err
	}
	sel := doc.Find(".news__content > ul")

	parsedURL, err := url.Parse(flags.URL)
	if err != nil {
		return fmt.Errorf("could not parse URL: %s", err)
	}

	newsArticleURL := ""
	sel.Children().Each(func(i int, sel *goquery.Selection) {
		if strings.Contains(strings.ToLower(sel.Text()), "congested") {
			if newsArticleURL == "" {
				link, exists := sel.Find("a").Attr("href")
				if !exists {
					log.Println("no href found")
				}
				newsArticleURL = link
			}
		}
	})

	if newsArticleURL == "" {
		return errors.New("news article URL not found")
	}

	articlePath := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, newsArticleURL)
	world := strings.Title(flags.World)

	isCongested, err := congested(articlePath, world)
	if err != nil {
		return fmt.Errorf("could not check article link: %s", err)
	}

	result := ""
	if isCongested {
		log.Printf("%s is congested\n", world)
		return nil
	}

	result = fmt.Sprintf("%s is no longer congested\n", world)

	bot, err := tb.NewBot(tb.Settings{
		Token:  flags.BotAPIToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return fmt.Errorf("could not create bot: %s", err)
	}

	_, err = bot.Send(flags.RecipientID, result)
	if err != nil {
		return fmt.Errorf("could not send message: %s", err)
	}

	return nil

}

// Recipient is a wrapper string class for users
type Recipient string

// Recipient returns the recipient
func (u Recipient) Recipient() string {
	return string(u)
}

func congested(url string, world string) (bool, error) {
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return false, err
	}

	sel := doc.Find(".news__detail__wrapper")

	if strings.Contains(sel.Text(), "× Tonberry") {
		return true, nil
	}

	return false, nil
}
