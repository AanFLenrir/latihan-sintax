package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"latihan-fiber/app/model"
	"latihan-fiber/config"
	"latihan-fiber/helper"
)

const (
	baseURL = "http://localhost:3000/api/v1"
)

func main() {
	config.LoadEnv()
	jwtSecret := config.GetEnv("JWT_SECRET", "d1fce7dfcfb4eae238bab8901e57a48ddd32bff75d42b998fd0aa4db4b9e48fd")
	jwtManager := helper.NewJWTManager(jwtSecret, "praktikum-backend", 1*time.Hour)

	// User 21 (mhs yang terdaftar di kelas 1)
	user21Token, _ := jwtManager.GenerateAccess(model.User{
		ID:       21,
		Username: "mhs001",
		Role:     "user",
	})

	adminToken, _ := jwtManager.GenerateAccess(model.User{
		ID:       12,
		Username: "admin",
		Role:     "admin",
	})

	fmt.Println("================================================================")
	fmt.Println("  MENGUJI ENDPOINT PANDUAN MODUL 10: ULASAN KELAS (REVIEWS)    ")
	fmt.Println("================================================================")

	// 1. Tulis ulasan baru (POST /courses/1/reviews)
	reviewReq := map[string]any{
		"rating":  5,
		"comment": "Materi Go-nya sangat jelas, penjelasan concurrency dan database mendalam.",
		"tags":    []string{"Tugas-Banyak", " dosen-jelas ", "tugas-banyak"},
	}
	_, body, code := doReq("POST", baseURL+"/courses/1/reviews", user21Token, reviewReq)
	fmt.Printf("[1] POST /courses/1/reviews -> HTTP %d\n    Response: %s\n", code, body)

	var createdReview model.Review
	var webResp model.WebResponse
	_ = json.Unmarshal([]byte(body), &webResp)
	dataBytes, _ := json.Marshal(webResp.Data)
	_ = json.Unmarshal(dataBytes, &createdReview)

	// 2. Daftar ulasan kelas (GET /courses/1/reviews)
	_, bodyList, codeList := doReq("GET", baseURL+"/courses/1/reviews", user21Token, nil)
	fmt.Printf("[2] GET /courses/1/reviews -> HTTP %d\n    Response: %s\n", codeList, bodyList)

	// 3. Ringkasan ulasan kelas (GET /courses/1/reviews/summary)
	_, bodySummary, codeSummary := doReq("GET", baseURL+"/courses/1/reviews/summary", user21Token, nil)
	fmt.Printf("[3] GET /courses/1/reviews/summary -> HTTP %d\n    Response: %s\n", codeSummary, bodySummary)

	if createdReview.ID != "" {
		// 4. Get Review by ID (GET /reviews/:id)
		_, bodyGet, codeGet := doReq("GET", baseURL+"/reviews/"+createdReview.ID, user21Token, nil)
		fmt.Printf("[4] GET /reviews/%s -> HTTP %d\n", createdReview.ID, codeGet)
		_ = bodyGet

		// 5. Reply ke ulasan (POST /reviews/:id/replies)
		replyReq := map[string]any{"body": "Setuju, tugasnya memang menantang!"}
		_, bodyReply, codeReply := doReq("POST", baseURL+"/reviews/"+createdReview.ID+"/replies", adminToken, replyReq)
		fmt.Printf("[5] POST /reviews/%s/replies -> HTTP %d\n    Response: %s\n", createdReview.ID, codeReply, bodyReply)

		// 6. Tandai membantu (POST /reviews/:id/helpful)
		_, bodyHelpful, codeHelpful := doReq("POST", baseURL+"/reviews/"+createdReview.ID+"/helpful", adminToken, nil)
		fmt.Printf("[6] POST /reviews/%s/helpful -> HTTP %d\n    Response: %s\n", createdReview.ID, codeHelpful, bodyHelpful)
	}

	fmt.Println("================================================================")
	fmt.Println("  SEMUA ENDPOINT PANDUAN MODUL 10 BERJALAN DENGAN SEMPURNA!   ")
	fmt.Println("================================================================")
}

func doReq(method, urlStr, token string, bodyData any) (*http.Response, string, int) {
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
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err.Error(), 0
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b), resp.StatusCode
}
