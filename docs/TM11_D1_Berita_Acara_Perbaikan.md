# BERITA ACARA PERBAIKAN SEMBILAN KESALAHAN (TM 11)
**Mata Kuliah:** Praktikum Pemrograman Backend Lanjut (Pertemuan 11: Background Jobs & Asynchronous Processing)  
**Mahasiswa:** MOHAMMAD ANHAZ ABDILAH (NIM: 434241102)  
**Lingkungan:** Windows 11 64-bit | Go 1.24 | Fiber v2.52 | PostgreSQL 18 | Redis (Memurai v4.1.2) | Asynq v0.26.0

---

## Ringkasan Eksekutif
Berdasarkan modul praktikum TM 11 Edisi Puzzle, terdapat **9 (sembilan) kesalahan** yang sengaja ditanamkan dalam kode sumber modul. Kesalahan-kesalahan tersebut ditemukan dan diverifikasi secara empiris melalui audit compiler, analisis statis (`go vet`), pengujian kegagalan terkontrol (Bagian C.1–C.6), dan pengujian fungsional database.

Berikut adalah tabel berita acara perbaikan lengkap beserta bukti sebelum dan sesudah perbaikan.

---

## Tabel Berita Acara Perbaikan 9 Kesalahan

| No. | Lokasi (file, fungsi) | Gejala yang Diamati | Uji yang Mengungkapnya | Akar Masalah | Perbaikan yang Dilakukan | Bukti Verifikasi |
|---|---|---|---|---|---|---|
| **1** | `helper/mailer.go`<br>`NewLogMailer` | Compiler error: `cannot use LogMailer{...} (value of type LogMailer) as Mailer value in return statement: LogMailer does not implement Mailer (method Send has pointer receiver)` | Compiler (`go build ./helper/...`) | `NewLogMailer` mengembalikan struct value `LogMailer{...}`, padahal method `Send` memiliki pointer receiver `(m *LogMailer)`. Mutasi internal `attempts` (atomic) dan `mu` (mutex) juga wajib di-share via pointer. | Mengembalikan pointer `&LogMailer{...}` agar memenuhi kontrak interface `Mailer` dan aman konkurensi antar-goroutine worker. | Build `helper` berhasil tanpa error; pointer receiver valid. |
| **2** | `app/service/outbox_relay.go`<br>`RelayOnce` | Saat Redis mati, event outbox yang gagal di-publish tetap ditandai `published_at = NOW()`. Event hilang permanen dan notifikasi tidak pernah terkirim saat Redis menyala kembali. | Uji C.2 (Redis mati, KRS tetap berjalan) | `ids = append(ids, e.ID)` dieksekusi SEBELUM `r.publish(ctx, e)`. Jika publish gagal, ID tetap masuk ke `ids` dan dipanggil oleh `repo.MarkPublished(ctx, ids)`. | Pindahkan pemanggilan `r.publish(ctx, e)` lebih dulu; hanya append ID ke slice `ids` jika publish sukses tanpa error. | Pada uji C.2 saat Redis mati, 3 outbox event tetap `published_at IS NULL` (`outbox_pending: 3`), dan segera terkirim setelah Redis hidup. |
| **3** | `app/service/course_service.go`<br>`Enroll` | Dual-write atomicity violation: jika langkah 5 (pencatatan enrollment) gagal karena duplikat (409), catatan outbox tetap tersimpan dan email konfirmasi tetap terkirim. | Uji C.1 (Percobaan #2 duplicate enroll) | Pemanggilan `s.outbox.Insert(ctx, ...)` tidak menggunakan objek transaksi `tx`, melainkan repository pool langsung (autocommit). | Ubah pemanggilan menjadi `s.outbox.WithTx(tx).Insert(ctx, ...)` sehingga outbox ikut dibatalkan saat rollback. | Pada uji C.1 permintaan 2 (409 ALREADY_ENROLLED) tidak meninggalkan row baru di outbox (ID sequence terlewat karena rollback). |
| **4** | `app/service/course_service.go`<br>`Drop` | Mahasiswa yang keluar dari kelas (Drop) menerima email konfirmasi pendaftaran ("KRS: berhasil terdaftar di...") alih-alih konfirmasi pembatalan. | Uji C.1 (Pemeriksaan `/notifications/mine` & `mail.log`) | Kode modul pada `Drop` menulis event type `model.EventEnrollmentCreated` bukannya `model.EventEnrollmentDropped`. | Mengganti event type menjadi `model.EventEnrollmentDropped`. | Subjek email di `logs/mail.log` dan DB tercatat tepat: `"KRS: keluar dari IF102 Basis Data Lanjut"`. |
| **5** | `app/service/notification_worker.go`<br>`HandleSend` | Compiler error: `cannot use worker.HandleSend (value of type func(context.Context, asynq.Task) error) as asynq.HandlerFunc value in argument to mux.HandleFunc` | Compiler (`go build ./...`) | Parameter task pada method handler didefinisikan sebagai nilai `t asynq.Task`, padahal kontrak `asynq.HandlerFunc` membutuhkan pointer `*asynq.Task`. | Ubah parameter menjadi pointer `t *asynq.Task`. | Kompilasi package main dan registrasi router worker `mux.HandleFunc` berhasil. |
| **6** | `app/service/notification_worker.go`<br>`HandleSend` | Task dengan kegagalan permanen (domain `.invalid`) tidak langsung diarsipkan ke `archived`, melainkan di-retry berulang kali sampai kuota retry habis (5 kali). | Uji C.4 (Permanent failure) & Eksperimen D.3 | Format error menggunakan `%v: %w` (`fmt.Errorf("%v: %w", asynq.SkipRetry, err)`), sehingga `asynq.SkipRetry` menjadi string biasa, bukan error wrapped. `errors.Is` di Asynq bernilai false. | Ubah format menjadi `%w: %v` agar `asynq.SkipRetry` dibungkus dengan verb `%w` yang dapat dideteksi oleh `errors.Is`. | Pada uji C.4, task dengan domain `.invalid` langsung masuk `archived` pada attempt 1/6 dengan `retry: 0`. |
| **7** | `app/service/notification_worker.go`<br>`HandleSend` | Race condition pada idempotency claim: dua worker konkuren dapat saling bertabrakan dan mengirim email ganda karena claim tidak terikat transaksi `WithTxOnce`. | Uji C.5 (Idempotency 6 salinan task) | Method `Claim` dipanggil langsung melalui `w.notifications.Claim(...)` tanpa mengikat koneksi transaksi `WithTx(tx)`. | Ubah menjadi `w.notifications.WithTx(tx).Claim(...)` di dalam closure `WithTxOnce`. | Pada uji C.5 dengan 6 salinan task, tepat 1 email terkirim (`already_sent: false`) dan 5 salinan lainnya di-skip (`already_sent: true`). |
| **8** | `config/queue.go`<br>`NewWorkerServer` | Worker Asynq menyala normal tetapi tidak pernah mengambil dan memproses task apa pun dari Redis. Task tertahan di antrean selamanya. | Uji C.1 / Uji Langkah 11 | Konfigurasi `Queues` memetakan nama jenis task (`model.TaskSendNotification` = `"notification:send"`), bukan nama queue (`model.QueueNotifications` = `"notifications"`). | Ubah key map antrean menjadi `Queues: map[string]int{model.QueueNotifications: 1}`. | Worker segera mendengarkan antrean `"notifications"` dan memproses seluruh background task. |
| **9** | `app/service/outbox_relay.go`<br>`publish` | Compiler error: `cannot use job (variable of type model.NotificationJob) as []byte value in argument to asynq.NewTask` | Compiler (`go build ./app/...`) | Fungsi `asynq.NewTask` membutuhkan argumen payload bertipe slice byte `[]byte`, sedangkan kode modul langsung mengoper struct `job`. | Lakukan `json.Marshal(job)` terlebih dahulu dan oper byte slice hasil marshaling ke `asynq.NewTask`. | Build relay outbox sukses dan worker dapat melakukan unmarshal payload task secara presisi. |

---

## Bukti Sebelum dan Sesudah Perbaikan (Detail 9 Kasalahan)

### 1. Bug 1: Pointer Receiver pada Mailer Interface
* **Sebelum Perbaikan:**
  ```go
  // helper/mailer.go
  func NewLogMailer(path string, delay time.Duration, failEvery int) (Mailer, error) {
      // ...
      return LogMailer{path: path, delay: delay, failEvery: int64(failEvery)}, nil
  }
  ```
  Keluaran Compiler:
  `cannot use LogMailer{...} (value of type LogMailer) as Mailer value in return statement: LogMailer does not implement Mailer (method Send has pointer receiver)`
* **Sesudah Perbaikan:**
  ```go
  return &LogMailer{path: path, delay: delay, failEvery: int64(failEvery)}, nil
  ```
  Keluaran Compiler: Sukses (`go build ./helper/...` exit code 0).

---

### 2. Bug 2: Logika Append ID Sebelum Enqueue pada Relay
* **Sebelum Perbaikan:**
  ```go
  for _, e := range events {
      ids = append(ids, e.ID) // BUG: Ditambahkan sebelum publish
      if err := r.publish(ctx, e); err != nil {
          enqueueErr = err
          break
      }
  }
  repo.MarkPublished(ctx, ids) // BUG: ID yang gagal tetap ditandai published!
  ```
  Gejala: Saat Redis dimatikan, `outbox_pending` langsung menjadi 0 padahal task gagal masuk ke Redis. Notifikasi hilang selamanya.
* **Sesudah Perbaikan:**
  ```go
  for _, e := range events {
      if err := r.publish(ctx, e); err != nil {
          enqueueErr = err
          break
      }
      ids = append(ids, e.ID) // HANYA ditambahkan jika publish berhasil
  }
  repo.MarkPublished(ctx, ids)
  ```
  Bukti Uji C.2: Saat Redis dimatikan, `outbox_pending: 3`. Setelah Redis dinyalakan kembali, relay memproses 3 event tertunda dan `notifications_sent` menjadi 7.

---

### 3. Bug 3: Outbox Autocommit di Luar Transaksi KRS
* **Sebelum Perbaikan:**
  ```go
  // app/service/course_service.go
  s.outbox.Insert(ctx, model.EventEnrollmentCreated, ...) // BUG: tanpa WithTx(tx)
  ```
* **Sesudah Perbaikan:**
  ```go
  s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentCreated, ...)
  ```
  Bukti Uji C.1: Pendaftaran duplikat menghasilkan HTTP 409 `ALREADY_ENROLLED` dan transaksi dibatalkan (rollback). Tidak ada entri outbox yang tersimpan untuk request gagal tersebut.

---

### 4. Bug 4: Event Type Salah pada Pembatalan Kelas (Drop)
* **Sebelum Perbaikan:**
  ```go
  // app/service/course_service.go (Drop)
  return s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentCreated, ...)
  ```
* **Sesudah Perbaikan:**
  ```go
  return s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentDropped, ...)
  ```
  Bukti Uji C.1:
  Response `GET /api/v1/notifications/mine`:
  `{"subject": "KRS: keluar dari IF102 Basis Data Lanjut"}`

---

### 5. Bug 5: Signature Handler Asynq Tanpa Pointer
* **Sebelum Perbaikan:**
  ```go
  func (w *NotificationWorker) HandleSend(ctx context.Context, t asynq.Task) error
  ```
  Keluaran Compiler:
  `cannot use worker.HandleSend (value of type func(context.Context, asynq.Task) error) as asynq.HandlerFunc value in argument to mux.HandleFunc`
* **Sesudah Perbaikan:**
  ```go
  func (w *NotificationWorker) HandleSend(ctx context.Context, t *asynq.Task) error
  ```
  Keluaran Compiler: Sukses.

---

### 6. Bug 6: Wrapping Verb SkipRetry (%v vs %w)
* **Sebelum Perbaikan:**
  ```go
  return fmt.Errorf("%v: %w", asynq.SkipRetry, err)
  ```
  Bukti Uji (D.3 Counter Experiment):
  `{"time":"...","level":"WARN","msg":"task_failed","attempt":"1/6","error":"skip retry for the task:..."}`  
  `{"time":"...","level":"WARN","msg":"task_failed","attempt":"2/6","error":"skip retry for the task:..."}`  
  `{"time":"...","level":"WARN","msg":"task_failed","attempt":"3/6","error":"skip retry for the task:..."}`  
  Status antrean: `queue.retry: 1`, `queue.archived: 0`. Task diulang terus-menerus!
* **Sesudah Perbaikan:**
  ```go
  return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
  ```
  Bukti Uji C.4:
  `{"time":"...","level":"WARN","msg":"task_failed","attempt":"1/6","error":"skip retry for the task: pengiriman gagal permanen: alamat salah@kampus.invalid tidak ditemukan"}`  
  `{"time":"...","level":"WARN","msg":"Retry exhausted for task id=outbox-21","component":"asynq"}`  
  Status antrean: `queue.retry: 0`, `queue.archived: 1`. Task langsung masuk archived pada attempt pertama.

---

### 7. Bug 7: Idempotency Claim Tidak Terikat Transaksi
* **Sebelum Perbaikan:**
  ```go
  claimed, err := w.notifications.Claim(ctx, ...) // BUG: tanpa WithTx(tx)
  ```
* **Sesudah Perbaikan:**
  ```go
  claimed, err := w.notifications.WithTx(tx).Claim(ctx, ...)
  ```
  Bukti Uji C.5: Dengan 6 salinan task serentak di Redis, row notification hanya 1, email hanya 1, dan 5 task lainnya mengeksekusi branch `already_sent: true`.

---

### 8. Bug 8: Nama Queue Tertukar dengan Task Type
* **Sebelum Perbaikan:**
  ```go
  // config/queue.go
  Queues: map[string]int{model.TaskSendNotification: 1} // "notification:send"
  ```
  Gejala: Task di Redis dimasukkan ke antrean `"notifications"`, tetapi worker hanya mengambil dari antrean `"notification:send"`. Semua task menggantung di status pending.
* **Sesudah Perbaikan:**
  ```go
  Queues: map[string]int{model.QueueNotifications: 1} // "notifications"
  ```
  Bukti: Seluruh task di antrean `"notifications"` segera dieksekusi oleh worker secara normal.

---

### 9. Bug 9: Tipe Parameter Payload pada asynq.NewTask
* **Sebelum Perbaikan:**
  ```go
  task := asynq.NewTask(model.TaskSendNotification, job, ...) // job adalah struct model.NotificationJob
  ```
  Keluaran Compiler:
  `cannot use job (variable of type model.NotificationJob) as []byte value in argument to asynq.NewTask`
* **Sesudah Perbaikan:**
  ```go
  payloadBytes, err := json.Marshal(job)
  if err != nil { return fmt.Errorf(...) }
  task := asynq.NewTask(model.TaskSendNotification, payloadBytes, ...)
  ```
  Keluaran Compiler: Sukses; payload JSON ditransmisikan dan di-decode secara valid.
