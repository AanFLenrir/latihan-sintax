package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"latihan-fiber/app/model"
	"latihan-fiber/config"
	"latihan-fiber/database"
	"latihan-fiber/helper"
)

func fire(n int, job func(i int) string) (map[string]int, time.Duration) {
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan string, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		// BUG FIX #3: gunakan 'chan<- string' (send-only), BUKAN '<-chan string' (receive-only)
		go func(i int, out chan<- string) {
			defer wg.Done()
			<-start
			out <- job(i)
		}(i, results)
	}

	startTime := time.Now()
	close(start) // buka gerbang serentak
	wg.Wait()
	duration := time.Since(startTime)
	close(results)

	tally := map[string]int{}
	for r := range results {
		tally[r]++
	}
	return tally, duration
}

func printTally(tally map[string]int, duration time.Duration) {
	fmt.Println("hasil request:")
	for status, count := range tally {
		fmt.Printf("  %-25s %d\n", status, count)
	}
	fmt.Printf("waktu total : %v\n", duration.Round(time.Millisecond))
}

func main() {
	mode := flag.String("mode", "crowd", "Mode pengujian: crowd, credits, swap, supervisor, reset")
	flag.Parse()

	config.LoadEnv()
	jwtSecret := config.GetEnv("JWT_SECRET", "d1fce7dfcfb4eae238bab8901e57a48ddd32bff75d42b998fd0aa4db4b9e48fd")
	jwtManager := helper.NewJWTManager(jwtSecret, config.GetEnv("JWT_ISSUER", "praktikum-backend"), 15*time.Minute)

	pool, err := database.NewPool(context.Background())
	if err != nil {
		fmt.Printf("gagal terhubung ke database: %v\n", err)
		return
	}
	defer pool.Close()

	baseURL := fmt.Sprintf("http://localhost:%s/api/v1", config.GetEnv("APP_PORT", "3000"))
	client := &http.Client{Timeout: 10 * time.Second}

	// Ambil semua akun mahasiswa mhs001 s.d. mhs060 dari database
	users, err := loadMhsUsers(pool)
	if err != nil {
		fmt.Printf("gagal memuat users mhs: %v\n", err)
		return
	}
	if len(users) < 60 {
		fmt.Printf("peringatan: jumlah user mhs hanya %d, diharapkan 60\n", len(users))
	}

	// Siapkan access token untuk setiap mahasiswa
	tokens := make([]string, len(users))
	for i, u := range users {
		token, err := jwtManager.GenerateAccess(u)
		if err != nil {
			fmt.Printf("gagal generate token untuk user %s: %v\n", u.Username, err)
			return
		}
		tokens[i] = token
	}

	// Ambil course ID untuk IF399, IF101, IF102
	courseIDs, err := loadCourseIDs(pool)
	if err != nil {
		fmt.Printf("gagal memuat data courses: %v\n", err)
		return
	}

	switch *mode {
	case "crowd":
		runCrowdTest(client, baseURL, tokens, courseIDs["IF399"], pool)
	case "credits":
		runCreditsTest(client, baseURL, tokens, courseIDs, pool)
	case "swap":
		runSwapTest(client, baseURL, tokens, courseIDs["IF101"], courseIDs["IF102"], pool)
	case "supervisor":
		runSupervisorTest(client, baseURL, tokens, pool)
	case "reset":
		resetDatabase(pool)
		fmt.Println("Database reset selesai.")
	default:
		fmt.Printf("mode tidak dikenal: %s\n", *mode)
	}
}

func loadMhsUsers(pool *pgxpool.Pool) ([]model.User, error) {
	rows, err := pool.Query(context.Background(),
		"SELECT id, username, email, role FROM users WHERE username LIKE 'mhs%' ORDER BY username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func loadCourseIDs(pool *pgxpool.Pool) (map[string]int, error) {
	rows, err := pool.Query(context.Background(), "SELECT id, code FROM courses")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]int)
	for rows.Next() {
		var id int
		var code string
		if err := rows.Scan(&id, &code); err != nil {
			return nil, err
		}
		m[code] = id
	}
	return m, rows.Err()
}

func resetDatabase(pool *pgxpool.Pool) {
	ctx := context.Background()
	_, _ = pool.Exec(ctx, "DELETE FROM enrollments WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'mhs%')")
	_, _ = pool.Exec(ctx, "UPDATE courses c SET enrolled_count = (SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id)")
}

func sendRequest(client *http.Client, method, url, token string, body any) string {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	} else if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		bodyReader = bytes.NewReader([]byte("{}"))
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Sprintf("ERR: %v", err)
	}
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("ERR: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		return "200 OK"
	}
	if resp.StatusCode == http.StatusCreated {
		return "201 OK"
	}

	var errResp struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(respBody, &errResp)
	if errResp.Code != "" {
		return fmt.Sprintf("%d %s", resp.StatusCode, errResp.Code)
	}
	return fmt.Sprintf("%d", resp.StatusCode)
}

func runCrowdTest(client *http.Client, baseURL string, tokens []string, courseID int, pool *pgxpool.Pool) {
	fmt.Println("60 mahasiswa berebut kelas IF399 secara serentak...")
	url := fmt.Sprintf("%s/courses/%d/enroll", baseURL, courseID)

	n := 60
	if len(tokens) < n {
		n = len(tokens)
	}

	tally, duration := fire(n, func(i int) string {
		return sendRequest(client, http.MethodPost, url, tokens[i], nil)
	})

	printTally(tally, duration)

	// Verifikasi database
	var quota, enrolledCount, actual int
	_ = pool.QueryRow(context.Background(),
		`SELECT c.quota, c.enrolled_count, COUNT(e.id)
		FROM courses c
		LEFT JOIN enrollments e ON e.course_id = c.id
		WHERE c.id = $1
		GROUP BY c.id, c.quota, c.enrolled_count`, courseID).Scan(&quota, &enrolledCount, &actual)

	status := "-> TIDAK KONSISTEN"
	if enrolledCount == quota && actual == quota {
		status = "-> KONSISTEN"
	}
	fmt.Printf("IF399 kuota=%d enrolled_count=%d baris enrollments=%d %s\n", quota, enrolledCount, actual, status)
}

func runCreditsTest(client *http.Client, baseURL string, tokens []string, courseIDs map[string]int, pool *pgxpool.Pool) {
	fmt.Println("10 mahasiswa, masing-masing mendaftar ke 12 kelas 3 SKS secara serentak (batas 24 SKS)...")

	// Urutkan 12 kelas non-IF399
	codes := []string{"IF101", "IF102", "IF301", "IF302", "IF303", "IF304", "IF305", "IF306", "IF307", "IF308", "IF309", "IF310"}

	type jobItem struct {
		token    string
		courseID int
	}
	var jobs []jobItem
	for i := 0; i < 10 && i < len(tokens); i++ {
		for _, code := range codes {
			if id, ok := courseIDs[code]; ok {
				jobs = append(jobs, jobItem{token: tokens[i], courseID: id})
			}
		}
	}

	tally, duration := fire(len(jobs), func(i int) string {
		j := jobs[i]
		url := fmt.Sprintf("%s/courses/%d/enroll", baseURL, j.courseID)
		return sendRequest(client, http.MethodPost, url, j.token, nil)
	})

	printTally(tally, duration)

	// Verifikasi database: pastikan tidak ada mahasiswa > 24 SKS
	rows, _ := pool.Query(context.Background(),
		`SELECT u.username, COALESCE(SUM(c.credits), 0) AS total_credits
		FROM users u
		LEFT JOIN enrollments e ON e.user_id = u.id
		LEFT JOIN courses c ON c.id = e.course_id
		WHERE u.username LIKE 'mhs%'
		GROUP BY u.id, u.username
		HAVING COALESCE(SUM(c.credits), 0) > 24`)
	defer rows.Close()

	violationCount := 0
	for rows.Next() {
		var uname string
		var credits int
		_ = rows.Scan(&uname, &credits)
		fmt.Printf("PELANGGARAN: %s memiliki %d SKS (> 24)!\n", uname, credits)
		violationCount++
	}

	if violationCount == 0 {
		fmt.Println("seluruh 10 mahasiswa <= 24 SKS -> KONSISTEN")
	} else {
		fmt.Println("-> TIDAK KONSISTEN: terdapat mahasiswa melewati batas 24 SKS")
	}
}

func runSwapTest(client *http.Client, baseURL string, tokens []string, c1, c2 int, pool *pgxpool.Pool) {
	fmt.Println("20 pasang mahasiswa pindah kelas berlawanan arah secara serentak...")

	// Persiapan: 20 mahasiswa mendaftar c1, 20 mahasiswa mendaftar c2
	resetDatabase(pool)
	for i := 0; i < 20; i++ {
		url := fmt.Sprintf("%s/courses/%d/enroll", baseURL, c1)
		sendRequest(client, http.MethodPost, url, tokens[i], nil)
	}
	for i := 20; i < 40; i++ {
		url := fmt.Sprintf("%s/courses/%d/enroll", baseURL, c2)
		sendRequest(client, http.MethodPost, url, tokens[i], nil)
	}

	type swapJob struct {
		token string
		from  int
		to    int
	}
	var jobs []swapJob
	for i := 0; i < 20; i++ {
		jobs = append(jobs, swapJob{token: tokens[i], from: c1, to: c2})
	}
	for i := 20; i < 40; i++ {
		jobs = append(jobs, swapJob{token: tokens[i], from: c2, to: c1})
	}

	tally, duration := fire(len(jobs), func(i int) string {
		j := jobs[i]
		url := fmt.Sprintf("%s/courses/swap", baseURL)
		body := model.SwapRequest{FromCourseID: j.from, ToCourseID: j.to}
		return sendRequest(client, http.MethodPost, url, j.token, body)
	})

	printTally(tally, duration)

	var q1, e1, a1, q2, e2, a2 int
	_ = pool.QueryRow(context.Background(),
		`SELECT c.quota, c.enrolled_count, COUNT(e.id)
		FROM courses c LEFT JOIN enrollments e ON e.course_id = c.id
		WHERE c.id = $1 GROUP BY c.id, c.quota, c.enrolled_count`, c1).Scan(&q1, &e1, &a1)
	_ = pool.QueryRow(context.Background(),
		`SELECT c.quota, c.enrolled_count, COUNT(e.id)
		FROM courses c LEFT JOIN enrollments e ON e.course_id = c.id
		WHERE c.id = $1 GROUP BY c.id, c.quota, c.enrolled_count`, c2).Scan(&q2, &e2, &a2)

	fmt.Printf("IF101 kuota=%d enrolled_count=%d baris enrollments=%d -> %s\n", q1, e1, a1, checkConsistent(e1, a1))
	fmt.Printf("IF102 kuota=%d enrolled_count=%d baris enrollments=%d -> %s\n", q2, e2, a2, checkConsistent(e2, a2))
}

func checkConsistent(a, b int) string {
	if a == b {
		return "KONSISTEN"
	}
	return "TIDAK KONSISTEN"
}

func runSupervisorTest(client *http.Client, baseURL string, tokens []string, pool *pgxpool.Pool) {
	fmt.Println("30 mahasiswa serentak mendaftar ke 1 dosen pembimbing berkuota 3...")

	// Dapatkan ID supervisor pertama berkuota 3
	var supID, quota, supervisedCount int
	err := pool.QueryRow(context.Background(), "SELECT id, quota, supervised_count FROM supervisors WHERE quota = 3 ORDER BY id LIMIT 1").
		Scan(&supID, &quota, &supervisedCount)
	if err != nil {
		fmt.Printf("supervisor berkuota 3 belum tersedia di database: %v\n", err)
		return
	}

	// Buat token admin untuk assign
	jwtSecret := config.GetEnv("JWT_SECRET", "d1fce7dfcfb4eae238bab8901e57a48ddd32bff75d42b998fd0aa4db4b9e48fd")
	jwtManager := helper.NewJWTManager(jwtSecret, config.GetEnv("JWT_ISSUER", "praktikum-backend"), 15*time.Minute)
	adminToken, err := jwtManager.GenerateAccess(model.User{ID: 1, Username: "admin", Role: "admin"})
	if err != nil {
		fmt.Printf("gagal membuat token admin: %v\n", err)
		return
	}

	// Reset data supervisions untuk testing
	_, _ = pool.Exec(context.Background(), "DELETE FROM supervisions WHERE supervisor_id = $1", supID)
	_, _ = pool.Exec(context.Background(), "UPDATE supervisors SET supervised_count = 0 WHERE id = $1", supID)

	// Dapatkan 30 student_id
	rows, err := pool.Query(context.Background(), "SELECT id FROM students ORDER BY id LIMIT 30")
	if err != nil {
		fmt.Printf("gagal memuat students: %v\n", err)
		return
	}
	defer rows.Close()

	var studentIDs []int
	for rows.Next() {
		var sid int
		_ = rows.Scan(&sid)
		studentIDs = append(studentIDs, sid)
	}

	if len(studentIDs) < 30 {
		fmt.Printf("jumlah student hanya %d, diharapkan minimal 30 untuk uji beban\n", len(studentIDs))
	}

	n := len(studentIDs)
	tally, duration := fire(n, func(i int) string {
		url := fmt.Sprintf("%s/supervisors/%d/assign", baseURL, supID)
		body := map[string]int{"student_id": studentIDs[i]}
		return sendRequest(client, http.MethodPost, url, adminToken, body)
	})

	printTally(tally, duration)

	var q, sc, actual int
	_ = pool.QueryRow(context.Background(),
		`SELECT s.quota, s.supervised_count, COUNT(sv.student_id)
		FROM supervisors s
		LEFT JOIN supervisions sv ON sv.supervisor_id = s.id
		WHERE s.id = $1
		GROUP BY s.id, s.quota, s.supervised_count`, supID).Scan(&q, &sc, &actual)

	status := "-> TIDAK KONSISTEN"
	if sc == q && actual == q {
		status = "-> KONSISTEN"
	}
	fmt.Printf("Supervisor #%d: kuota=%d supervised_count=%d baris supervisions=%d %s\n", supID, q, sc, actual, status)
}
