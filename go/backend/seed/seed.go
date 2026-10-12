package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"bigtable-backend/db"
)

const (
	totalRows = 1_000_000
	batchSize = 5000
)

var customers = []struct {
	name string
	code string
}{
	{"田中製作所", "CST-001"}, {"山田工業", "CST-002"}, {"鈴木電機", "CST-003"},
	{"佐藤商事", "CST-004"}, {"高橋物産", "CST-005"}, {"伊藤重工", "CST-006"},
	{"渡辺精密", "CST-007"}, {"中村機械", "CST-008"}, {"小林金属", "CST-009"},
	{"加藤化学", "CST-010"}, {"吉田建設", "CST-011"}, {"山口食品", "CST-012"},
	{"松本繊維", "CST-013"}, {"井上電子", "CST-014"}, {"木村自動車", "CST-015"},
}

var products = []struct {
	name string
	code string
}{
	{"六角ボルト M10", "PRD-001"}, {"ステンレス配管", "PRD-002"}, {"精密ベアリング", "PRD-003"},
	{"油圧シリンダー", "PRD-004"}, {"制御基板 TypeA", "PRD-005"}, {"アルミフレーム", "PRD-006"},
	{"耐熱ガスケット", "PRD-007"}, {"産業用モーター", "PRD-008"}, {"ゴムパッキン", "PRD-009"},
	{"ステンレス歯車", "PRD-010"}, {"銅製端子台", "PRD-011"}, {"防振ゴムマウント", "PRD-012"},
	{"チタン合金板", "PRD-013"}, {"セラミックノズル", "PRD-014"}, {"カーボンブラシ", "PRD-015"},
	{"高圧ホース", "PRD-016"}, {"精密スプリング", "PRD-017"}, {"電磁弁ユニット", "PRD-018"},
	{"断熱材シート", "PRD-019"}, {"光学レンズ", "PRD-020"},
}

var statuses = []string{"受注確認", "出荷準備中", "出荷済み", "納品完了", "キャンセル"}
var orderTypes = []string{"受注", "発注"}

func main() {
	conn, err := db.Connect()
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	createTable(conn)
	seedData(conn)
}

func createTable(db *sql.DB) {
	query := `
CREATE TABLE IF NOT EXISTS orders (
	id BIGINT AUTO_INCREMENT PRIMARY KEY,
	order_number VARCHAR(20) NOT NULL,
	order_type VARCHAR(10) NOT NULL,
	order_date DATE NOT NULL,
	customer_name VARCHAR(100) NOT NULL,
	customer_code VARCHAR(20) NOT NULL,
	product_name VARCHAR(100) NOT NULL,
	product_code VARCHAR(20) NOT NULL,
	quantity INT NOT NULL,
	unit_price DECIMAL(12,2) NOT NULL,
	total_amount DECIMAL(14,2) NOT NULL,
	status VARCHAR(20) NOT NULL,
	delivery_date DATE NOT NULL,
	notes TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	INDEX idx_order_type (order_type),
	INDEX idx_status (status),
	INDEX idx_customer_name (customer_name),
	INDEX idx_product_name (product_name),
	INDEX idx_order_date (order_date),
	INDEX idx_order_number (order_number)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`

	if _, err := db.Exec(query); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	log.Println("Table 'orders' ready")
}

// planSeed decides what to do with rows already in orders. TRUNCATE cannot be
// rolled back, so a partial table is cleared only when SEED_RESET=1 asks for it.
func planSeed(count int64, reset bool) (skip, truncate bool, err error) {
	switch {
	case count >= totalRows:
		return true, false, nil
	case count == 0:
		return false, false, nil
	case reset:
		return false, true, nil
	default:
		return false, false, fmt.Errorf("orders already has %d rows (expected 0 or %d); set SEED_RESET=1 to delete them and reseed", count, totalRows)
	}
}

func seedData(db *sql.DB) {
	// Check if data already exists
	var count int64
	if err := db.QueryRow("SELECT COUNT(*) FROM orders").Scan(&count); err != nil {
		log.Fatalf("Failed to count existing rows: %v", err)
	}
	skip, truncate, err := planSeed(count, os.Getenv("SEED_RESET") == "1")
	if err != nil {
		log.Fatal(err)
	}
	if skip {
		log.Printf("Table already has %d rows, skipping seed", count)
		return
	}
	if truncate {
		if _, err := db.Exec("TRUNCATE TABLE orders"); err != nil {
			log.Fatalf("Failed to truncate orders: %v", err)
		}
		log.Printf("Truncated %d existing rows (SEED_RESET=1)", count)
	}

	rng := rand.New(rand.NewSource(42))
	startDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	dateRange := int(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC).Sub(startDate).Hours() / 24)

	notes := []string{
		"通常納品", "急ぎ対応", "分割納品可", "検品済み", "要確認",
		"特急対応", "定期発注", "サンプル品", "試作品", "量産品",
	}

	log.Printf("Seeding %d rows in batches of %d...", totalRows, batchSize)
	start := time.Now()

	for batch := 0; batch < totalRows/batchSize; batch++ {
		insertBatch(db, rng, batch, startDate, dateRange, notes)
		if (batch+1)%20 == 0 {
			log.Printf("  Inserted %d / %d rows (%.1fs)", (batch+1)*batchSize, totalRows, time.Since(start).Seconds())
		}
	}

	log.Printf("Seeding complete: %d rows in %.1fs", totalRows, time.Since(start).Seconds())
}

func insertBatch(db *sql.DB, rng *rand.Rand, batch int, startDate time.Time, dateRange int, notes []string) {
	cols := "(order_number, order_type, order_date, customer_name, customer_code, product_name, product_code, quantity, unit_price, total_amount, status, delivery_date, notes)"
	placeholders := make([]string, 0, batchSize)
	args := make([]any, 0, batchSize*13)

	for i := 0; i < batchSize; i++ {
		rowNum := batch*batchSize + i + 1
		orderNum := fmt.Sprintf("ORD-%07d", rowNum)
		orderType := orderTypes[rng.Intn(len(orderTypes))]
		orderDate := startDate.AddDate(0, 0, rng.Intn(dateRange))
		cust := customers[rng.Intn(len(customers))]
		prod := products[rng.Intn(len(products))]
		qty := rng.Intn(100) + 1
		unitPrice := float64(rng.Intn(50000)+100) + float64(rng.Intn(100))/100.0
		totalAmt := float64(qty) * unitPrice
		status := statuses[rng.Intn(len(statuses))]
		deliveryDate := orderDate.AddDate(0, 0, rng.Intn(30)+7)
		note := notes[rng.Intn(len(notes))]

		placeholders = append(placeholders, "(?,?,?,?,?,?,?,?,?,?,?,?,?)")
		args = append(args, orderNum, orderType, orderDate.Format("2006-01-02"),
			cust.name, cust.code, prod.name, prod.code,
			qty, unitPrice, totalAmt, status,
			deliveryDate.Format("2006-01-02"), note)
	}

	query := fmt.Sprintf("INSERT INTO orders %s VALUES %s", cols, strings.Join(placeholders, ","))
	if _, err := db.Exec(query, args...); err != nil {
		log.Fatalf("Batch insert failed at batch %d: %v", batch, err)
	}
}
