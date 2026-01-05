package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

var (
	db  *sql.DB
	rdb *redis.Client
	ctx = context.Background()
)

type OrderRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type Order struct {
	ID        int       `json:"id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func initDB() {
	var err error
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	// Retry loop for DB connection
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			log.Println("Connected to Database")
			break
		}
		log.Printf("Failed to connect to DB, retrying... (%v)", err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY,
		product_id VARCHAR(50),
		quantity INT,
		status VARCHAR(20),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
}

func initRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
	})

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	log.Println("Connected to Redis")
}

// SanitizeInput removes potentially dangerous characters
func SanitizeInput(input string) string {
	safe := strings.ReplaceAll(input, "<", "")
	safe = strings.ReplaceAll(safe, ">", "")
	safe = strings.ReplaceAll(safe, "'", "")
	safe = strings.ReplaceAll(safe, ";", "")
	return safe
}

func buyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	safeProductID := SanitizeInput(req.ProductID)

	// insert into DB
	var orderID int
	err := db.QueryRow("INSERT INTO orders (product_id, quantity, status) VALUES ($1, $2, 'pending') RETURNING id",
		safeProductID, req.Quantity).Scan(&orderID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("DB Insert Error: %v", err)
		return
	}

	// Publish to Redis
	orderMsg := map[string]interface{}{
		"order_id":   orderID,
		"product_id": safeProductID,
		"quantity":   req.Quantity,
	}
	msgBytes, _ := json.Marshal(orderMsg)
	err = rdb.Publish(ctx, "orders", msgBytes).Err()
	if err != nil {
		log.Printf("Redis Publish Error: %v", err)
		// Don't fail the request if redis fails, just log it (or handle retry)
	}

	log.Printf("Processed order #%d for ProductID: %s", orderID, safeProductID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "Order processed",
		"product_id": safeProductID,
		"order_id":   orderID,
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	initDB()
	initRedis()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/buy", buyHandler)

	log.Printf("Store API (Secure) starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
