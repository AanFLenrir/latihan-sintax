package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"latihan-fiber/app/model"
	"latihan-fiber/config"
	"latihan-fiber/database"
	"latihan-fiber/helper"
)

const (
	baseURL = "http://localhost:3000/api/v1"
)

func main() {
	testName := flag.String("test", "c1", "c1 | c2_part1 | c2_part2 | c3 | c4 | c5_step1 | c5_step2 | reset")
	flag.Parse()

	config.LoadEnv()
	ctx := context.Background()

	pool, err := database.NewPool(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gagal connect db: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	jwtSecret := config.GetEnv("JWT_SECRET", "")
	jwtManager := helper.NewJWTManager(jwtSecret, config.GetEnv("JWT_ISSUER", "praktikum-backend"), 15*time.Minute)

	// Users
	var mhsuji, salahalamat, admin model.User
	pool.QueryRow(ctx, "SELECT id, username, email, role FROM users WHERE username = 'mhsuji'").
		Scan(&mhsuji.ID, &mhsuji.Username, &mhsuji.Email, &mhsuji.Role)
	pool.QueryRow(ctx, "SELECT id, username, email, role FROM users WHERE username = 'salahalamat'").
		Scan(&salahalamat.ID, &salahalamat.Username, &salahalamat.Email, &salahalamat.Role)
	pool.QueryRow(ctx, "SELECT id, username, email, role FROM users WHERE username = 'admin'").
		Scan(&admin.ID, &admin.Username, &admin.Email, &admin.Role)

	tokenMhsuji, _ := jwtManager.GenerateAccess(mhsuji)
	tokenSalah, _ := jwtManager.GenerateAccess(salahalamat)
	tokenAdmin, _ := jwtManager.GenerateAccess(admin)

	client := &http.Client{Timeout: 10 * time.Second}

	switch *testName {
	case "reset":
		runReset(ctx, pool)
		fmt.Println("Reset berhasil!")

	case "c1":
		runTestC1(ctx, pool, client, tokenMhsuji, mhsuji)

	case "c2_part1":
		runTestC2Part1(ctx, client, tokenMhsuji, tokenAdmin)

	case "c2_part2":
		runTestC2Part2(ctx, client, tokenMhsuji, tokenAdmin)

	case "c3":
		runTestC3(ctx, pool, client, tokenMhsuji, mhsuji)

	case "c4":
		runTestC4(ctx, pool, client, tokenSalah, tokenAdmin)

	case "c5_step1":
		runTestC5Step1(ctx, pool, client, tokenMhsuji)

	case "c5_step2":
		runTestC5Step2(ctx, pool, tokenMhsuji)

	case "c6_enroll":
		code, resp, _ := doReq(client, "POST", baseURL+"/courses/9/enroll", tokenMhsuji, nil)
		fmt.Printf("POST /courses/9/enroll -> Status: %d, Response: %s\n", code, resp)

	case "c6_verify":
		var count int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM notifications n JOIN outbox o ON o.id = n.outbox_id WHERE (o.payload->>'course_id')::int = 9").Scan(&count)
		fmt.Printf("Notifikasi untuk kelas 9 di DB: %d (Diharapkan: 1)\n", count)
		mailLog, _ := os.ReadFile("logs/mail.log")
		lines := strings.Split(strings.TrimSpace(string(mailLog)), "\n")
		c9Lines := 0
		for _, l := range lines {
			if strings.Contains(l, "IF307") {
				c9Lines++
				fmt.Printf("  Log mail: %s\n", l)
			}
		}
		fmt.Printf("Baris mail.log untuk kelas 9 (IF307): %d (Diharapkan: 1)\n", c9Lines)

	case "d2":
		runTestD2(ctx, pool, client, tokenAdmin)
	}
}

func runTestD2(ctx context.Context, pool *pgxpool.Pool, client *http.Client, tokenAdmin string) {
	fmt.Println("=== MENJALANKAN UJI D.2 (Notifikasi Perubahan Kuota Kelas) ===")
	_ = os.WriteFile("logs/mail.log", []byte(""), 0644)

	var currentQuota, currentVersion int
	pool.QueryRow(ctx, "SELECT quota, version FROM courses WHERE id = 13").Scan(&currentQuota, &currentVersion)
	fmt.Printf("Status awal Course 13: Quota=%d, Version=%d\n", currentQuota, currentVersion)

	newQuota := currentQuota + 5
	patchBody := map[string]int{"quota": newQuota, "version": currentVersion}
	code, resp, err := doReq(client, "PATCH", baseURL+"/courses/13", tokenAdmin, patchBody)
	fmt.Printf("PATCH /courses/13 -> Status: %d, Err: %v, Response: %s\n", code, err, resp)

	fmt.Println("\nMenunggu 25 detik agar relay & worker memproses fan-out notifikasi ke seluruh peserta...")
	time.Sleep(25 * time.Second)

	var outboxID int64
	var eventType string
	var pub bool
	err = pool.QueryRow(ctx, "SELECT id, event_type, published_at IS NOT NULL FROM outbox WHERE event_type = 'course.quota_changed' ORDER BY id DESC LIMIT 1").
		Scan(&outboxID, &eventType, &pub)
	fmt.Printf("\nOutbox Quota Changed: ID=%d, Event=%s, Published=%v, Err=%v\n", outboxID, eventType, pub, err)

	var notifCount int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM notifications WHERE outbox_id = $1", outboxID).Scan(&notifCount)
	fmt.Printf("Total notifications tercatat di DB untuk outbox %d: %d (Diharapkan: 5)\n", outboxID, notifCount)

	mailLog, _ := os.ReadFile("logs/mail.log")
	lines := strings.Split(strings.TrimSpace(string(mailLog)), "\n")
	quotaLines := 0
	for _, l := range lines {
		if strings.Contains(l, "KRS: kuota IF399 Kelas Rebutan berubah menjadi") {
			quotaLines++
			fmt.Printf("  %s\n", l)
		}
	}
	fmt.Printf("Total baris mail.log perubahan kuota: %d (Diharapkan: 5)\n", quotaLines)
}

func runReset(ctx context.Context, pool *pgxpool.Pool) {
	stmts := []string{
		"DELETE FROM notifications WHERE user_id IN (SELECT id FROM users WHERE username IN ('mhsuji', 'salahalamat'))",
		"DELETE FROM outbox WHERE (payload->>'user_id')::INT IN (SELECT id FROM users WHERE username IN ('mhsuji', 'salahalamat'))",
		"DELETE FROM enrollments WHERE user_id IN (SELECT id FROM users WHERE username IN ('mhsuji', 'salahalamat'))",
		"UPDATE courses c SET enrolled_count = (SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id)",
	}
	for _, s := range stmts {
		if _, err := pool.Exec(ctx, s); err != nil {
			fmt.Printf("Reset error executing %s: %v\n", s, err)
		}
	}
	_ = os.WriteFile("logs/mail.log", []byte(""), 0644)
}

func doReq(client *http.Client, method, url, token string, body any) (int, string, error) {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	} else if method == "POST" || method == "PUT" || method == "PATCH" {
		bodyReader = bytes.NewReader([]byte("{}"))
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	return resp.StatusCode, string(respBytes), err
}

func runTestC1(ctx context.Context, pool *pgxpool.Pool, client *http.Client, token string, user model.User) {
	fmt.Println("=== MENJALANKAN UJI C.1 (Jalur Normal) ===")
	runReset(ctx, pool)

	// 1. POST /courses/2/enroll
	code1, resp1, _ := doReq(client, "POST", baseURL+"/courses/2/enroll", token, nil)
	fmt.Printf("[1] POST /courses/2/enroll -> Status: %d, Response: %s\n", code1, resp1)

	// 2. POST /courses/2/enroll (diulang)
	code2, resp2, _ := doReq(client, "POST", baseURL+"/courses/2/enroll", token, nil)
	fmt.Printf("[2] POST /courses/2/enroll (diulang) -> Status: %d, Response: %s\n", code2, resp2)

	// 3. DELETE /courses/2/enroll
	code3, resp3, _ := doReq(client, "DELETE", baseURL+"/courses/2/enroll", token, nil)
	fmt.Printf("[3] DELETE /courses/2/enroll -> Status: %d, Response: %s\n", code3, resp3)

	// 4. POST /courses/2/enroll
	code4, resp4, _ := doReq(client, "POST", baseURL+"/courses/2/enroll", token, nil)
	fmt.Printf("[4] POST /courses/2/enroll -> Status: %d, Response: %s\n", code4, resp4)

	// 5. POST /enrollments/swap {"from_course_id": 2, "to_course_id": 4}
	swapBody := map[string]int{"from_course_id": 2, "to_course_id": 4}
	code5, resp5, _ := doReq(client, "POST", baseURL+"/enrollments/swap", token, swapBody)
	fmt.Printf("[5] POST /enrollments/swap -> Status: %d, Response: %s\n", code5, resp5)

	fmt.Println("\nMenunggu 5 detik untuk relay dan worker...")
	time.Sleep(5 * time.Second)

	// Verifikasi Outbox
	fmt.Println("\n--- VERIFIKASI TABEL OUTBOX ---")
	rows, err := pool.Query(ctx, "SELECT id, event_type, (published_at IS NOT NULL) AS published, created_at FROM outbox WHERE (payload->>'user_id')::int = $1 ORDER BY id ASC", user.ID)
	if err != nil {
		fmt.Printf("Query outbox error: %v\n", err)
	} else {
		defer rows.Close()
		outboxCount := 0
		for rows.Next() {
			var id int64
			var eventType string
			var pub bool
			var createdAt time.Time
			rows.Scan(&id, &eventType, &pub, &createdAt)
			outboxCount++
			fmt.Printf("  Outbox #%d: ID=%d, Event=%s, Published=%v, CreatedAt=%s\n",
				outboxCount, id, eventType, pub, createdAt.Format(time.RFC3339))
		}
		fmt.Printf("Total Outbox mhsuji: %d (Diharapkan: 4)\n", outboxCount)
	}

	// Verifikasi GET /notifications/mine
	fmt.Println("\n--- VERIFIKASI GET /notifications/mine ---")
	codeMine, respMine, _ := doReq(client, "GET", baseURL+"/notifications/mine", token, nil)
	fmt.Printf("Status: %d\nResponse: %s\n", codeMine, respMine)

	// Verifikasi logs/mail.log
	fmt.Println("\n--- VERIFIKASI logs/mail.log ---")
	mailLog, _ := os.ReadFile("logs/mail.log")
	lines := strings.Split(strings.TrimSpace(string(mailLog)), "\n")
	mhsujiLines := 0
	for _, l := range lines {
		if strings.Contains(l, user.Email) {
			mhsujiLines++
			fmt.Printf("  %s\n", l)
		}
	}
	fmt.Printf("Total baris mail.log mhsuji: %d (Diharapkan: 4)\n", mhsujiLines)
}

func runTestC2Part1(ctx context.Context, client *http.Client, tokenMhs, tokenAdmin string) {
	fmt.Println("=== MENJALANKAN UJI C.2 BAGIAN 1 (Redis Mati, KRS Tetap Berjalan) ===")
	// Daftar ke kelas 5, 6, 7
	for _, cid := range []int{5, 6, 7} {
		start := time.Now()
		code, resp, err := doReq(client, "POST", fmt.Sprintf("%s/courses/%d/enroll", baseURL, cid), tokenMhs, nil)
		dur := time.Since(start)
		fmt.Printf("POST /courses/%d/enroll -> Status: %d, Durasi: %v, Err: %v, Resp: %s\n", cid, code, dur, err, resp)
	}

	fmt.Println("\nMenunggu 5 detik...")
	time.Sleep(5 * time.Second)

	// GET /admin/queue
	codeAdmin, respAdmin, _ := doReq(client, "GET", baseURL+"/admin/queue", tokenAdmin, nil)
	fmt.Printf("GET /admin/queue (saat Redis mati) -> Status: %d, Response: %s\n", codeAdmin, respAdmin)
}

func runTestC2Part2(ctx context.Context, client *http.Client, tokenMhs, tokenAdmin string) {
	fmt.Println("=== MENJALANKAN UJI C.2 BAGIAN 2 (Redis Dinyalakan Kembali) ===")
	fmt.Println("Menunggu 10 detik agar relay memproses outbox...")
	time.Sleep(10 * time.Second)

	codeAdmin, respAdmin, _ := doReq(client, "GET", baseURL+"/admin/queue", tokenAdmin, nil)
	fmt.Printf("GET /admin/queue (setelah Redis hidup) -> Status: %d, Response: %s\n", codeAdmin, respAdmin)

	codeMine, respMine, _ := doReq(client, "GET", baseURL+"/notifications/mine", tokenMhs, nil)
	fmt.Printf("GET /notifications/mine -> Status: %d, Response: %s\n", codeMine, respMine)
}

func runTestC3(ctx context.Context, pool *pgxpool.Pool, client *http.Client, token string, user model.User) {
	fmt.Println("=== MENJALANKAN UJI C.3 (Temporary Failure) ===")
	runReset(ctx, pool)

	for cid := 1; cid <= 6; cid++ {
		code, resp, _ := doReq(client, "POST", fmt.Sprintf("%s/courses/%d/enroll", baseURL, cid), token, nil)
		fmt.Printf("POST /courses/%d/enroll -> Status: %d, Response: %s\n", cid, code, resp)
	}

	fmt.Println("\nMenunggu 20 detik untuk retry Asynq dan pemrosesan...")
	time.Sleep(20 * time.Second)

	// 1. Jumlah di DB notifications
	var notifDB int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM notifications WHERE user_id = $1", user.ID).Scan(&notifDB)
	if err != nil {
		fmt.Printf("Error count DB notifications: %v\n", err)
	}

	// 2. Jumlah di /notifications/mine
	codeMine, respMine, _ := doReq(client, "GET", baseURL+"/notifications/mine", token, nil)
	var mineObj struct {
		Data []model.Notification `json:"data"`
	}
	json.Unmarshal([]byte(respMine), &mineObj)

	// 3. Jumlah baris logs/mail.log
	mailLog, _ := os.ReadFile("logs/mail.log")
	lines := strings.Split(strings.TrimSpace(string(mailLog)), "\n")
	mailCount := 0
	for _, l := range lines {
		if strings.Contains(l, user.Email) {
			mailCount++
		}
	}

	fmt.Printf("\nHASIL C.3:\n")
	fmt.Printf("  1. Database notifications: %d\n", notifDB)
	fmt.Printf("  2. GET /notifications/mine: %d (HTTP %d)\n", len(mineObj.Data), codeMine)
	fmt.Printf("  3. logs/mail.log untuk %s: %d\n", user.Email, mailCount)
	fmt.Printf("  -> Ketiganya sama-sama 6: %v\n", notifDB == 6 && len(mineObj.Data) == 6 && mailCount == 6)
}

func runTestC4(ctx context.Context, pool *pgxpool.Pool, client *http.Client, tokenSalah, tokenAdmin string) {
	fmt.Println("=== MENJALANKAN UJI C.4 (Permanent Failure) ===")
	pool.Exec(ctx, "DELETE FROM enrollments WHERE user_id IN (SELECT id FROM users WHERE username = 'salahalamat')")
	pool.Exec(ctx, "UPDATE courses c SET enrolled_count = (SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id)")

	codeAdminBefore, respBefore, _ := doReq(client, "GET", baseURL+"/admin/queue", tokenAdmin, nil)
	fmt.Printf("Kondisi awal /admin/queue: %s (status %d)\n", respBefore, codeAdminBefore)

	codeEnroll, respEnroll, _ := doReq(client, "POST", baseURL+"/courses/2/enroll", tokenSalah, nil)
	fmt.Printf("POST /courses/2/enroll sebagai salahalamat -> Status: %d, Response: %s\n", codeEnroll, respEnroll)

	fmt.Println("Menunggu 15 detik untuk SkipRetry / archiving...")
	time.Sleep(15 * time.Second)

	codeAdminAfter, respAfter, _ := doReq(client, "GET", baseURL+"/admin/queue", tokenAdmin, nil)
	fmt.Printf("Kondisi akhir /admin/queue: %s (status %d)\n", respAfter, codeAdminAfter)
}

func runTestC5Step1(ctx context.Context, pool *pgxpool.Pool, client *http.Client, token string) {
	fmt.Println("=== MENJALANKAN UJI C.5 LANGKAH 1 (Daftar saat worker mati) ===")
	code, resp, _ := doReq(client, "POST", baseURL+"/courses/8/enroll", token, nil)
	fmt.Printf("POST /courses/8/enroll -> Status: %d, Response: %s\n", code, resp)

	var maxID int64
	var pub *time.Time
	err := pool.QueryRow(ctx, "SELECT id, published_at FROM outbox WHERE id = (SELECT MAX(id) FROM outbox)").Scan(&maxID, &pub)
	fmt.Printf("Outbox event terbaru: ID=%d, published_at=%v, err=%v\n", maxID, pub, err)
}

func runTestC5Step2(ctx context.Context, pool *pgxpool.Pool, token string) {
	fmt.Println("=== MENJALANKAN UJI C.5 LANGKAH 2 (Verifikasi Idempotency) ===")
	var maxID int64
	pool.QueryRow(ctx, "SELECT MAX(id) FROM outbox").Scan(&maxID)

	var notifCount int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM notifications WHERE outbox_id = $1", maxID).Scan(&notifCount)
	fmt.Printf("Notifikasi di DB untuk outbox_id %d: %d (Diharapkan: 1)\n", maxID, notifCount)

	mailLog, _ := os.ReadFile("logs/mail.log")
	lines := strings.Split(strings.TrimSpace(string(mailLog)), "\n")
	c8Lines := 0
	for _, l := range lines {
		if strings.Contains(l, "IF306") {
			c8Lines++
		}
	}
	fmt.Printf("Baris mail.log untuk kelas IF306: %d (Diharapkan: 1)\n", c8Lines)
}
