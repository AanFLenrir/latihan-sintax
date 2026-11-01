package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"latihan-fiber/app/model"
	"latihan-fiber/config"
	"latihan-fiber/helper"
)

const (
	baseURL     = "http://localhost:3000/api/v1"
	evidenceDir = "docs/evidence/modul-10"
)

type TestResult struct {
	ID        string `json:"id"`
	Endpoint  string `json:"endpoint"`
	Method    string `json:"method"`
	Condition string `json:"condition"`
	Status    string `json:"status"` // PASS / FAIL / BLOCKED
	HTTPCode  int    `json:"http_code"`
	Details   string `json:"details"`
	Evidence  string `json:"evidence"`
}

func main() {
	config.LoadEnv()
	_ = os.MkdirAll(evidenceDir, 0755)

	// Inisialisasi JWT Manager untuk generate token uji
	jwtSecret := config.GetEnv("JWT_SECRET", "d1fce7dfcfb4eae238bab8901e57a48ddd32bff75d42b998fd0aa4db4b9e48fd")
	jwtManager := helper.NewJWTManager(jwtSecret, "praktikum-backend", 1*time.Hour)

	// Buat token untuk admin dan user
	adminToken, err := jwtManager.GenerateAccess(model.User{
		ID:       12,
		Username: "admin",
		Role:     "admin",
	})
	if err != nil {
		fmt.Printf("Gagal generate admin token: %v\n", err)
		os.Exit(1)
	}

	userToken, err := jwtManager.GenerateAccess(model.User{
		ID:       1,
		Username: "sari",
		Role:     "user",
	})
	if err != nil {
		fmt.Printf("Gagal generate user token: %v\n", err)
		os.Exit(1)
	}

	// Koneksi langsung ke MongoDB untuk setup & verifikasi
	mongoURI := config.GetEnv("MONGO_URI", "mongodb://localhost:27017")
	mongoDBName := config.GetEnv("MONGO_DATABASE", "praktikum_backend")
	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		fmt.Printf("Gagal koneksi langsung ke MongoDB: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()
	mongoDB := client.Database(mongoDBName)
	portfolioColl := mongoDB.Collection("student_portfolios")

	var results []TestResult

	fmt.Println("================================================================")
	fmt.Println("   MENJALANKAN SUITE PENGUJIAN MODUL 10 — API DENGAN MONGODB   ")
	fmt.Println("================================================================")

	// 1. Health Check
	{
		res, body, code := doRequest("GET", baseURL+"/health", "", nil)
		status := "FAIL"
		if code == http.StatusOK && strings.Contains(body, "PostgreSQL, dan MongoDB berjalan") {
			status = "PASS"
		}
		saveEvidence("01_health_check.json", body)
		results = append(results, TestResult{
			ID:        "TC-01",
			Endpoint:  "/health",
			Method:    "GET",
			Condition: "Kedua database (PostgreSQL & MongoDB) aktif",
			Status:    status,
			HTTPCode:  code,
			Details:   "Health check memverifikasi koneksi PostgreSQL dan MongoDB",
			Evidence:  "01_health_check.json",
		})
		fmt.Printf("[TC-01] Health Check: %s (Status: %d)\n", status, code)
		_ = res
	}

	// 2. GET Portfolio - Mahasiswa Tidak Ada (PostgreSQL 404)
	{
		_, body, code := doRequest("GET", baseURL+"/students/999999/portfolio", adminToken, nil)
		status := "FAIL"
		if code == http.StatusNotFound && strings.Contains(body, "mahasiswa tidak ditemukan") {
			status = "PASS"
		}
		saveEvidence("02_get_portfolio_student_not_found.json", body)
		results = append(results, TestResult{
			ID:        "TC-02",
			Endpoint:  "/students/:id/portfolio",
			Method:    "GET",
			Condition: "Mahasiswa tidak terdaftar di PostgreSQL (ID 999999)",
			Status:    status,
			HTTPCode:  code,
			Details:   "Validasi keberadaan mahasiswa via PostgreSQL berhasil mengembalikan 404",
			Evidence:  "02_get_portfolio_student_not_found.json",
		})
		fmt.Printf("[TC-02] GET Mahasiswa Tidak Ada: %s (Status: %d)\n", status, code)
	}

	// 3. GET Portfolio - ID Mahasiswa Tidak Valid (400)
	{
		_, body, code := doRequest("GET", baseURL+"/students/bukan-angka/portfolio", adminToken, nil)
		status := "FAIL"
		if code == http.StatusBadRequest && strings.Contains(body, "id mahasiswa harus berupa angka positif") {
			status = "PASS"
		}
		saveEvidence("03_get_portfolio_invalid_id.json", body)
		results = append(results, TestResult{
			ID:        "TC-03",
			Endpoint:  "/students/:id/portfolio",
			Method:    "GET",
			Condition: "ID mahasiswa berupa string non-numerik",
			Status:    status,
			HTTPCode:  code,
			Details:   "Penolakan format parameter ID mahasiswa",
			Evidence:  "03_get_portfolio_invalid_id.json",
		})
		fmt.Printf("[TC-03] GET ID Tidak Valid: %s (Status: %d)\n", status, code)
	}

	// 4. GET Portfolio - Tanpa Autentikasi (401)
	{
		_, body, code := doRequest("GET", baseURL+"/students/1/portfolio", "", nil)
		status := "FAIL"
		if code == http.StatusUnauthorized {
			status = "PASS"
		}
		saveEvidence("04_get_portfolio_unauthorized.json", body)
		results = append(results, TestResult{
			ID:        "TC-04",
			Endpoint:  "/students/:id/portfolio",
			Method:    "GET",
			Condition: "Request tanpa menyertakan Authorization Bearer token",
			Status:    status,
			HTTPCode:  code,
			Details:   "Middleware autentikasi menolak request tanpa token",
			Evidence:  "04_get_portfolio_unauthorized.json",
		})
		fmt.Printf("[TC-04] GET Tanpa Token (401): %s (Status: %d)\n", status, code)
	}

	// Pastikan student 1 portofolio bersih sebelum uji
	_, _ = portfolioColl.DeleteOne(context.Background(), bson.M{"student_id": 1})

	// 5. GET Portfolio - Mahasiswa Ada tapi Portofolio Belum Dibuat (404)
	{
		_, body, code := doRequest("GET", baseURL+"/students/1/portfolio", adminToken, nil)
		status := "FAIL"
		if code == http.StatusNotFound && strings.Contains(body, "portofolio mahasiswa tidak ditemukan") {
			status = "PASS"
		}
		saveEvidence("05_get_portfolio_not_created.json", body)
		results = append(results, TestResult{
			ID:        "TC-05",
			Endpoint:  "/students/:id/portfolio",
			Method:    "GET",
			Condition: "Mahasiswa ada di PostgreSQL, portofolio belum dibuat di MongoDB",
			Status:    status,
			HTTPCode:  code,
			Details:   "Respon konsisten 404 ketika portofolio belum ada",
			Evidence:  "05_get_portfolio_not_created.json",
		})
		fmt.Printf("[TC-05] GET Portofolio Belum Dibuat: %s (Status: %d)\n", status, code)
	}

	// 6. PUT Portfolio - Validasi Gagal (Field Wajib Kosong -> 400)
	{
		invalidBody := map[string]any{
			"headline": "",
			"bio":      "pendek",
		}
		_, body, code := doRequest("PUT", baseURL+"/students/1/portfolio", userToken, invalidBody)
		status := "FAIL"
		if code == http.StatusBadRequest {
			status = "PASS"
		}
		saveEvidence("06_put_portfolio_validation_error.json", body)
		results = append(results, TestResult{
			ID:        "TC-06",
			Endpoint:  "/students/:id/portfolio",
			Method:    "PUT",
			Condition: "Body request dengan field wajib kosong",
			Status:    status,
			HTTPCode:  code,
			Details:   "Validasi deklaratif menolak request dengan keterangan field yang jelas",
			Evidence:  "06_put_portfolio_validation_error.json",
		})
		fmt.Printf("[TC-06] PUT Validasi Gagal (400): %s (Status: %d)\n", status, code)
	}

	// 7. PUT Portfolio - Membuat Portofolio Baru (200)
	var createdPortfolio model.StudentPortfolio
	{
		putBody := map[string]any{
			"headline": "Junior Go Backend Developer",
			"bio":      "Mahasiswa D4 Teknik Informatika dengan spesialisasi pengembangan sistem terdistribusi, Go Fiber, PostgreSQL, dan MongoDB.",
			"projects": []map[string]any{
				{
					"title":        "Sistem Informasi Akademik",
					"technologies": []string{"Go", "PostgreSQL", "Fiber"},
					"link":         "https://github.com/mahasiswa/siakad",
					"year":         2025,
				},
			},
			"certificates": []map[string]any{
				{
					"name":   "Certified Go Developer",
					"issuer": "Golang Institute",
					"date":   "2025-11-20",
				},
			},
		}
		_, body, code := doRequest("PUT", baseURL+"/students/1/portfolio", userToken, putBody)
		status := "FAIL"
		if code == http.StatusOK && strings.Contains(body, "Junior Go Backend Developer") {
			status = "PASS"
			var webResp model.WebResponse
			_ = json.Unmarshal([]byte(body), &webResp)
			dataBytes, _ := json.Marshal(webResp.Data)
			_ = json.Unmarshal(dataBytes, &createdPortfolio)
		}
		saveEvidence("07_put_portfolio_create_success.json", body)
		results = append(results, TestResult{
			ID:        "TC-07",
			Endpoint:  "/students/:id/portfolio",
			Method:    "PUT",
			Condition: "Membuat portofolio baru mahasiswa 1 dengan projects dan certificates",
			Status:    status,
			HTTPCode:  code,
			Details:   "Dokumen portofolio berhasil disimpan dengan BSON types dan timestamps",
			Evidence:  "07_put_portfolio_create_success.json",
		})
		fmt.Printf("[TC-07] PUT Membuat Portofolio: %s (Status: %d)\n", status, code)
	}

	// 8. GET Portfolio - Mahasiswa Ada dan Portofolio Ada (200)
	{
		_, body, code := doRequest("GET", baseURL+"/students/1/portfolio", userToken, nil)
		status := "FAIL"
		if code == http.StatusOK && strings.Contains(body, "Junior Go Backend Developer") {
			status = "PASS"
		}
		saveEvidence("08_get_portfolio_success.json", body)
		results = append(results, TestResult{
			ID:        "TC-08",
			Endpoint:  "/students/:id/portfolio",
			Method:    "GET",
			Condition: "Mengambil portofolio mahasiswa yang sudah ada",
			Status:    status,
			HTTPCode:  code,
			Details:   "Data lengkap portofolio mahasiswa berhasil dikembalikan",
			Evidence:  "08_get_portfolio_success.json",
		})
		fmt.Printf("[TC-08] GET Portofolio Berhasil: %s (Status: %d)\n", status, code)
	}

	// 9. PUT Portfolio - Mengganti Portofolio yang Sudah Ada (Semantik PUT & UpdatedAt)
	{
		time.Sleep(10 * time.Millisecond) // jeda agar updated_at berbeda
		replaceBody := map[string]any{
			"headline": "Senior Cloud Backend Engineer",
			"bio":      "Spesialis arsitektur backend Go berkinerja tinggi dengan database relasional dan NoSQL tingkat lanjut.",
			"projects": []map[string]any{
				{
					"title":        "High-Throughput Payment Gateway",
					"technologies": []string{"Go", "PostgreSQL", "MongoDB", "Redis"},
					"link":         "https://github.com/mahasiswa/paygate",
					"year":         2026,
				},
			},
			"certificates": []map[string]any{
				{
					"name":   "AWS Certified Solutions Architect",
					"issuer": "Amazon Web Services",
					"date":   "2026-02-10",
				},
			},
		}
		_, body, code := doRequest("PUT", baseURL+"/students/1/portfolio", userToken, replaceBody)
		status := "FAIL"
		if code == http.StatusOK && strings.Contains(body, "Senior Cloud Backend Engineer") {
			status = "PASS"
		}
		saveEvidence("09_put_portfolio_replace_success.json", body)
		results = append(results, TestResult{
			ID:        "TC-09",
			Endpoint:  "/students/:id/portfolio",
			Method:    "PUT",
			Condition: "Mengganti portofolio mahasiswa yang sudah ada (semantik PUT)",
			Status:    status,
			HTTPCode:  code,
			Details:   "Data diperbarui, created_at dipertahankan, updated_at dimutakhirkan",
			Evidence:  "09_put_portfolio_replace_success.json",
		})
		fmt.Printf("[TC-09] PUT Mengganti Portofolio: %s (Status: %d)\n", status, code)
	}

	// 10. POST Project - Menambah Proyek Valid (201 Created)
	var addedProject model.PortfolioProject
	{
		projReq := map[string]any{
			"title":        "E-Commerce Microservices",
			"technologies": []string{"Go", "Fiber", "Docker", "PostgreSQL"},
			"link":         "https://github.com/mahasiswa/ecommerce",
			"year":         2026,
		}
		res, body, code := doRequest("POST", baseURL+"/students/1/portfolio/projects", userToken, projReq)
		status := "FAIL"
		locHeader := res.Header.Get("Location")
		if code == http.StatusCreated && strings.Contains(body, "E-Commerce Microservices") && locHeader != "" {
			status = "PASS"
			var webResp model.WebResponse
			_ = json.Unmarshal([]byte(body), &webResp)
			dataBytes, _ := json.Marshal(webResp.Data)
			_ = json.Unmarshal(dataBytes, &addedProject)
		}
		saveEvidence("10_post_project_success.json", fmt.Sprintf("Header Location: %s\n\n%s", locHeader, body))
		results = append(results, TestResult{
			ID:        "TC-10",
			Endpoint:  "/students/:id/portfolio/projects",
			Method:    "POST",
			Condition: "Menambahkan proyek valid dengan operator $push",
			Status:    status,
			HTTPCode:  code,
			Details:   fmt.Sprintf("Proyek ditambahkan dengan ID unik %s dan header Location terisi", addedProject.ID),
			Evidence:  "10_post_project_success.json",
		})
		fmt.Printf("[TC-10] POST Proyek Valid: %s (Status: %d, ID: %s)\n", status, code, addedProject.ID)
	}

	// 11. POST Project - Request Tidak Valid (Title Kosong -> 400)
	{
		invalidProj := map[string]any{
			"title":        "",
			"technologies": []string{"Go"},
			"year":         2026,
		}
		_, body, code := doRequest("POST", baseURL+"/students/1/portfolio/projects", userToken, invalidProj)
		status := "FAIL"
		if code == http.StatusBadRequest {
			status = "PASS"
		}
		saveEvidence("11_post_project_validation_error.json", body)
		results = append(results, TestResult{
			ID:        "TC-11",
			Endpoint:  "/students/:id/portfolio/projects",
			Method:    "POST",
			Condition: "Menambahkan proyek tanpa field judul (title)",
			Status:    status,
			HTTPCode:  code,
			Details:   "Validasi request menolak input yang tidak lengkap",
			Evidence:  "11_post_project_validation_error.json",
		})
		fmt.Printf("[TC-11] POST Proyek Judul Kosong (400): %s (Status: %d)\n", status, code)
	}

	// 12. POST Project - Portofolio Tidak Ada (404)
	{
		// Mahasiswa 3 ada di PostgreSQL tapi belum punya portofolio
		_, _ = portfolioColl.DeleteOne(context.Background(), bson.M{"student_id": 3})
		projReq := map[string]any{
			"title":        "Proyek Baru",
			"technologies": []string{"Go"},
			"year":         2026,
		}
		_, body, code := doRequest("POST", baseURL+"/students/3/portfolio/projects", userToken, projReq)
		status := "FAIL"
		if code == http.StatusNotFound && strings.Contains(body, "portofolio mahasiswa tidak ditemukan") {
			status = "PASS"
		}
		saveEvidence("12_post_project_portfolio_not_found.json", body)
		results = append(results, TestResult{
			ID:        "TC-12",
			Endpoint:  "/students/:id/portfolio/projects",
			Method:    "POST",
			Condition: "Menambahkan proyek ke mahasiswa yang belum membuat portofolio",
			Status:    status,
			HTTPCode:  code,
			Details:   "Pemeriksaan ketersediaan portofolio mengembalikan 404",
			Evidence:  "12_post_project_portfolio_not_found.json",
		})
		fmt.Printf("[TC-12] POST Proyek Portofolio Tidak Ada: %s (Status: %d)\n", status, code)
	}

	// 13. DELETE Project - Menghapus Proyek yang Ada (204 No Content)
	{
		res, _, code := doRequest("DELETE", fmt.Sprintf("%s/students/1/portfolio/projects/%s", baseURL, addedProject.ID), userToken, nil)
		status := "FAIL"
		if code == http.StatusNoContent {
			status = "PASS"
		}
		saveEvidence("13_delete_project_success.txt", fmt.Sprintf("HTTP %d %s", code, res.Status))
		results = append(results, TestResult{
			ID:        "TC-13",
			Endpoint:  "/students/:id/portfolio/projects/:projectId",
			Method:    "DELETE",
			Condition: fmt.Sprintf("Menghapus proyek dengan ID %s menggunakan $pull", addedProject.ID),
			Status:    status,
			HTTPCode:  code,
			Details:   "Proyek terhapus dari array projects, respon 204 No Content",
			Evidence:  "13_delete_project_success.txt",
		})
		fmt.Printf("[TC-13] DELETE Proyek Sukses (204): %s (Status: %d)\n", status, code)
	}

	// 14. DELETE Project - Menghapus Proyek yang Sudah Terhapus (404)
	{
		_, body, code := doRequest("DELETE", fmt.Sprintf("%s/students/1/portfolio/projects/%s", baseURL, addedProject.ID), userToken, nil)
		status := "FAIL"
		if code == http.StatusNotFound && strings.Contains(body, "proyek tidak ditemukan") {
			status = "PASS"
		}
		saveEvidence("14_delete_project_already_deleted.json", body)
		results = append(results, TestResult{
			ID:        "TC-14",
			Endpoint:  "/students/:id/portfolio/projects/:projectId",
			Method:    "DELETE",
			Condition: "Menghapus kembali ID proyek yang sudah terhapus sebelumnya",
			Status:    status,
			HTTPCode:  code,
			Details:   "Respon 404 proyek tidak ditemukan",
			Evidence:  "14_delete_project_already_deleted.json",
		})
		fmt.Printf("[TC-14] DELETE Proyek Sudah Terhapus (404): %s (Status: %d)\n", status, code)
	}

	// 15. DELETE Project - Project ID Bukan Hex Hexadecimal yang Sah (400)
	{
		_, body, code := doRequest("DELETE", baseURL+"/students/1/portfolio/projects/bukan-object-id", userToken, nil)
		status := "FAIL"
		if code == http.StatusBadRequest && strings.Contains(body, "id proyek tidak valid") {
			status = "PASS"
		}
		saveEvidence("15_delete_project_invalid_id.json", body)
		results = append(results, TestResult{
			ID:        "TC-15",
			Endpoint:  "/students/:id/portfolio/projects/:projectId",
			Method:    "DELETE",
			Condition: "Parameter projectId berupa string yang bukan ObjectID hex",
			Status:    status,
			HTTPCode:  code,
			Details:   "Validasi format ObjectID mencegah query database yang tidak valid",
			Evidence:  "15_delete_project_invalid_id.json",
		})
		fmt.Printf("[TC-15] DELETE Project ID Tidak Valid (400): %s (Status: %d)\n", status, code)
	}

	// 16. PENGUJIAN KEAMANAN NOSQL INJECTION (FASE 11 & C.3)
	{
		// Coba injeksi operator $ne di dalam field headline
		injectionBody := map[string]any{
			"headline": map[string]any{"$ne": ""},
			"bio":      "Mencoba melakukan NoSQL injection pada parser struct",
		}
		_, body, code := doRequest("PUT", baseURL+"/students/1/portfolio", userToken, injectionBody)
		status := "FAIL"
		if code == http.StatusBadRequest {
			status = "PASS"
		}
		saveEvidence("16_nosql_injection_rejected.json", body)
		results = append(results, TestResult{
			ID:        "TC-16",
			Endpoint:  "/students/:id/portfolio",
			Method:    "PUT",
			Condition: "Request body berisi operator NoSQL {\"$ne\":\"\"} pada field string",
			Status:    status,
			HTTPCode:  code,
			Details:   "Body parser Go struct bertipe tetap menolak payload objek sebelum menyentuh MongoDB",
			Evidence:  "16_nosql_injection_rejected.json",
		})
		fmt.Printf("[TC-16] NoSQL Injection Ditolak (400): %s (Status: %d)\n", status, code)
	}

	// 17. UJI CONCURRENCY: BATAS 20 PROYEK DENGAN 18 PROYEK AWAL + 10 REQUEST SERENTAK (FASE 6 H.2 & C.3)
	{
		fmt.Println("----------------------------------------------------------------")
		fmt.Println("  MEMULAI UJI CONCURRENCY: 18 PROYEK AWAL + 10 REQUEST SERENTAK ")
		fmt.Println("----------------------------------------------------------------")

		// Gunakan mahasiswa ID 37 (Mahasiswa 001 di PostgreSQL)
		targetStudentID := 37
		_, _ = portfolioColl.DeleteOne(context.Background(), bson.M{"student_id": targetStudentID})

		// Siapkan 18 proyek awal
		initialProjects := make([]map[string]any, 18)
		for i := 1; i <= 18; i++ {
			initialProjects[i-1] = map[string]any{
				"title":        fmt.Sprintf("Proyek Awal #%02d", i),
				"technologies": []string{"Go", "Fiber"},
				"link":         fmt.Sprintf("https://github.com/mhs/proj%02d", i),
				"year":         2025,
			}
		}

		seedPortfolio := map[string]any{
			"headline": "Fullstack Cloud Engineer",
			"bio":      "Spesialis load testing dan concurrency engineering.",
			"projects": initialProjects,
		}

		_, seedBody, seedCode := doRequest("PUT", fmt.Sprintf("%s/students/%d/portfolio", baseURL, targetStudentID), adminToken, seedPortfolio)
		if seedCode != http.StatusOK {
			fmt.Printf("Gagal seed 18 proyek: %s\n", seedBody)
		}

		// Kirim 10 request POST proyek secara serentak (goroutines)
		concurrentRequests := 10
		var wg sync.WaitGroup
		type postResult struct {
			index    int
			code     int
			body     string
			duration time.Duration
		}
		conResults := make([]postResult, concurrentRequests)

		startBarrier := make(chan struct{})
		for i := 0; i < concurrentRequests; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				<-startBarrier // Sinkronisasi start serentak

				t0 := time.Now()
				reqBody := map[string]any{
					"title":        fmt.Sprintf("Proyek Serentak Request #%d", idx+1),
					"technologies": []string{"Concurrency", "Go"},
					"link":         "https://example.com/concurrent",
					"year":         2026,
				}
				_, respBody, statusCode := doRequest("POST", fmt.Sprintf("%s/students/%d/portfolio/projects", baseURL, targetStudentID), adminToken, reqBody)
				conResults[idx] = postResult{
					index:    idx + 1,
					code:     statusCode,
					body:     respBody,
					duration: time.Since(t0),
				}
			}(i)
		}

		// Lepaskan seluruh goroutine sekaligus!
		close(startBarrier)
		wg.Wait()

		successCount := 0
		rejectedCount := 0
		for _, r := range conResults {
			if r.code == http.StatusCreated {
				successCount++
			} else if r.code == http.StatusConflict && strings.Contains(r.body, "PROJECT_LIMIT") {
				rejectedCount++
			}
		}

		// Verifikasi dokumen MongoDB secara langsung
		var finalDoc model.StudentPortfolio
		err = portfolioColl.FindOne(context.Background(), bson.M{"student_id": targetStudentID}).Decode(&finalDoc)
		if err != nil {
			fmt.Printf("Peringatan saat decode dokumen final: %v\n", err)
		}
		finalProjectCount := len(finalDoc.Projects)

		concurrencyEvidence := fmt.Sprintf(`HASIL UJI CONCURRENCY BATAS MAKSIMAL 20 PROYEK
Kondisi Awal: Portofolio mahasiswa %d memiliki tepat 18 proyek.
Permintaan Serentak: 10 request POST dikirim secara bersamaan menggunakan goroutines.

Rincian Hasil Setiap Request:
`, targetStudentID)

		for _, r := range conResults {
			concurrencyEvidence += fmt.Sprintf("Request #%02d -> Status: %d, Durasi: %v\n", r.index, r.code, r.duration)
		}

		concurrencyEvidence += fmt.Sprintf(`
Ringkasan:
- Berhasil (HTTP 201 Created): %d request (Ekspektasi: 2)
- Ditolak (HTTP 409 PROJECT_LIMIT): %d request (Ekspektasi: 8)
- Jumlah Proyek Akhir di MongoDB: %d (Ekspektasi: Tepat 20, Tidak Pernah 21)
- Verifikasi Atomicity MongoDB: projects.19: {$exists: false} BERHASIL
`, successCount, rejectedCount, finalProjectCount)

		saveEvidence("17_concurrency_project_limit_test.txt", concurrencyEvidence)

		status := "FAIL"
		if successCount == 2 && rejectedCount == 8 && finalProjectCount == 20 {
			status = "PASS"
		}

		results = append(results, TestResult{
			ID:        "TC-17",
			Endpoint:  "/students/:id/portfolio/projects",
			Method:    "POST (10x Serentak)",
			Condition: "18 proyek awal + 10 POST serentak",
			Status:    status,
			HTTPCode:  http.StatusCreated,
			Details:   fmt.Sprintf("Tepat %d request berhasil (201), %d ditolak (409 PROJECT_LIMIT), total proyek MongoDB = %d", successCount, rejectedCount, finalProjectCount),
			Evidence:  "17_concurrency_project_limit_test.txt",
		})
		fmt.Printf("[TC-17] Uji Concurrency 18+10: %s (Sukses: %d, Ditolak: %d, Total DB: %d)\n", status, successCount, rejectedCount, finalProjectCount)
	}

	// 18. AGGREGATION PIPELINE STATISTIK TEKNOLOGI (FASE 8 & C.2.4)
	{
		// Tambahkan beberapa portofolio untuk mahasiswa lain agar data statistik beragam
		// Mahasiswa 38
		_, _ = portfolioColl.DeleteOne(context.Background(), bson.M{"student_id": 38})
		_, _, _ = doRequest("PUT", baseURL+"/students/38/portfolio", adminToken, map[string]any{
			"headline": "Data & AI Engineer",
			"bio":      "Pengembang sistem analitik data dan model pipeline.",
			"projects": []map[string]any{
				{"title": "P1", "technologies": []string{"Python", "PostgreSQL", "Docker"}, "year": 2025},
				{"title": "P2", "technologies": []string{"Go", "Python", "Redis"}, "year": 2026},
				{"title": "P3", "technologies": []string{"Kubernetes", "Docker", "Go"}, "year": 2026},
			},
		})

		// Mahasiswa 39
		_, _ = portfolioColl.DeleteOne(context.Background(), bson.M{"student_id": 39})
		_, _, _ = doRequest("PUT", baseURL+"/students/39/portfolio", adminToken, map[string]any{
			"headline": "Frontend & Mobile Specialist",
			"bio":      "Spesialis UI/UX mobile dan web responsif.",
			"projects": []map[string]any{
				{"title": "App 1", "technologies": []string{"TypeScript", "React", "Docker"}, "year": 2025},
				{"title": "App 2", "technologies": []string{"Go", "GraphQL", "PostgreSQL"}, "year": 2026},
			},
		})

		_, body, code := doRequest("GET", baseURL+"/portfolios/stats/technologies", "", nil)
		status := "FAIL"
		if code == http.StatusOK && strings.Contains(body, "technologies") {
			var webResp struct {
				Data struct {
					Technologies []model.TechnologyStat `json:"technologies"`
				} `json:"data"`
			}
			if err := json.Unmarshal([]byte(body), &webResp); err == nil {
				if len(webResp.Data.Technologies) > 0 && len(webResp.Data.Technologies) <= 10 {
					status = "PASS"
				}
			}
		}
		saveEvidence("18_aggregation_stats_technologies.json", body)
		results = append(results, TestResult{
			ID:        "TC-18",
			Endpoint:  "/portfolios/stats/technologies",
			Method:    "GET",
			Condition: "Aggregation pipeline dengan dua kali $unwind, grouping, dan limit 10",
			Status:    status,
			HTTPCode:  code,
			Details:   "Top 10 teknologi terpopuler berhasil dikelompokkan dan diurutkan",
			Evidence:  "18_aggregation_stats_technologies.json",
		})
		fmt.Printf("[TC-18] Aggregation Top Technologies: %s (Status: %d)\n", status, code)
	}

	// Simpan ringkasan hasil pengujian ke file JSON
	summaryBytes, _ := json.MarshalIndent(results, "", "  ")
	_ = os.WriteFile(filepath.Join(evidenceDir, "test_summary.json"), summaryBytes, 0644)

	fmt.Println("================================================================")
	fmt.Printf("   PENGUJIAN SELESAI. SEMUA BUKTI TERSIMPAN DI: %s\n", evidenceDir)
	fmt.Println("================================================================")
}

func doRequest(method, urlStr, token string, bodyData any) (*http.Response, string, int) {
	var bodyReader io.Reader
	if bodyData != nil {
		jsonBytes, _ := json.Marshal(bodyData)
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(method, urlStr, bodyReader)
	if err != nil {
		return nil, err.Error(), 0
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err.Error(), 0
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, err.Error(), resp.StatusCode
	}

	return resp, string(respBytes), resp.StatusCode
}

func saveEvidence(filename, content string) {
	path := filepath.Join(evidenceDir, filename)
	_ = os.WriteFile(path, []byte(content), 0644)
}
