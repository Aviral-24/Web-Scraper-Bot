package scraper

import (
	"fiverr-go-scraper/models"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
)

// Function ab URL ke sath selectors bhi accept karta hai
func RunAdvancedScraper(targetURL, containerSel, titleSel, priceSel string) ([]models.Product, error) {
	c := colly.NewCollector(
		colly.Async(true),
	)

	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/114.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Version/14.0.3 Safari/605.1.15",
	}

	c.OnRequest(func(r *colly.Request) {
		rand.Seed(time.Now().UnixNano())
		r.Headers.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
		fmt.Println("Scraping Page:", r.URL)
	})

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 5,
		RandomDelay: 1 * time.Second,
	})

	var products []models.Product
	var mutex sync.Mutex

	// DYNAMIC EXTRACTION LOGIC (Koi hardcoding nahi!)
	c.OnHTML(containerSel, func(e *colly.HTMLElement) {
		item := models.Product{
			Title: e.ChildText(titleSel),
			Price: e.ChildText(priceSel),
		}

		if item.Title != "" {
			mutex.Lock()
			products = append(products, item)
			mutex.Unlock()
		}
	})

	err := c.Visit(targetURL)
	if err != nil {
		return nil, err
	}

	c.Wait()

	return products, nil
}
