// package handlers

// import (
// 	"fiverr-go-scraper/scraper"
// 	"net/http"

// 	"github.com/gin-gonic/gin"
// )

// // Ab humari API 4 cheezein mangegi
// type ScrapeRequest struct {
// 	TargetURL         string `json:"targetUrl" binding:"required"`
// 	ContainerSelector string `json:"containerSelector" binding:"required"` // Main dabba (jaise card)
// 	TitleSelector     string `json:"titleSelector" binding:"required"`     // Title kahan hai
// 	PriceSelector     string `json:"priceSelector" binding:"required"`     // Price kahan hai
// }

// func HandleScrape(c *gin.Context) {
// 	var req ScrapeRequest

// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide targetUrl, containerSelector, titleSelector, and priceSelector."})
// 		return
// 	}

// 	// Ab hum scraper ko saari details bhej rahe hain
// 	results, err := scraper.RunAdvancedScraper(req.TargetURL, req.ContainerSelector, req.TitleSelector, req.PriceSelector)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Scraping failed."})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status": "success",
// 		"total":  len(results),
// 		"data":   results,
// 	})
// }


package handlers

import (
	"encoding/csv" // CSV file banane ke liye naya package
	"net/http"
	"fiverr-go-scraper/scraper"

	"github.com/gin-gonic/gin"
)

// Request body mein humne 'export' field add kar diya hai
type ScrapeRequest struct {
	TargetURL         string `json:"targetUrl" binding:"required"`
	ContainerSelector string `json:"containerSelector" binding:"required"`
	TitleSelector     string `json:"titleSelector" binding:"required"`
	PriceSelector     string `json:"priceSelector" binding:"required"`
	Export            string `json:"export"` // Agar user "csv" likhega toh file download hogi
}

func HandleScrape(c *gin.Context) {
	var req ScrapeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide targetUrl, containerSelector, titleSelector, and priceSelector."})
		return
	}

	// Scraper ko call karna
	results, err := scraper.RunAdvancedScraper(req.TargetURL, req.ContainerSelector, req.TitleSelector, req.PriceSelector)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Scraping failed."})
		return
	}

	
	// NEW FEATURE: CSV DOWNLOAD LOGIC
	
	if req.Export == "csv" {
		// Browser/Postman ko batana ki ye ek downloadable file hai
		c.Writer.Header().Set("Content-Type", "text/csv")
		c.Writer.Header().Set("Content-Disposition", "attachment; filename=scraped_data.csv")

		writer := csv.NewWriter(c.Writer)

		// 1. Excel file ki pehli line (Header) likhna
		writer.Write([]string{"Title", "Price"})

		// 2. Loop chala kar saara data Excel rows mein bharna
		for _, item := range results {
			writer.Write([]string{item.Title, item.Price})
		}

		// Data save karna aur request end karna
		writer.Flush()
		return
	}


	// DEFAULT LOGIC (Normal JSON Response)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"total":  len(results),
		"data":   results,
	})
}