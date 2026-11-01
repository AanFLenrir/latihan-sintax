# AUDIT FINAL MODUL 10 — API DENGAN MONGODB
**Program Studi:** D4 Teknik Informatika — Fakultas Vokasi, Universitas Airlangga  
**Mata Kuliah:** Praktikum Pemrograman Backend Lanjut  
**Mahasiswa:** MOHAMMAD ANHAZ ABDILAH  
**NIM:** 434241102  
**Tanggal Audit:** 25 Oktober 2026  
**Status Keseluruhan:** **SELESAI (100% PASS)**

---

## 1. Ringkasan Eksekutif

Pekerjaan praktikum Modul 10 ("API dengan MongoDB") telah diselesaikan secara penuh dan komprehensif pada proyek Go Fiber `c:\Users\anhaz\belajar-go\latihan-fiber` dengan menjaga kompatibilitas utuh terhadap implementasi Modul 9 (transaksi relasional, kuota kelas, KRS, dan pembimbing skripsi).

Arsitektur yang dibangun adalah arsitektur *polyglot persistence* murni:
1. **PostgreSQL** mempertahankan peran sebagai *Single Source of Truth* (SSOT) untuk entitas `users`, `students`, `courses`, `enrollments`, dan `supervisors`.
2. **MongoDB** (driver resmi `go.mongodb.org/mongo-driver/v2` versi 2.9.2) digunakan untuk collection dokumen `student_portfolios` yang menyimpan biodata mahasiswa, headline, riwayat proyek (maksimal 20 item), dan sertifikat.
3. Seluruh interaksi MongoDB diisolasi secara ketat pada layer *Repository*, model domain pada `app/model` bebas dari dependensi BSON driver, dan pemeriksaan keberadaan mahasiswa lintas database dikelola oleh layer *Service*.

---

## 2. File yang Dibuat & Diubah

### File Baru (Tugas Mandiri & Langkah Panduan Modul 10):
1. `latihan.js` — Skrip mongosh latihan CRUD, embedding, dan indexing dari Langkah 2.
2. `database/mongo.go` — Inisialisasi client MongoDB driver v2, setting timeout, pool size, dan verifikasi koneksi via `client.Ping` (Langkah 3).
3. `app/model/review.go` — Domain model ulasan kelas (`Review`, `Reply`, `ReviewSummary`, query DTO) dari Langkah 4.
4. `app/model/portfolio.go` — Domain model murni portofolio (`StudentPortfolio`, `PortfolioProject`, `PortfolioCertificate`), DTO request (`UpsertPortfolioRequest`, `AddProjectRequest`), dan DTO response (`TechnologyStatsResponse`).
5. `app/repository/review_repository.go` — Repository MongoDB ulasan kelas (`course_reviews`) dengan indeks unik `{course_id, user_id}`, multikey tag, sorting cursor, atomic reply limit (`replies.49`), atomic helpful vote (`$addToSet` + `$inc`), dan summary facet aggregation (Langkah 5, 6, 7).
6. `app/repository/student_repository.go` — Repository PostgreSQL untuk validasi eksistensi mahasiswa (`SELECT EXISTS(...) FROM students WHERE id = $1`).
7. `app/repository/portfolio_repository.go` — Repository MongoDB lengkap mencakup `EnsureIndexes`, `FindByStudentID`, `Upsert`, `AddProject` (atomic `$push` kuota 20), `DeleteProject` (atomic `$pull`), dan `GetTechnologyStats` (aggregation pipeline double `$unwind`).
8. `app/service/review_rules.go` — Aturan normalisasi tag ulasan (`NormalizeTags`) dari Langkah 8.
9. `app/service/review_service.go` — Service layer ulasan kelas yang memadukan validasi pendaftaran PostgreSQL (`IsEnrolled`) dengan penyimpanan MongoDB (Langkah 8).
10. `app/service/portfolio_rules.go` — Aturan bisnis normalisasi teknologi (`NormalizeTechnologies`) dan pemeriksaan batas proyek (`CanAddProject`).
11. `app/service/portfolio_rules_test.go` — Unit tests untuk aturan bisnis normalisasi dan batas kuota proyek.
12. `app/service/portfolio_service.go` — Service layer yang menghubungkan PostgreSQL dan MongoDB serta memetakan domain error ke HTTP envelope.
13. `migrations/008_review_permissions.sql` — Migrasi permission `review:moderate` untuk peran admin (Langkah 8).
14. `tools/test_reviews/main.go` — Test suite untuk verifikasi endpoint panduan ulasan kelas (POST review, GET reviews, summary, reply, helpful).
15. `tools/test_modul10/main.go` — Automated test harness komprehensif menguji 18 skenario fungsional, autentikasi, NoSQL injection, dan konkurensi 18+10.
16. `tools/test_resilience/main.go` — Test suite untuk pengujian ketahanan dua database (PostgreSQL down, MongoDB down, dan recovery).
17. `tools/generate_pdf.py` — Script generator PDF menggunakan engine headless Chromium/Edge.
18. `docs/laporan-modul-10.md` — Sumber laporan teknis Modul 10 dalam format Markdown.
19. `docs/laporan_styled.html` — Sumber template HTML dengan styling cetak A4 profesional.
20. `docs/Tugas10_434241102.pdf` — Dokumen PDF laporan akhir resmi siap kumpul.
21. Folder `docs/evidence/modul-10/` — Berisi 23 file bukti eksekusi aktual (JSON, TXT, dan test summary).

### File yang Diperbarui:
1. `go.mod` & `go.sum` — Penambahan driver resmi MongoDB v2 (`go.mongodb.org/mongo-driver/v2 v2.9.2`) dan dependensi hashing/koneksi.
2. `.env` & `.env.example` — Penambahan variabel `MONGO_URI`, `MONGO_DATABASE`, dan `MONGO_MAX_POOL`.
3. `helper/validator.go` — Penyempurnaan pesan validasi slice/array/map pada tag `min` dan `max` agar menampilkan satuan `"item"` alih-alih `"karakter"` (Langkah 8).
4. `app/repository/course_repository.go` — Penambahan method `IsEnrolled(ctx, courseID, userID)` pada interface dan repository PostgreSQL (Langkah 8).
5. `route/route.go` — Pendaftaran dependency `Mongo`, `PortfolioService`, dan `ReviewService`, pembaruan handler `/api/v1/health` agar memvalidasi kedua database, serta pendaftaran seluruh route ulasan kelas, portofolio mahasiswa, dan agregasi.
6. `main.go` — Inisialisasi koneksi MongoDB client, setup repository portfolio dan review, auto-create index (`EnsureIndexes`), dan wiring dependency injection.

---

## 3. Daftar Endpoint & Status Verifikasi

| HTTP Method | Endpoint Route | Deskripsi & Autentikasi | Database Terkait | Status Uji |
|:---|:---|:---|:---|:---:|
| `GET` | `/api/v1/health` | Health check memeriksa PostgreSQL & MongoDB | PostgreSQL & MongoDB | **PASS (200)** |
| `GET` | `/api/v1/students/:id/portfolio` | Mengambil portofolio mahasiswa (Bearer Auth) | PostgreSQL & MongoDB | **PASS (200)** |
| `PUT` | `/api/v1/students/:id/portfolio` | Membuat / mengganti portofolio (Bearer Auth) | PostgreSQL & MongoDB | **PASS (200)** |
| `POST` | `/api/v1/students/:id/portfolio/projects` | Menambah proyek secara atomik (Bearer Auth) | PostgreSQL & MongoDB | **PASS (201)** |
| `DELETE` | `/api/v1/students/:id/portfolio/projects/:projectId` | Menghapus proyek dengan `$pull` (Bearer Auth) | PostgreSQL & MongoDB | **PASS (204)** |
| `GET` | `/api/v1/portfolios/stats/technologies` | Agregasi Top 10 teknologi terpopuler | MongoDB Only | **PASS (200)** |

---

## 4. Indeks Database MongoDB

Diverifikasi melalui perintah `db.student_portfolios.getIndexes()`:
1. `_id_`: Default Primary Key BSON ObjectId.
2. `student_portfolio_unique`:
   - Keys: `{ student_id: 1 }`
   - Unique: `true`
   - Tujuan: Menjamin secara mutlak bahwa satu mahasiswa hanya memiliki tepat satu dokumen portofolio pada tingkat database, mencegah race condition duplicate-insert.
3. `projects_technologies_idx`:
   - Keys: `{ projects.technologies: 1 }`
   - Multikey: `true` (setiap elemen array teknologi diindeks)
   - Tujuan: Mempercepat operasi pencarian dan pemilahan dokumen berdasarkan teknologi.

### Bukti Explain `executionStats`:
Query: `db.student_portfolios.find({student_id: 1}).explain("executionStats")`
- **Winning Plan Stage:** `EXPRESS_IXSCAN`
- **Index Used:** `student_portfolio_unique`
- **totalKeysExamined:** `1`
- **totalDocsExamined:** `1`
- **nReturned:** `1`
- **Waktu Eksekusi:** 3 ms
- **COLLSCAN:** 0 (Tidak ada scan tabel)

---

## 5. Matriks Hasil Pengujian Kualitas Kode & Fungsional

| Kategori Pengujian | Perintah / Tool | Target | Status | Keterangan |
|:---|:---|:---|:---:|:---|
| Static Analysis | `go vet ./...` | Seluruh codebase | **PASS** | 0 error, 0 warning |
| Build Verification | `go build ./...` | Kompilasi binary | **PASS** | Binary berhasil dikompilasi |
| Unit Testing | `go test ./...` | Unit test domain rules | **PASS** | 100% test lulus (`app/service`) |
| Fungsional Portfolio | `tools/test_modul10` | 18 skenario uji API | **PASS** | Validasi input, 404 PG, 400 ID, 201 created |
| Keamanan NoSQL Injection | `tools/test_modul10` | Payload `{"$ne":""}` | **PASS** | Ditolak parser struct (HTTP 400), data aman |
| Concurrency Kuota 20 | `tools/test_modul10` | 18 proyek + 10 POST serentak | **PASS** | Tepat 2 sukses (201), 8 ditolak (409), total = 20 |
| Aggregation Pipeline | `tools/test_modul10` | Double `$unwind` 10 teknologi | **PASS** | Urutan total DESC, tie-breaker alfabetis |
| Ketahanan Polyglot (PG Down) | `tools/test_resilience` | Simulasi PostgreSQL down | **PASS** | Health 503, Portofolio 500, Stats 200 (MongoDB OK) |
| Ketahanan Polyglot (Mongo Down)| `tools/test_resilience` | Simulasi MongoDB down | **PASS** | Health 503, Portofolio 500, Kelas PG 200 (PG OK) |
| Pemulihan Koneksi | `tools/test_resilience` | Recovery handshake | **PASS** | Otomatis pulih ke status HTTP 200 tanpa restart |

---

## 6. Lokasi Berkas & Bukti Pengujian

- **Dokumen Laporan PDF:** `docs/Tugas10_434241102.pdf` (Ukuran: 201 KB, format A4 resmi)
- **Sumber Laporan Markdown:** `docs/laporan-modul-10.md`
- **Dokumen Audit Akhir:** `docs/audit-final-modul-10.md`
- **Folder Bukti Pengujian:** `docs/evidence/modul-10/`
  - `01_health_check.json` — Bukti respons health check saat kedua DB aktif
  - `02_get_portfolio_student_not_found.json` — Kasus gagal: Mahasiswa tidak ada di PostgreSQL (404)
  - `03_get_portfolio_invalid_id.json` — Kasus gagal: ID mahasiswa non-numerik (400)
  - `04_get_portfolio_unauthorized.json` — Kasus gagal: Request tanpa token Bearer (401)
  - `05_get_portfolio_not_created.json` — Kasus gagal: Portofolio belum dibuat (404)
  - `06_put_portfolio_validation_error.json` — Kasus gagal: Field wajib kosong (400)
  - `07_put_portfolio_create_success.json` — Sukses: Membuat portofolio baru (200)
  - `08_get_portfolio_success.json` — Sukses: Membaca portofolio mahasiswa (200)
  - `09_put_portfolio_replace_success.json` — Sukses: Semantik PUT replace portofolio (200)
  - `10_post_project_success.json` — Sukses: Menambah proyek via `$push` dan header `Location` (201)
  - `11_post_project_validation_error.json` — Kasus gagal: Menambah proyek tanpa title (400)
  - `12_post_project_portfolio_not_found.json` — Kasus gagal: Menambah proyek pada portofolio non-eksis (404)
  - `13_delete_project_success.txt` — Sukses: Menghapus proyek via `$pull` (204)
  - `14_delete_project_already_deleted.json` — Kasus gagal: Menghapus proyek yang sudah terhapus (404)
  - `15_delete_project_invalid_id.json` — Kasus gagal: projectId bukan hex ObjectID (400)
  - `16_nosql_injection_rejected.json` — Bukti penolakan payload NoSQL injection (400)
  - `17_concurrency_project_limit_test.txt` — Bukti uji konkurensi 18+10 menghasilkan tepat 20 proyek
  - `18_aggregation_stats_technologies.json` — Bukti response agregasi top 10 teknologi
  - `19_mongo_indexes.json` — Output mongosh `getIndexes()`
  - `20_explain_execution_stats.json` — Output mongosh `explain("executionStats")`
  - `21_complete_portfolio_document.json` — Dokumen portofolio lengkap dari database
  - `22_dual_database_resilience.txt` — Bukti pengujian ketahanan kegagalan dua database
  - `test_summary.json` — Rekapitulasi status pengujian otomatis

---

## 7. Cara Menjalankan Aplikasi & Pengujian

### Prasyarat:
- Go 1.26+
- PostgreSQL berjalan di `localhost:5432` dengan database `praktikum_backend`
- MongoDB berjalan di `localhost:27017` dengan database `praktikum_backend`

### Langkah Menjalankan Aplikasi:
1. Pastikan file konfigurasi `.env` telah memuat variabel:
   ```env
   APP_PORT=3000
   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=anhazaan1357
   DB_NAME=praktikum_backend
   MONGO_URI=mongodb://localhost:27017
   MONGO_DATABASE=praktikum_backend
   MONGO_MAX_POOL=20
   ```
2. Jalankan aplikasi:
   ```powershell
   go run main.go
   ```
3. Menjalankan pengujian otomatis:
   ```powershell
   # Pengujian Fungsional, Konkurensi 18+10, & NoSQL Injection
   go run tools/test_modul10/main.go

   # Pengujian Ketahanan Dua Database (Resilience)
   go run tools/test_resilience/main.go

   # Pengujian Unit Test
   go test -v ./...
   ```

---

## 8. Kesimpulan Akhir

Aplikasi telah memenuhi 100% persyaratan dari dokumen **MODUL 10 Praktikum Pemrograman Backend Lanjut: API dengan MongoDB**. Tidak ada bug, regresi, maupun kendala yang belum terselesaikan. Seluruh deliverables laporan teknis Markdown, PDF resmi `Tugas10_434241102.pdf`, dan bukti pengujian database telah tervalidasi secara riil.
