package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"latihan-fiber/app/model"
	"latihan-fiber/config"
	"latihan-fiber/database"
	"latihan-fiber/helper"
)

func main() {
	config.LoadEnv()
	jwtSecret := config.GetEnv("JWT_SECRET", "d1fce7dfcfb4eae238bab8901e57a48ddd32bff75d42b998fd0aa4db4b9e48fd")
	jwtManager := helper.NewJWTManager(jwtSecret, config.GetEnv("JWT_ISSUER", "praktikum-backend"), 15*time.Minute)

	pool, err := database.NewPool(context.Background())
	if err != nil {
		fmt.Printf("DB error: %v\n", err)
		return
	}
	defer pool.Close()

	// Reset database
	_, _ = pool.Exec(context.Background(), "DELETE FROM enrollments WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'mhs%')")
	_, _ = pool.Exec(context.Background(), "UPDATE courses c SET enrolled_count = (SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id)")

	// Ambil user mhs001 s.d. mhs007
	var mhsUsers []model.User
	rows, _ := pool.Query(context.Background(), "SELECT id, username, email, role FROM users WHERE username LIKE 'mhs%' ORDER BY username LIMIT 10")
	for rows.Next() {
		var u model.User
		_ = rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role)
		mhsUsers = append(mhsUsers, u)
	}
	rows.Close()

	mhsTokens := make([]string, len(mhsUsers))
	for i, u := range mhsUsers {
		mhsTokens[i], _ = jwtManager.GenerateAccess(u)
	}

	adminToken, _ := jwtManager.GenerateAccess(model.User{ID: 1, Username: "admin", Role: "admin"})

	baseURL := "http://localhost:3000/api/v1"
	client := &http.Client{Timeout: 5 * time.Second}

	// Cari ID course
	var if399ID, if101ID, if102ID int
	_ = pool.QueryRow(context.Background(), "SELECT id FROM courses WHERE code = 'IF399'").Scan(&if399ID)
	_ = pool.QueryRow(context.Background(), "SELECT id FROM courses WHERE code = 'IF101'").Scan(&if101ID)
	_ = pool.QueryRow(context.Background(), "SELECT id FROM courses WHERE code = 'IF102'").Scan(&if102ID)

	fmt.Println("=== PENGUJIAN FUNGSIONAL SATU PER SATU ===")

	// 1. Daftar ke kelas yang masih kosong -> 201
	status, body := doReq(client, "POST", fmt.Sprintf("%s/courses/%d/enroll", baseURL, if101ID), mhsTokens[0], nil)
	fmt.Printf("1. Daftar ke kelas masih kosong (mhs001 ke IF101): HTTP %d\n   Resp: %s\n", status, body)

	// 2. Daftar ke kelas yang sama sekali lagi -> 409 ALREADY_ENROLLED
	status, body = doReq(client, "POST", fmt.Sprintf("%s/courses/%d/enroll", baseURL, if101ID), mhsTokens[0], nil)
	fmt.Printf("2. Daftar duplikat (mhs001 ke IF101): HTTP %d\n   Resp: %s\n", status, body)

	// Isi IF399 sampai penuh (5 mahasiswa: mhs001 s.d. mhs005)
	for i := 0; i < 5; i++ {
		doReq(client, "POST", fmt.Sprintf("%s/courses/%d/enroll", baseURL, if399ID), mhsTokens[i], nil)
	}

	// 3. Mahasiswa ke-6 mendaftar ke IF399 (kuota 5) -> 409 COURSE_FULL
	status, body = doReq(client, "POST", fmt.Sprintf("%s/courses/%d/enroll", baseURL, if399ID), mhsTokens[5], nil)
	fmt.Printf("3. Mahasiswa ke-6 ke IF399 (kuota 5): HTTP %d\n   Resp: %s\n", status, body)

	// 4. Batalkan kelas yang tidak pernah diambil -> 409 NOT_ENROLLED
	status, body = doReq(client, "DELETE", fmt.Sprintf("%s/courses/%d/enroll", baseURL, if102ID), mhsTokens[0], nil)
	fmt.Printf("4. Batalkan kelas yang tidak pernah diambil (mhs001 dari IF102): HTTP %d\n   Resp: %s\n", status, body)

	// 5. Pindah ke kelas yang penuh (mhs006 di IF101 mau pindah ke IF399 yang penuh)
	// Daftarkan dulu mhs006 ke IF101
	doReq(client, "POST", fmt.Sprintf("%s/courses/%d/enroll", baseURL, if101ID), mhsTokens[5], nil)
	status, body = doReq(client, "POST", fmt.Sprintf("%s/courses/swap", baseURL), mhsTokens[5], model.SwapRequest{
		FromCourseID: if101ID,
		ToCourseID:   if399ID,
	})
	fmt.Printf("5. Pindah ke kelas penuh (mhs006 dari IF101 ke IF399): HTTP %d\n   Resp: %s\n", status, body)

	// Cek apakah mhs006 tetap terdaftar di IF101
	var countInOld int
	_ = pool.QueryRow(context.Background(), "SELECT count(*) FROM enrollments WHERE course_id = $1 AND user_id = $2", if101ID, mhsUsers[5].ID).Scan(&countInOld)
	fmt.Printf("   mhs006 tetap di IF101: %v (count=%d)\n", countInOld == 1, countInOld)

	// 6. Pindah ke kelas yang masih kosong (mhs006 dari IF101 ke IF102) -> 200
	status, body = doReq(client, "POST", fmt.Sprintf("%s/courses/swap", baseURL), mhsTokens[5], model.SwapRequest{
		FromCourseID: if101ID,
		ToCourseID:   if102ID,
	})
	fmt.Printf("6. Pindah ke kelas kosong (mhs006 dari IF101 ke IF102): HTTP %d\n   Resp: %s\n", status, body)

	// 7. Pindah dengan from_course_id == to_course_id -> 422 VALIDATION_ERROR
	status, body = doReq(client, "POST", fmt.Sprintf("%s/courses/swap", baseURL), mhsTokens[5], model.SwapRequest{
		FromCourseID: if101ID,
		ToCourseID:   if101ID,
	})
	fmt.Printf("7. Pindah kelas sama (from==to): HTTP %d\n   Resp: %s\n", status, body)

	// 8. Admin ubah kuota dengan version terbaru (IF101 version=1, ubah quota=45) -> 200
	var curVersion int
	_ = pool.QueryRow(context.Background(), "SELECT version FROM courses WHERE id = $1", if101ID).Scan(&curVersion)
	status, body = doReq(client, "PATCH", fmt.Sprintf("%s/courses/%d", baseURL, if101ID), adminToken, model.UpdateQuotaRequest{
		Quota:   45,
		Version: curVersion,
	})
	fmt.Printf("8. Admin ubah kuota version terbaru (v=%d): HTTP %d\n   Resp: %s\n", curVersion, status, body)

	// 9. Admin ubah kuota dengan version lama (kirim version lama yang sudah lewat) -> 409 VERSION_CONFLICT
	status, body = doReq(client, "PATCH", fmt.Sprintf("%s/courses/%d", baseURL, if101ID), adminToken, model.UpdateQuotaRequest{
		Quota:   50,
		Version: curVersion, // version lama
	})
	fmt.Printf("9. Admin ubah kuota version lama (v=%d): HTTP %d\n   Resp: %s\n", curVersion, status, body)

	// 10. Admin menurunkan kuota di bawah jumlah peserta -> 409 QUOTA_BELOW_ENROLLED
	// IF399 punya 5 peserta, coba turunkan kuota jadi 4
	_ = pool.QueryRow(context.Background(), "SELECT version FROM courses WHERE id = $1", if399ID).Scan(&curVersion)
	status, body = doReq(client, "PATCH", fmt.Sprintf("%s/courses/%d", baseURL, if399ID), adminToken, model.UpdateQuotaRequest{
		Quota:   4, // dibawah peserta saat ini (5)
		Version: curVersion,
	})
	fmt.Printf("10. Admin menurunkan kuota < jumlah peserta: HTTP %d\n    Resp: %s\n", status, body)

	// 11. Mahasiswa mencoba mengubah kuota -> 403 FORBIDDEN
	status, body = doReq(client, "PATCH", fmt.Sprintf("%s/courses/%d", baseURL, if101ID), mhsTokens[0], model.UpdateQuotaRequest{
		Quota:   50,
		Version: 1,
	})
	fmt.Printf("11. Mahasiswa ubah kuota: HTTP %d\n    Resp: %s\n", status, body)

	fmt.Println("\n=== PENGUJIAN FUNGSIONAL SUPERVISORS (D.2) ===")

	// 12. Admin ubah kuota supervisor version terbaru -> 200
	var supVersion int
	_ = pool.QueryRow(context.Background(), "SELECT version FROM supervisors WHERE id = 1").Scan(&supVersion)
	status, body = doReq(client, "PATCH", fmt.Sprintf("%s/supervisors/1", baseURL), adminToken, model.UpdateSupervisorQuotaRequest{
		Quota:   4,
		Version: supVersion,
	})
	fmt.Printf("12. Admin ubah kuota supervisor version terbaru (v=%d): HTTP %d\n    Resp: %s\n", supVersion, status, body)

	// 13. Admin ubah kuota supervisor version lama -> 409 VERSION_CONFLICT
	status, body = doReq(client, "PATCH", fmt.Sprintf("%s/supervisors/1", baseURL), adminToken, model.UpdateSupervisorQuotaRequest{
		Quota:   5,
		Version: supVersion, // version lama
	})
	fmt.Printf("13. Admin ubah kuota supervisor version lama (v=%d): HTTP %d\n    Resp: %s\n", supVersion, status, body)

	// 14. Switch supervisor: tetapkan student 1 ke supervisor 1 dulu
	var studentID1, studentID2 int
	_ = pool.QueryRow(context.Background(), "SELECT id FROM students ORDER BY id LIMIT 1").Scan(&studentID1)
	_ = pool.QueryRow(context.Background(), "SELECT id FROM students ORDER BY id OFFSET 1 LIMIT 1").Scan(&studentID2)

	_, _ = pool.Exec(context.Background(), "DELETE FROM supervisions WHERE supervisor_id IN (1, 2)")
	_, _ = pool.Exec(context.Background(), "UPDATE supervisors SET supervised_count = 0 WHERE id IN (1, 2)")

	doReq(client, "POST", fmt.Sprintf("%s/supervisors/1/assign", baseURL), adminToken, model.AssignSupervisorRequest{StudentID: studentID1})

	// Switch student 1 dari supervisor 1 ke supervisor 2 -> 200 OK
	status, body = doReq(client, "POST", fmt.Sprintf("%s/supervisions/switch", baseURL), adminToken, model.SwitchSupervisorRequest{
		StudentID:       studentID1,
		NewSupervisorID: 2,
	})
	fmt.Printf("14. Switch supervisor yang sah (student %d dari sup 1 ke sup 2): HTTP %d\n    Resp: %s\n", studentID1, status, body)

	// 15. Switch supervisor ke supervisor yang penuh -> 409 SUPERVISOR_FULL dan mahasiswa tetap di supervisor lama
	// Penuhi supervisor 1 (kuota saat ini 4, isi 4 mahasiswa)
	var testStudents []int
	sRows, _ := pool.Query(context.Background(), "SELECT id FROM students WHERE id != $1 ORDER BY id LIMIT 4", studentID1)
	for sRows.Next() {
		var sid int
		_ = sRows.Scan(&sid)
		testStudents = append(testStudents, sid)
	}
	sRows.Close()

	for _, sid := range testStudents {
		doReq(client, "POST", fmt.Sprintf("%s/supervisors/1/assign", baseURL), adminToken, model.AssignSupervisorRequest{StudentID: sid})
	}

	// Coba switch student 1 (saat ini di supervisor 2) ke supervisor 1 yang penuh (4/4)
	status, body = doReq(client, "POST", fmt.Sprintf("%s/supervisions/switch", baseURL), adminToken, model.SwitchSupervisorRequest{
		StudentID:       studentID1,
		NewSupervisorID: 1,
	})
	fmt.Printf("15. Switch ke supervisor penuh: HTTP %d\n    Resp: %s\n", status, body)

	// Verifikasi student 1 tetap di supervisor 2
	var currentSup int
	_ = pool.QueryRow(context.Background(), "SELECT supervisor_id FROM supervisions WHERE student_id = $1", studentID1).Scan(&currentSup)
	fmt.Printf("    Mahasiswa %d tetap di supervisor lama (sup 2): %v (sup_id=%d)\n", studentID1, currentSup == 2, currentSup)
}

func doReq(client *http.Client, method, url, token string, body any) (int, string) {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	} else if method == "POST" || method == "PUT" || method == "PATCH" {
		bodyReader = bytes.NewReader([]byte("{}"))
	}

	req, _ := http.NewRequest(method, url, bodyReader)
	if method == "POST" || method == "PUT" || method == "PATCH" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}
