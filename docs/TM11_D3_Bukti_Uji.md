# BUKTI UJI DAN PERCOBAAN PEMBANDING (TM 11)
**Mata Kuliah:** Praktikum Pemrograman Backend Lanjut (Pertemuan 11: Background Jobs & Asynchronous Processing)  
**Mahasiswa:** MOHAMMAD ANHAZ ABDILAH (NIM: 434241102)  
**Lingkungan Uji:** Windows 11 64-bit | PowerShell | Go 1.24 | PostgreSQL 18 | Memurai v4.1.2 | Asynq v0.26.0

---

## 1. Pengujian C.2 — Redis Mati, KRS Tetap Berjalan

### Tujuan
Membuktikan bahwa pemisahan arsitektur melalui transactional outbox membuat API pendaftaran KRS tidak bergantung pada ketersediaan broker (Redis). Bila Redis mati, operasi KRS tetap berhasil dengan latensi rendah (orde milidetik), data tersimpan aman di PostgreSQL, dan notifikasi tertunda secara otomatis di-relay ketika Redis menyala kembali tanpa ada data yang hilang.

### Kondisi Awal
* PostgreSQL berjalan normal.
* Server API dan Worker berjalan.
* Riwayat outbox dan notifikasi akun uji telah di-reset via `reset11.sql`.
* Redis (Memurai) dimatikan secara sengaja.

### Langkah Pelaksanaan & Perintah
1. Mematikan service Redis:
   ```powershell
   & "C:\Users\anhaz\memurai\Memurai\memurai-cli.exe" shutdown
   ```
2. Mahasiswa `mhsuji` mendaftar ke kelas 5 (`IF303`), 6 (`IF304`), dan 7 (`IF305`):
   ```powershell
   POST http://localhost:3000/api/v1/courses/5/enroll
   POST http://localhost:3000/api/v1/courses/6/enroll
   POST http://localhost:3000/api/v1/courses/7/enroll
   ```
3. Memeriksa antrean admin `/api/v1/admin/queue`.
4. Menyalakan kembali Redis:
   ```powershell
   Start-Process -FilePath "C:\Users\anhaz\memurai\Memurai\memurai.exe" -ArgumentList "--port 6379"
   ```
5. Menunggu relay memproses outbox dan memeriksa kembali endpoint admin dan notifikasi mahasiswa.

### Response API Aktual
* **Pendaftaran saat Redis mati:**
  ```json
  POST /courses/5/enroll -> Status: 201 Created (Durasi: 17.81 ms)
  {"success":true,"message":"berhasil mendaftar kelas","data":{"id":5,"code":"IF303","name":"Komputasi Awan","credits":3,"quota":40,"enrolled_count":1,"version":1}}

  POST /courses/6/enroll -> Status: 201 Created (Durasi: 2.83 ms)
  {"success":true,"message":"berhasil mendaftar kelas","data":{"id":6,"code":"IF304","name":"Data Mining","credits":3,"quota":40,"enrolled_count":1,"version":1}}

  POST /courses/7/enroll -> Status: 201 Created (Durasi: 4.34 ms)
  {"success":true,"message":"berhasil mendaftar kelas","data":{"id":7,"code":"IF305","name":"Pemrograman Mobile","credits":3,"quota":40,"enrolled_count":1,"version":1}}
  ```
* **Endpoint Admin `/admin/queue` saat Redis mati:**
  ```json
  HTTP 200 OK
  {
    "success": true,
    "message": "keadaan queue berhasil diambil",
    "data": {
      "outbox_pending": 3,
      "outbox_published": 4,
      "oldest_pending_age": "5.029s",
      "notifications_sent": 4,
      "queue": null,
      "queue_error": "queue tidak dapat dibaca: Redis tidak dapat dihubungi atau queue belum pernah dipakai"
    }
  }
  ```
* **Endpoint Admin `/admin/queue` setelah Redis hidup kembali:**
  ```json
  HTTP 200 OK
  {
    "success": true,
    "message": "keadaan queue berhasil diambil",
    "data": {
      "outbox_pending": 0,
      "outbox_published": 7,
      "oldest_pending_age": "0s",
      "notifications_sent": 7,
      "queue": {
        "name": "notifications",
        "pending": 0,
        "active": 0,
        "scheduled": 0,
        "retry": 0,
        "archived": 0,
        "completed": 7,
        "processed_today": 7,
        "failed_today": 0
      }
    }
  }
  ```

### Log Terkait
* **Log Worker saat Redis mati:**
  ```json
  {"time":"2026-11-01T18:50:50+07:00","level":"WARN","msg":"relay_delayed","published":0,"error":"enqueue outbox 6: UNKNOWN: redis eval error: dial tcp [::1]:6379: connectex: No connection could be made because the target machine actively refused it."}
  ```
* **Log Worker setelah Redis menyala kembali:**
  ```json
  {"time":"2026-11-01T18:51:02+07:00","level":"INFO","msg":"relay_done","published":3}
  {"time":"2026-11-01T18:51:04+07:00","level":"INFO","msg":"notification_processed","outbox_id":7,"event":"enrollment.created","retry":0,"already_sent":false}
  {"time":"2026-11-01T18:51:04+07:00","level":"INFO","msg":"notification_processed","outbox_id":8,"event":"enrollment.created","retry":0,"already_sent":false}
  {"time":"2026-11-01T18:51:04+07:00","level":"INFO","msg":"notification_processed","outbox_id":6,"event":"enrollment.created","retry":0,"already_sent":false}
  ```

### Kesimpulan C.2
API tetap responsif dalam hitungan milidetik saat broker down, tidak terjadi error 500 ke pengguna, dan seluruh task yang tertahan segera diproses begitu broker pulih.

---

## 2. Pengujian C.3 — Temporary Failure & Retry Backoff

### Tujuan
Menguji ketahanan worker menghadapi kegagalan jaringan/server email yang bersifat sementara (`ErrTemporaryDelivery`). Sistem harus melakukan retry otomatis dengan backoff terukur tanpa kehilangan task.

### Kondisi Awal
* `.env`: `MAILER_FAIL_EVERY=3`.
* Database di-reset menggunakan `reset11.sql`.
* `logs/mail.log` dikosongkan.

### Langkah Pelaksanaan & Perintah
1. Menjalankan worker dengan konfigurasi `MAILER_FAIL_EVERY=3`.
2. Mahasiswa `mhsuji` mendaftar ke 6 kelas secara berurutan (kelas 1 s.d. 6):
   ```powershell
   1..6 | ForEach-Object {
       Invoke-RestMethod -Method Post -Uri "http://localhost:3000/api/v1/courses/$_/enroll" -Headers $headers
   }
   ```
3. Menunggu 20 detik untuk siklus retry Asynq.
4. Memeriksa konsistensi tiga angka: jumlah row `notifications`, jumlah di endpoint `/notifications/mine`, dan baris di `logs/mail.log`.

### Hasil Aktual & Bukti Pengujian
* **Output Uji:**
  ```text
  === MENJALANKAN UJI C.3 (Temporary Failure) ===
  POST /courses/1/enroll -> Status: 201
  POST /courses/2/enroll -> Status: 201
  POST /courses/3/enroll -> Status: 201
  POST /courses/4/enroll -> Status: 201
  POST /courses/5/enroll -> Status: 201
  POST /courses/6/enroll -> Status: 201

  Menunggu 20 detik untuk retry Asynq dan pemrosesan...

  HASIL C.3:
    1. Database notifications: 6
    2. GET /notifications/mine: 6 (HTTP 200)
    3. logs/mail.log untuk mhsuji@student.unair.ac.id: 6
    -> Ketiganya sama-sama 6: true
  ```
* **Log Worker (Bukti Terjadinya Kegagalan Sementara dan Retry Berhasil):**
  ```json
  {"time":"2026-11-01T18:54:10+07:00","level":"WARN","msg":"task_failed","type":"notification:send","attempt":"1/6","error":"pengiriman gagal sementara: server email sibuk (attempt ke-6)"}
  {"time":"2026-11-01T18:54:10+07:00","level":"WARN","msg":"task_failed","type":"notification:send","attempt":"1/6","error":"pengiriman gagal sementara: server email sibuk (attempt ke-9)"}
  {"time":"2026-11-01T18:54:16+07:00","level":"WARN","msg":"task_failed","type":"notification:send","attempt":"2/6","error":"pengiriman gagal sementara: server email sibuk (attempt ke-12)"}
  {"time":"2026-11-01T18:54:16+07:00","level":"INFO","msg":"notification_processed","outbox_id":17,"event":"enrollment.created","retry":1,"already_sent":false}
  {"time":"2026-11-01T18:54:27+07:00","level":"INFO","msg":"notification_processed","outbox_id":19,"event":"enrollment.created","retry":2,"already_sent":false}
  ```

### Kesimpulan C.3
Tiga angka verifikasi bernilai sama-sama 6. Log membuktikan minimal dua kegagalan sementara terjadi, Asynq mengeksekusi retry sesuai jeda backoff, dan seluruh notifikasi akhirnya terkirim secara sempurna.

---

## 3. Pengujian C.4 — Permanent Failure & SkipRetry

### Tujuan
Menguji perilaku worker saat menemui kegagalan permanen (`ErrPermanentDelivery` / domain email RFC 2606 `.invalid`). Task harus langsung dihentikan pada attempt pertama dan dipindahkan ke `archived` tanpa di-retry berulang kali.

### Kondisi Awal
* `.env`: `MAILER_FAIL_EVERY=0`.
* User `salahalamat` dengan email `salah@kampus.invalid`.

### Langkah Pelaksanaan & Perintah
1. Mencatat nilai awal `queue.archived` di `GET /api/v1/admin/queue`.
2. Mendaftarkan user `salahalamat` ke kelas 2 (`IF102`):
   ```powershell
   POST http://localhost:3000/api/v1/courses/2/enroll
   ```
3. Menunggu 15 detik dan memeriksa kembali status antrean admin dan log worker.

### Hasil Aktual & Bukti Pengujian
* **Output Uji:**
  ```text
  === MENJALANKAN UJI C.4 (Permanent Failure) ===
  Kondisi awal /admin/queue: archived: 0
  POST /courses/2/enroll sebagai salahalamat -> Status: 201 Created
  Menunggu 15 detik untuk SkipRetry / archiving...
  Kondisi akhir /admin/queue: archived: 1, retry: 0
  ```
* **Log Worker:**
  ```json
  {"time":"2026-11-01T18:55:42+07:00","level":"INFO","msg":"relay_done","published":1}
  {"time":"2026-11-01T18:55:45+07:00","level":"WARN","msg":"task_failed","type":"notification:send","attempt":"1/6","error":"skip retry for the task: pengiriman gagal permanen: alamat salah@kampus.invalid tidak ditemukan"}
  {"time":"2026-11-01T18:55:45+07:00","level":"WARN","msg":"Retry exhausted for task id=outbox-21","component":"asynq"}
  ```

### Kesimpulan C.4
Task gagal permanen dihentikan tepat pada attempt 1/6 karena membungkus `asynq.SkipRetry`. Nilai `archived` bertambah tepat 1, dan `retry` bernilai 0. KRS mahasiswa tetap sukses 201 tanpa terpengaruh kegagalan email.

---

## 4. Pengujian C.5 — Idempotency dengan Enam Salinan Task

### Tujuan
Membuktikan bahwa handler worker idempoten saat menerima multiple duplicate task (pengiriman task ganda akibat kegagalan jaringan atau re-delivery).

### Kondisi Awal
* Worker dimatikan sementara.
* Mahasiswa `mhsuji` mendaftar ke kelas 8 (`IF306`).
* Event tertahan di tabel `outbox` dengan `published_at IS NULL` (ID=22).

### Langkah Pelaksanaan & Perintah
1. Menginjeksi 5 salinan task langsung ke Redis menggunakan `tools/redeliver`:
   ```powershell
   .\redeliver.exe -outbox 22 -copies 5
   ```
2. Menyalakan worker. Relay menambahkan 1 task orisinal, sehingga total ada 6 salinan task di antrean.
3. Menunggu 8 detik dan memeriksa database, log email, serta log worker.

### Hasil Aktual & Bukti Pengujian
* **Output Injeksi Redeliver:**
  ```text
  salinan 1/5 ter-enqueue (task_id: a2b693ae-2d79-4cee-a8a0-5cabe42f3113)
  salinan 2/5 ter-enqueue (task_id: 0f8c28c6-ac07-4dfe-ad10-a93ec37c86e4)
  salinan 3/5 ter-enqueue (task_id: f227c4ab-879e-47de-8723-15591c0dc315)
  salinan 4/5 ter-enqueue (task_id: ce5f1e27-799c-45be-b192-be61d85a9165)
  salinan 5/5 ter-enqueue (task_id: 853c6034-defa-4302-bdea-1db3dec2671c)
  berhasil menambahkan 5 salinan task untuk outbox 22 ke queue notifications
  ```
* **Output Verifikasi Database dan Log:**
  ```text
  Notifikasi di DB untuk outbox_id 22: 1 (Diharapkan: 1)
  Baris mail.log untuk kelas IF306: 1 (Diharapkan: 1)
  ```
* **Log Worker (Tepat 6 Baris Pemrosesan):**
  ```json
  {"time":"2026-11-01T18:57:30+07:00","level":"INFO","msg":"notification_processed","outbox_id":22,"event":"enrollment.created","retry":0,"already_sent":false}
  {"time":"2026-11-01T18:57:30+07:00","level":"INFO","msg":"notification_processed","outbox_id":22,"event":"enrollment.created","retry":0,"already_sent":true}
  {"time":"2026-11-01T18:57:30+07:00","level":"INFO","msg":"notification_processed","outbox_id":22,"event":"enrollment.created","retry":0,"already_sent":true}
  {"time":"2026-11-01T18:57:30+07:00","level":"INFO","msg":"notification_processed","outbox_id":22,"event":"enrollment.created","retry":0,"already_sent":true}
  {"time":"2026-11-01T18:57:30+07:00","level":"INFO","msg":"notification_processed","outbox_id":22,"event":"enrollment.created","retry":0,"already_sent":true}
  {"time":"2026-11-01T18:57:30+07:00","level":"INFO","msg":"notification_processed","outbox_id":22,"event":"enrollment.created","retry":0,"already_sent":true}
  ```

### Kesimpulan C.5
Mekanisme claim berbasis database `UNIQUE (outbox_id)` dan `ON CONFLICT DO NOTHING` berhasil mencegah duplikasi. Dari 6 task konkuren, tepat 1 task yang mengirim email (`already_sent: false`) dan 5 task lainnya langsung di-skip (`already_sent: true`).

---

## 5. Percobaan Pembanding Terkontrol (Controlled Counter-Experiment)

Sesuai instruksi Bagian D.3, dilakukan eksperimen pembanding dengan mengembalikan satu kesalahan yang lolos compiler untuk membuktikan bahwa alat uji mampu mendeteksi kegagalan secara sahih.

### Kesalahan yang Dipilih
**Bug 6:** Kesalahan pembungkusan format error `asynq.SkipRetry` pada `app/service/notification_worker.go` (`HandleSend`).

### Skenario Eksperimen
1. **Kondisi Kode Salah (Bug Dikembalikan):**
   ```go
   // notification_worker.go
   if errors.Is(err, helper.ErrPermanentDelivery) {
       return fmt.Errorf("%v: %w", asynq.SkipRetry, err) // Verb %v membuat SkipRetry tidak di-wrap
   }
   ```
2. **Kompilasi & Uji:**
   `go build ./...` berhasil (membuktikan bug ini lolos compiler tanpa error).
3. **Eksekusi Uji C.4 saat Bug Aktif:**
   * User `salahalamat` mendaftar ke kelas 2.
   * **Hasil Pengamatan (GAGAL):**
     Nilai antrean: `queue.retry: 1`, `queue.archived: 0`.
     Task tidak masuk ke `archived`, melainkan diulang oleh Asynq:
     ```json
     {"level":"WARN","msg":"task_failed","attempt":"1/6","error":"skip retry for the task: pengiriman gagal permanen..."}
     {"level":"WARN","msg":"task_failed","attempt":"2/6","error":"skip retry for the task: pengiriman gagal permanen..."}
     {"level":"WARN","msg":"task_failed","attempt":"3/6","error":"skip retry for the task: pengiriman gagal permanen..."}
     ```
     **Kesimpulan:** Uji C.4 berhasil mengungkap kegagalan sistemik bahwa task permanen terus di-retry.

4. **Kondisi Kode Benar (Perbaikan Diterapkan):**
   ```go
   if errors.Is(err, helper.ErrPermanentDelivery) {
       return fmt.Errorf("%w: %v", asynq.SkipRetry, err) // Verb %w membungkus SkipRetry dengan benar
   }
   ```
5. **Eksekusi Uji C.4 setelah Perbaikan:**
   * **Hasil Pengamatan (SUKSES):**
     Nilai antrean: `queue.retry: 0`, `queue.archived: 1`.
     Log worker membuktikan penghentian langsung:
     ```json
     {"level":"WARN","msg":"task_failed","attempt":"1/6","error":"skip retry for the task: pengiriman gagal permanen..."}
     {"level":"WARN","msg":"Retry exhausted for task id=outbox-21","component":"asynq"}
     ```
     Task langsung masuk archived tanpa retry attempt kedua (`attempt 1/6`).
