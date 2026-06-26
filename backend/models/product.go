package models

// Product struct JSON response ke liye data model define karta hai
type Product struct {
	Title string `json:"title"`
	Price string `json:"price"`
}
