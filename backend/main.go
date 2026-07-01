package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"fiverr-go-scraper/database"
	pb "fiverr-go-scraper/pb"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gocolly/colly/v2"
	"github.com/robfig/cron/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// MICROSERVICE 1: gRPC SCRAPER ENGINE (PORT 50051)
type scraperServer struct {
	pb.UnimplementedScraperServiceServer
}

// 🚨 NEW FUNCTION: To identify exact block reason
func checkBlockReason(err error, statusCode int, body []byte) string {
	bodyStr := strings.ToLower(string(body))
	
	// Layer 2: TLS Fingerprint Check (Network level blocks)
	if err != nil {
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "tls") || strings.Contains(errStr, "handshake") || strings.Contains(errStr, "certificate") || strings.Contains(errStr, "connection reset") {
			return "Blocked at Layer 2: TLS/JA3 Fingerprint mismatch (Works in dev, dies at 10x)"
		}
	}
	
	// Layer 3: Captcha / Interstitial Check (Hidden blocks with status 200)
	if strings.Contains(bodyStr, "captcha") || strings.Contains(bodyStr, "cf-browser-verification") || strings.Contains(bodyStr, "robot check") || strings.Contains(bodyStr, "datadome") || strings.Contains(bodyStr, "px-captcha") {
		return "Blocked at Layer 3: Captcha or Interstitial Challenge Triggered"
	}
	
	// Layer 1: Header / WAF Check
	if statusCode == 401 || statusCode == 403 || statusCode == 429 || statusCode == 503 {
		return fmt.Sprintf("Blocked at Layer 1: Header/User-Agent Ban or IP Rate Limit (Status Code: %d)", statusCode)
	}
	
	if err != nil {
		return fmt.Sprintf("Network Error: %v", err)
	}
	
	return "Success"
}

func (s *scraperServer) RunScrape(ctx context.Context, req *pb.ScrapeRequest) (*pb.ScrapeResponse, error) {
	c := colly.NewCollector(colly.Async(true))

	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/114.0.0.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Safari/605.1.15",
	}

	c.OnRequest(func(r *colly.Request) {
		rand.Seed(time.Now().UnixNano())
		r.Headers.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
	})

	// 🚨 UPDATED INSTRUMENTATION: Error handler for TLS and Network blocks
	c.OnError(func(r *colly.Response, err error) {
		reason := checkBlockReason(err, r.StatusCode, r.Body)
		log.Printf("⚠️ [Scraper Blocked] URL: %s | Error Reason: %s\n", r.Request.URL, reason)
	})

	// 🚨 NEW INSTRUMENTATION: Response handler for hidden Captchas (Status might be 200 but page is blocked)
	c.OnResponse(func(r *colly.Response) {
		reason := checkBlockReason(nil, r.StatusCode, r.Body)
		if reason != "Success" {
			log.Printf("⚠️ [Hidden Block Detected] URL: %s | Block Reason: %s\n", r.Request.URL, reason)
		}
	})

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 5,
		RandomDelay: 1 * time.Second,
	})

	var products []*pb.Product
	var mutex sync.Mutex

	c.OnHTML(req.ContainerSelector, func(e *colly.HTMLElement) {
		item := &pb.Product{
			Title: e.ChildText(req.TitleSelector),
			Price: e.ChildText(req.PriceSelector),
		}
		if item.Title != "" {
			mutex.Lock()
			products = append(products, item)
			mutex.Unlock()
		}
	})

	if err := c.Visit(req.TargetUrl); err != nil {
		return nil, err
	}
	c.Wait()

	return &pb.ScrapeResponse{
		Status: "success",
		Total:  int32(len(products)),
		Data:   products,
	}, nil
}

// MICROSERVICE 2: API GATEWAY (GIN HTTP - PORT 8080)
type APIRequest struct {
	TargetURL         string `json:"targetUrl" binding:"required"`
	ContainerSelector string `json:"containerSelector" binding:"required"`
	TitleSelector     string `json:"titleSelector" binding:"required"`
	PriceSelector     string `json:"priceSelector" binding:"required"`
	Export            string `json:"export"`
}

func main() {
	// 0. Initialize Database securely
	database.ConnectDB()

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Type", "Accept"},
	}))

	// 1. Start gRPC Server
	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("gRPC failed to listen: %v", err)
		}
		grpcServer := grpc.NewServer()
		pb.RegisterScraperServiceServer(grpcServer, &scraperServer{})
		log.Println("⚡ [gRPC Engine] Running on port 50051")
		grpcServer.Serve(lis)
	}()

	time.Sleep(1 * time.Second)

	// 2. Connect to gRPC
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Gateway failed to connect to gRPC: %v", err)
	}
	defer conn.Close()
	grpcClient := pb.NewScraperServiceClient(conn)

	// 3. CRON SCHEDULER (DB Save + CSV File Creation)
	scheduler := cron.New()
	scheduler.AddFunc("@every 1m", func() {
		target := "https://quotes.toscrape.com/"
		autoReq := &pb.ScrapeRequest{
			TargetUrl:         target,
			ContainerSelector: ".quote",
			TitleSelector:     ".text",
			PriceSelector:     ".author",
		}
		response, err := grpcClient.RunScrape(context.Background(), autoReq)
		if err != nil {
			log.Println("❌ [CRON JOB] Scraping failed:", err)
			return
		}

		if database.DB != nil && response.Total > 0 {
			for _, item := range response.Data {
				_, err := database.DB.Exec("INSERT INTO products (target_url, title, price) VALUES (?, ?, ?)", target, item.Title, item.Price)
				if err != nil {
					log.Println("❌ DB Insert Error:", err)
				}
			}
			log.Printf("✅ [CRON JOB] Saved %d items to MySQL Database!\n", response.Total)
		}

		if response.Total > 0 {
			fileName := fmt.Sprintf("auto_report_%s.csv", time.Now().Format("2006-01-02_15-04-05"))
			file, err := os.Create(fileName)
			if err == nil {
				defer file.Close()
				writer := csv.NewWriter(file)
				writer.Write([]string{"Title", "Author Data"})
				for _, item := range response.Data {
					writer.Write([]string{item.Title, item.Price})
				}
				writer.Flush()
				log.Printf("📄 [CRON JOB] Success! File saved locally with %d items as %s\n", response.Total, fileName)
			}
		} else {
			log.Println("⚠️ [CRON JOB] 0 items scraped. Skipping CSV creation.")
		}
	})
	scheduler.Start()

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "🚀 Web Scraper API is Running Successfully!"})
	})

	router.POST("/api/scrape", func(c *gin.Context) {
		var req APIRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid fields"})
			return
		}

		grpcReq := &pb.ScrapeRequest{
			TargetUrl:         req.TargetURL,
			ContainerSelector: req.ContainerSelector,
			TitleSelector:     req.TitleSelector,
			PriceSelector:     req.PriceSelector,
		}

		response, err := grpcClient.RunScrape(c.Request.Context(), grpcReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Scraping engine failed"})
			return
		}

		// SAVE TO MYSQL DATABASE
		if database.DB != nil && response.Total > 0 {
			for _, item := range response.Data {
				_, err := database.DB.Exec("INSERT INTO products (target_url, title, price) VALUES (?, ?, ?)", req.TargetURL, item.Title, item.Price)
				if err != nil {
					log.Println("❌ DB Insert Error:", err)
				}
			}
			log.Println("💾 [Database] Scraped data successfully saved to MySQL.")
		}

		// Handle CSV Export Request
		if req.Export == "csv" {
			c.Writer.Header().Set("Content-Type", "text/csv")
			c.Writer.Header().Set("Content-Disposition", "attachment; filename=scraped_data.csv")
			writer := csv.NewWriter(c.Writer)
			writer.Write([]string{"Title", "Price"})
			for _, item := range response.Data {
				writer.Write([]string{item.Title, item.Price})
			}
			writer.Flush()
			return
		}

		// Map Data for React (Lowercase keys)
		var formattedData []map[string]string
		for _, item := range response.Data {
			formattedData = append(formattedData, map[string]string{
				"title": item.Title,
				"price": item.Price,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"status": response.Status,
			"total":  response.Total,
			"data":   formattedData,
		})
	})

	log.Println("🌐 [API Gateway] Ready! Send POST request to http://localhost:8080/api/scrape")
	router.Run(":8080")
}