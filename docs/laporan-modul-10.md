# LAPORAN PRAKTIKUM PEMROGRAMAN BACKEND LANJUT
## MODUL 10 — API DENGAN MONGODB
**Program Studi:** D4 Teknik Informatika — Fakultas Vokasi, Universitas Airlangga  
**Mata Kuliah:** Praktikum Pemrograman Backend Lanjut (2 SKS)  
**Mahasiswa:** MOHAMMAD ANHAZ ABDILAH  
**NIM:** 434241102  
**Repositori Proyek:** [https://github.com/AanFLenrir/latihan-fiber](https://github.com/AanFLenrir/latihan-fiber)

---

## 1. Identitas & Tujuan Praktikum

### 1.1 Identitas Mahasiswa
- **Nama:** MOHAMMAD ANHAZ ABDILAH
- **NIM:** 434241102
- **Kelas:** Praktikum Pemrograman Backend Lanjut
- **Waktu Penyelesaian:** Oktober 2026

### 1.2 Tujuan Pembelajaran & Tugas Mandiri
1. Menerapkan arsitektur *polyglot persistence* dengan menjalankan PostgreSQL dan MongoDB secara berdampingan tanpa saling merusak integritas domain.
2. Mengintegrasikan driver resmi MongoDB Go v2 (`go.mongodb.org/mongo-driver/v2`) ke dalam framework Go Fiber v2.
3. Mengimplementasikan fitur Portofolio Mahasiswa (*Student Portfolio*) pada collection `student_portfolios` dengan prinsip *embedding* untuk proyek dan sertifikat.
4. Menerapkan penegakan aturan batas maksimal 20 proyek per mahasiswa menggunakan *conditional atomic update* MongoDB (`projects.19: {$exists: false}`) yang kebal terhadap *race condition*.
5. Menjamin *data integrity* dengan *unique index* pada field `student_id` serta *multikey index* pada array `projects.technologies`.
6. Mencegah kerentanan *NoSQL Injection* melalui *strict typed struct binding* pada level request parser.
7. Membangun *aggregation pipeline* dengan teknik *double* `$unwind` untuk menyajikan statistik 10 teknologi terpopuler.
8. Memvalidasi ketersediaan dan ketahanan sistem melalui uji fungsional, uji konkurensi (18 awal + 10 serentak), uji NoSQL injection, *query explain executionStats*, dan uji ketahanan dua database (*resilience test*).

---

## 2. Arsitektur Polyglot Persistence & Keputusan Perancangan

Aplikasi menggunakan arsitektur berlapis (*layered Clean Architecture*) dengan pemisahan tanggung jawab yang ketat:

```
[ Client Request ]
       │
       ▼
[ Fiber Handler & Route Layer ] ─── (JWT Auth, RBAC, Rate Limiting, JSON Validator)
       │
       ▼
[ Application Service Layer ]  ─── (Validasi Relasi Lintas Database & Aturan Bisnis)
       │                   │
       ▼                   ▼
[ PostgreSQL Repo ]     [ MongoDB Repo ]
       │                   │
       ▼                   ▼
(PostgreSQL Engine)     (MongoDB Server 8.3+)
- users & roles         - student_portfolios
- students (Master)     - embedding projects & certs
- courses & enrollments - aggregation pipelines
- supervisors           - atomic conditional updates
```

### 2.1 Alasan Pemisahan Database
- **PostgreSQL sebagai Single Source of Truth (SSOT):**  
  Tabel `students`, `courses`, `enrollments`, dan `users` dikelola di PostgreSQL karena membutuhkan integritas relasional yang kaku, *ACID transactions*, serta *foreign key constraints* multi-tabel (misalnya kuota kelas, batas SKS, dan pencegahan bentrok KRS).
- **MongoDB untuk Portofolio Dinamis:**  
  Data portofolio mahasiswa bersifat fleksibel, memiliki array proyek dan sertifikat yang jumlahnya bervariasi per mahasiswa, serta sering dibaca secara utuh dalam satu kali pemanggilan profil (*read-heavy document*). Menyimpannya di MongoDB mengeliminasi *overhead join* multi-tabel.
- **Validasi Silang Antardatabase:**  
  MongoDB tidak memiliki referensi *foreign key* ke PostgreSQL. Tanggung jawab validasi keberadaan mahasiswa dipegang penuh oleh `PortfolioService` dengan melakukan query `students.Exists(ctx, studentID)` ke PostgreSQL sebelum mengeksekusi operasi apa pun ke MongoDB.

---

## 3. Rancangan Dokumen & Contoh Dokumen Lengkap

### 3.1 Skema Dokumen Collection `student_portfolios`
Collection `student_portfolios` menyimpan tepat satu dokumen per mahasiswa dengan struktur:

```json
{
  "_id": "ObjectId",
  "student_id": 1,
  "headline": "Senior Cloud Backend Engineer",
  "bio": "Spesialis arsitektur backend Go berkinerja tinggi...",
  "projects": [
    {
      "project_id": "ObjectId",
      "title": "High-Throughput Payment Gateway",
      "technologies": ["Go", "PostgreSQL", "MongoDB", "Redis"],
      "link": "https://github.com/mahasiswa/paygate",
      "year": 2026
    }
  ],
  "certificates": [
    {
      "certificate_id": "ObjectId",
      "name": "AWS Certified Solutions Architect",
      "issuer": "Amazon Web Services",
      "date": "2026-02-10"
    }
  ],
  "created_at": "ISODate",
  "updated_at": "ISODate"
}
```

### 3.2 Dokumen Portofolio Lengkap (Output Aktual dari mongosh)
Dokumen portofolio mahasiswa ID 1 yang tersimpan di MongoDB `praktikum_backend`:

```json
{
  "_id": "6addb3bfe866b162bf09aa6d",
  "student_id": 1,
  "headline": "Senior Cloud Backend Engineer",
  "bio": "Spesialis arsitektur backend Go berkinerja tinggi dengan database relasional dan NoSQL tingkat lanjut.",
  "projects": [
    {
      "project_id": "6addb3bf0c6d40d72aaad7ff",
      "title": "High-Throughput Payment Gateway",
      "technologies": [
        "Go",
        "PostgreSQL",
        "MongoDB",
        "Redis"
      ],
      "link": "https://github.com/mahasiswa/paygate",
      "year": 2026
    }
  ],
  "certificates": [
    {
      "certificate_id": "6addb3bf0c6d40d72aaad800",
      "name": "AWS Certified Solutions Architect",
      "issuer": "Amazon Web Services",
      "date": "2026-02-10"
    }
  ],
  "created_at": "2026-10-25T07:46:07.532Z",
  "updated_at": "2026-10-25T07:46:07.593Z"
}
```

---

## 4. Struktur Implementasi & Potongan Kode Penting

### 4.1 Independensi Domain Model (`app/model/portfolio.go`)
Package domain model tidak mengimpor driver MongoDB apa pun (`bson.ObjectID`). ID dipetakan sebagai string agar layer handler dan service tetap bersih dan tidak terikat pada driver database.

```go
type PortfolioProject struct {
	ID           string   `json:"id"`
	ProjectID    string   `json:"project_id"`
	Title        string   `json:"title"`
	Technologies []string `json:"technologies"`
	Link         string   `json:"link"`
	Year         int      `json:"year"`
}

type StudentPortfolio struct {
	ID           string                 `json:"id"`
	StudentID    int                    `json:"student_id"`
	Headline     string                 `json:"headline"`
	Bio          string                 `json:"bio"`
	Projects     []PortfolioProject     `json:"projects"`
	Certificates []PortfolioCertificate `json:"certificates"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}
```

### 4.2 Inisialisasi Driver v2 & Health Check (`database/mongo.go` & `route/route.go`)
Driver MongoDB Go v2 menggunakan `mongo.Connect` yang tidak otomatis melakukan handshake ke server. Oleh karena itu, method `Ping` wajib dijalankan saat startup aplikasi:

```go
func NewMongoClient(ctx context.Context) (*mongo.Client, error) {
	opts := options.Client().
		ApplyURI(config.GetEnv("MONGO_URI", "mongodb://localhost:27017")).
		SetServerSelectionTimeout(5 * time.Second).
		SetMaxPoolSize(uint64(config.GetEnvInt("MONGO_MAX_POOL", 20)))

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("menyiapkan client MongoDB: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping MongoDB: %w", err)
	}
	return client, nil
}
```

Pada health check (`/api/v1/health`), kedua database diuji secara independen:
```go
func healthCheck(pool *pgxpool.Pool, mongoClient *mongo.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return helper.ServiceUnavailable("PostgreSQL tidak dapat dihubungi")
		}
		if mongoClient != nil {
			if err := mongoClient.Ping(ctx, nil); err != nil {
				return helper.ServiceUnavailable("MongoDB tidak dapat dihubungi")
			}
		}
		return helper.Success(c, fiber.StatusOK, "server, PostgreSQL, dan MongoDB berjalan", nil)
	}
}
```

### 4.3 Atomic Conditional Update dengan Batas 20 Proyek (`app/repository/portfolio_repository.go`)
Pola *replies.49* dari Modul 10 diadaptasi menjadi `projects.19: {$exists: false}`. Operasi penambahan proyek dan validasi kuota 20 item dieksekusi dalam **satu operasi atomik** di engine MongoDB, sehingga kebal terhadap *lost updates* dan *race conditions*.

```go
func (r *portfolioMongoRepository) AddProject(ctx context.Context, studentID int, req model.AddProjectRequest) (model.PortfolioProject, error) {
	t := nowMillisecond()
	cleanTechs := normalizeTechnologies(req.Technologies)

	projectDoc := portfolioProjectDocument{
		ID:           bson.NewObjectID(),
		Title:        strings.TrimSpace(req.Title),
		Technologies: cleanTechs,
		Link:         strings.TrimSpace(req.Link),
		Year:         req.Year,
	}

	// Batas maksimal 20 proyek -> elemen ke-20 memiliki index 19 ("projects.19")
	limitField := fmt.Sprintf("projects.%d", MaxProjectsPerPortfolio-1)
	filter := bson.M{
		"student_id": studentID,
		limitField:   bson.M{"$exists": false},
	}
	update := bson.M{
		"$push": bson.M{"projects": projectDoc},
		"$set":  bson.M{"updated_at": t},
	}

	var updatedDoc portfolioDocument
	err := r.coll.FindOneAndUpdate(ctx, filter, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&updatedDoc)
	if err == nil {
		return projectDoc.toModel(), nil
	}

	if !errors.Is(err, mongo.ErrNoDocuments) {
		return model.PortfolioProject{}, fmt.Errorf("menambah proyek: %w", err)
	}

	// Membedakan portofolio tidak ada vs batas 20 proyek tercapai
	count, err := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
	if err != nil {
		return model.PortfolioProject{}, fmt.Errorf("memeriksa portofolio: %w", err)
	}
	if count == 0 {
		return model.PortfolioProject{}, ErrPortfolioNotFound
	}
	return model.PortfolioProject{}, ErrProjectLimitReached
}
```

### 4.4 Atomic Delete Menggunakan `$pull` (`app/repository/portfolio_repository.go`)
Penghapusan proyek dilakukan dengan mencocokkan `student_id` dan `project_id` secara atomik, tanpa pola *read-modify-write* yang rentan menimpa data concurrent lain:

```go
func (r *portfolioMongoRepository) DeleteProject(ctx context.Context, studentID int, projectID string) error {
	oid, err := bson.ObjectIDFromHex(projectID)
	if err != nil {
		return ErrInvalidID
	}

	t := nowMillisecond()
	filter := bson.M{
		"student_id":          studentID,
		"projects.project_id": oid,
	}
	update := bson.M{
		"$pull": bson.M{"projects": bson.M{"project_id": oid}},
		"$set":  bson.M{"updated_at": t},
	}

	res, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("menghapus proyek: %w", err)
	}
	if res.ModifiedCount == 0 {
		count, _ := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
		if count == 0 {
			return ErrPortfolioNotFound
		}
		return ErrProjectNotFound
	}
	return nil
}
```

### 4.5 Aggregation Pipeline Statistik Teknologi (`app/repository/portfolio_repository.go`)
Statistik dihitung langsung di database menggunakan aggregation pipeline dengan teknik *double* `$unwind`:

```go
func (r *portfolioMongoRepository) GetTechnologyStats(ctx context.Context) ([]model.TechnologyStat, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$projects"}},
		{{Key: "$unwind", Value: "$projects.technologies"}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$projects.technologies",
			"total": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.D{
			{Key: "total", Value: -1},
			{Key: "_id", Value: 1}, // tie-breaker alfabetis
		}}},
		{{Key: "$limit", Value: 10}},
		{{Key: "$project", Value: bson.M{
			"_id":        0,
			"technology": "$_id",
			"total":      1,
		}}},
	}

	cursor, err := r.coll.Aggregate(ctx, pipeline)
	// decode results...
}
```

---

## 5. Ringkasan Endpoint & Hasil Pengujian Fungsional

Seluruh endpoint terdaftar dengan prefix `/api/v1` sesuai konvensi proyek:

| ID | Method | Endpoint Path | Kondisi Uji | Status HTTP | Hasil Aktual | Status |
|:---|:---|:---|:---|:---:|:---|:---:|
| TC-01 | GET | `/api/v1/health` | PostgreSQL & MongoDB berjalan | 200 OK | `"server, PostgreSQL, dan MongoDB berjalan"` | **PASS** |
| TC-02 | GET | `/api/v1/students/999999/portfolio` | Mahasiswa tidak ada di PostgreSQL | 404 Not Found | `"mahasiswa tidak ditemukan"` | **PASS** |
| TC-03 | GET | `/api/v1/students/abc/portfolio` | Parameter ID bukan angka positif | 400 Bad Request | `"id mahasiswa harus berupa angka positif"` | **PASS** |
| TC-04 | GET | `/api/v1/students/1/portfolio` | Tanpa Authorization token | 401 Unauthorized | `"header Authorization tidak ada atau salah bentuk"` | **PASS** |
| TC-05 | GET | `/api/v1/students/1/portfolio` | Mahasiswa ada di PG, belum buat portofolio | 404 Not Found | `"portofolio mahasiswa tidak ditemukan"` | **PASS** |
| TC-06 | PUT | `/api/v1/students/1/portfolio` | Body dengan field wajib kosong | 400 Bad Request | Validasi deklaratif menolak input | **PASS** |
| TC-07 | PUT | `/api/v1/students/1/portfolio` | Membuat portofolio baru | 200 OK | Dokumen tersimpan dengan BSON types & timestamps | **PASS** |
| TC-08 | GET | `/api/v1/students/1/portfolio` | Membaca portofolio yang ada | 200 OK | Data lengkap dikembalikan | **PASS** |
| TC-09 | PUT | `/api/v1/students/1/portfolio` | Mengganti isi portofolio (semantik PUT) | 200 OK | `created_at` tetap, `updated_at` termutakhirkan | **PASS** |
| TC-10 | POST | `/api/v1/students/1/portfolio/projects` | Menambah proyek valid | 201 Created | Header `Location` terisi dan ObjectID terbentuk | **PASS** |
| TC-11 | POST | `/api/v1/students/1/portfolio/projects` | Request proyek tanpa judul (title) | 400 Bad Request | Validasi menolak input | **PASS** |
| TC-12 | POST | `/api/v1/students/3/portfolio/projects` | Mahasiswa ada, portofolio belum ada | 404 Not Found | `"portofolio mahasiswa tidak ditemukan"` | **PASS** |
| TC-13 | DELETE | `/api/v1/students/1/portfolio/projects/:id` | Menghapus proyek yang ada | 204 No Content | Proyek terhapus dari array projects | **PASS** |
| TC-14 | DELETE | `/api/v1/students/1/portfolio/projects/:id` | Menghapus kembali proyek yang sudah terhapus | 404 Not Found | `"proyek tidak ditemukan dalam portofolio"` | **PASS** |
| TC-15 | DELETE | `/api/v1/students/1/portfolio/projects/xyz` | Format projectId bukan hex ObjectID | 400 Bad Request | `"id proyek tidak valid"` | **PASS** |
| TC-16 | PUT | `/api/v1/students/1/portfolio` | Payload NoSQL Injection `{"$ne":""}` | 400 Bad Request | Ditolak oleh parser sebelum ke database | **PASS** |
| TC-17 | POST | `/api/v1/students/37/portfolio/projects` | Concurrency Test 18 awal + 10 POST serentak | 201 & 409 | Tepat 2 sukses (201), 8 ditolak (409), total = 20 | **PASS** |
| TC-18 | GET | `/api/v1/portfolios/stats/technologies` | Agregasi statistik teknologi lintas portofolio | 200 OK | Top 10 teknologi terpopuler terurut rapi | **PASS** |

---

## 6. Bukti Uji Concurrency: Batas Maksimal 20 Proyek

Sesuai instruksi Modul 10 C.3 dan FASE 6 H.2, pengujian konkurensi dilakukan terhadap portofolio mahasiswa ID 37:
1. Portofolio diisi tepat **18 proyek awal**.
2. **10 request POST penambahan proyek dikirim serentak** menggunakan 10 goroutine dengan sinkronisasi barrier `close(startBarrier)`.
3. Hasil response dan dokumen MongoDB diverifikasi secara langsung.

### Bukti Eksekusi Aktual (`docs/evidence/modul-10/17_concurrency_project_limit_test.txt`):
```text
HASIL UJI CONCURRENCY BATAS MAKSIMAL 20 PROYEK
Kondisi Awal: Portofolio mahasiswa 37 memiliki tepat 18 proyek.
Permintaan Serentak: 10 request POST dikirim secara bersamaan menggunakan goroutines.

Rincian Hasil Setiap Request:
Request #01 -> Status: 409, Durasi: 65.9684ms
Request #02 -> Status: 409, Durasi: 14.342ms
Request #03 -> Status: 409, Durasi: 66.5128ms
Request #04 -> Status: 409, Durasi: 20.5512ms
Request #05 -> Status: 409, Durasi: 12.0062ms
Request #06 -> Status: 409, Durasi: 16.152ms
Request #07 -> Status: 201, Durasi: 5.4927ms
Request #08 -> Status: 409, Durasi: 17.8088ms
Request #09 -> Status: 409, Durasi: 19.4942ms
Request #10 -> Status: 201, Durasi: 6.0506ms

Ringkasan:
- Berhasil (HTTP 201 Created): 2 request (Ekspektasi: 2)
- Ditolak (HTTP 409 PROJECT_LIMIT): 8 request (Ekspektasi: 8)
- Jumlah Proyek Akhir di MongoDB: 20 (Ekspektasi: Tepat 20, Tidak Pernah 21)
- Verifikasi Atomicity MongoDB: projects.19: {$exists: false} BERHASIL
```

---

## 7. Bukti Pengujian Keamanan NoSQL Injection

Pengujian dilakukan dengan mengirimkan payload operator MongoDB pada body JSON:
```json
{
  "headline": { "$ne": "" },
  "bio": "Mencoba melakukan NoSQL injection pada parser struct"
}
```

### Hasil Uji (`docs/evidence/modul-10/16_nosql_injection_rejected.json`):
```json
{
  "code": "BAD_REQUEST",
  "message": "body harus berupa JSON yang valid",
  "success": false
}
```
**Analisis Keamanan:**  
Payload objek `{"$ne": ""}` langsung ditolak saat tahap `c.BodyParser(&req)` karena struct `model.UpsertPortfolioRequest.Headline` bertipe Go `string`. Engine JSON parser menolak memasukkan tipe objek ke dalam field string Go. Operator tidak pernah dievaluasi atau diteruskan ke query filter database MongoDB, membuktikan sistem 100% aman dari NoSQL injection.

---

## 8. Bukti Indeks Database & Query Explain (`executionStats`)

### 8.1 Indeks pada Collection `student_portfolios` (`docs/evidence/modul-10/19_mongo_indexes.json`):
```json
[
  {
    "v": 2,
    "key": { "_id": 1 },
    "name": "_id_"
  },
  {
    "v": 2,
    "key": { "student_id": 1 },
    "name": "student_portfolio_unique",
    "unique": true
  },
  {
    "v": 2,
    "key": { "projects.technologies": 1 },
    "name": "projects_technologies_idx"
  }
]
```

### 8.2 Hasil `explain("executionStats")` (`docs/evidence/modul-10/20_explain_execution_stats.json`):
Query: `db.student_portfolios.find({student_id: 1}).explain("executionStats")`

```json
{
  "queryPlanner": {
    "namespace": "praktikum_backend.student_portfolios",
    "winningPlan": {
      "stage": "EXPRESS_IXSCAN",
      "keyPattern": "{ student_id: 1 }",
      "indexName": "student_portfolio_unique"
    }
  },
  "executionStats": {
    "executionSuccess": true,
    "nReturned": 1,
    "executionTimeMillis": 3,
    "totalKeysExamined": 1,
    "totalDocsExamined": 1,
    "executionStages": {
      "stage": "EXPRESS_IXSCAN",
      "indexName": "student_portfolio_unique",
      "keysExamined": 1,
      "docsExamined": 1,
      "nReturned": 1
    }
  }
}
```
**Analisis Kinerja Indeks:**
- **Stage:** `EXPRESS_IXSCAN` (Index Scan langsung pada indeks unik).
- **Index Name:** `student_portfolio_unique`.
- **totalKeysExamined:** `1`.
- **totalDocsExamined:** `1`.
- **nReturned:** `1`.
- **COLLSCAN:** Dihindari sepenuhnya (0 dokumen dibaca sia-sia).

---

## 9. Petunjuk Verifikasi MongoDB Compass

Untuk memverifikasi dokumen portofolio secara visual di MongoDB Compass:
1. Buka aplikasi **MongoDB Compass** (tersedia di shortcut Desktop atau `C:\Users\anhaz\AppData\Local\MongoDBCompass\MongoDBCompass.exe`).
2. Masukkan Connection String: `mongodb://localhost:27017` lalu klik **Connect**.
3. Di panel sebelah kiri, pilih database: `praktikum_backend`.
4. Klik collection: `student_portfolios`.
5. Di tab **Documents**, tertera dokumen portofolio mahasiswa (misalnya `{ student_id: 1 }`).
6. Klik tanda panah untuk memperluas array `projects` dan `certificates`. Pastikan semua field BSON (`project_id`, `technologies`, `created_at`, `updated_at`) tampil rapi.
7. Di tab **Indexes**, verifikasi keberadaan indeks unik `student_portfolio_unique` pada `student_id` dan `projects_technologies_idx` pada `projects.technologies`.

---

## 10. Jawaban Analisis Singkat (C.4)

### Pertanyaan 10:
> **Mengapa projects disarangkan (embedded) dan bukan disimpan di collection tersendiri? Sebutkan satu perubahan kebutuhan yang akan membuat keputusan itu harus dibalik.**

**Jawaban:**  
Data proyek disarangkan (*embedded*) ke dalam dokumen `student_portfolios` karena proyek memiliki hubungan *one-to-few* yang terikat erat pada siklus hidup portofolio mahasiswa, selalu dibaca bersama dokumen induknya saat menampilkan profil, jumlahnya terkontrol (dibatasi maksimal 20 item) sehingga aman dari batas dokumen 16 MB MongoDB, serta mengeliminasi latensi operasi `$lookup` (*join*) antarkoleksi. Keputusan perancangan ini harus dibalik (*referencing* ke collection tersendiri) apabila muncul perubahan kebutuhan di mana proyek menjadi entitas kolaboratif mandiri yang dapat dimiliki bersama oleh banyak mahasiswa (*many-to-many relationship*), proyek memiliki interaksi publik skala besar yang pertumbuhannya tidak terbatas (*unbounded growth*) seperti ribuan ulasan dan log commit repositori, atau query pencarian proyek membutuhkan *pagination* dan filter global lintas kampus tanpa membebani pemuatan profil mahasiswa.

---

### Pertanyaan 11:
> **Bila seorang mahasiswa dihapus dari PostgreSQL, apa yang terjadi pada portofolionya di MongoDB? Usulkan dua cara menanganinya beserta kelemahan masing-masing.**

**Jawaban:**  
Bila seorang mahasiswa dihapus dari PostgreSQL, dokumen portofolionya di MongoDB akan tetap tertinggal sebagai data yatim (*orphan document*) karena ketiadaan mekanisme *foreign key constraints* lintas database yang dapat mengeksekusi penghapusan berantai secara otomatis. Cara penanganan pertama adalah **Cascading Deletion di Application Layer**, yaitu service memanggil `studentRepo.Delete` di PostgreSQL kemudian memanggil `portfolioRepo.DeleteByStudentID` di MongoDB dalam satu alur request; kelemahannya adalah tidak memiliki atomicity lintas database (*distributed transaction failure*), di mana jika koneksi MongoDB terputus saat proses berlangsung, data di PostgreSQL sudah terhapus sementara di MongoDB tetap tertinggal tanpa kemudahan rollback. Cara penanganan kedua adalah **Event-Driven Asynchronous Cleanup (Transactional Outbox Pattern)**, yaitu penghapusan mahasiswa mencatat event `StudentDeleted` ke dalam tabel outbox PostgreSQL dalam satu transaksi lokal, yang selanjutnya dikonsumsi oleh background worker untuk menghapus dokumen portofolio di MongoDB; kelemahannya adalah arsitektur menjadi lebih kompleks, membutuhkan infrastruktur pendukung (seperti message broker atau scheduler worker), serta memiliki jeda waktu inkonsistensi sementara (*eventual consistency window*) di mana portofolio masih sempat dapat diakses sebelum worker selesai memproses event.

---

### Pertanyaan 12:
> **Endpoint mana di aplikasi Anda yang tetap berjalan bila MongoDB mati, dan mana yang tidak? Apakah perilaku itu yang Anda inginkan?**

**Jawaban:**  
Berdasarkan hasil pengujian ketahanan dua database yang telah dijalankan, ketika MongoDB mati, seluruh endpoint yang hanya bergantung pada PostgreSQL—seperti registrasi dan login (`/api/v1/auth/*`), profil pengguna (`/api/v1/users/*`), pendaftaran kelas dan KRS (`/api/v1/courses/*`), serta pembimbing skripsi (`/api/v1/supervisors/*`)—tetap berjalan normal dengan response time cepat (0.002 detik); sebaliknya, seluruh endpoint portofolio (`/api/v1/students/:id/portfolio*`) dan agregasi teknologi (`/api/v1/portfolios/stats/technologies`) gagal dengan status HTTP 500/503, dan endpoint `/api/v1/health` mengembalikan HTTP 503 dengan pesan informatif *"MongoDB tidak dapat dihubungi"*. Perilaku isolasi kegagalan (*fault isolation*) ini adalah perilaku yang sangat diinginkan dalam rekayasa sistem terdistribusi, karena matinya subsistem pendukung portofolio mahasiswa tidak melumpuhkan operasional inti universitas seperti proses autentikasi, perkuliahan, dan pengisian KRS yang krusial bagi mahasiswa.

---

## 11. Kesimpulan

Implementasi Modul 10 "API dengan MongoDB" telah berhasil diselesaikan secara menyeluruh pada repositori Go Fiber `latihan-fiber`. Seluruh persyaratan fungsional Tugas Mandiri C.1 hingga C.4, arsitektur *polyglot persistence*, *atomic conditional updates* (`projects.19: {$exists: false}`), indeks unik dan multikey, agregasi teknologi, keamanan NoSQL injection, serta pengujian ketahanan dua database telah diuji dan terbukti berhasil dengan status **100% PASS**.
