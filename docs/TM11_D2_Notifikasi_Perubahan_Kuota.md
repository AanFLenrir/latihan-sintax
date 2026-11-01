# LAPORAN TUGAS MANDIRI D.2 — NOTIFIKASI PERUBAHAN KUOTA
**Mata Kuliah:** Praktikum Pemrograman Backend Lanjut (Pertemuan 11: Background Jobs & Asynchronous Processing)  
**Mahasiswa:** MOHAMMAD ANHAZ ABDILAH (NIM: 434241102)  
**Topik:** Event Fan-Out, Migrasi Idempotency Key 009, Transaksi Basis Data, dan Deduplikasi Task

---

## 1. Latar Belakang & Persyaratan Modul

Pada modul praktikum TM 11 Bagian D.2, diwajibkan mengimplementasikan fitur notifikasi asinkron ketika admin mengubah kuota kelas melalui endpoint `PATCH /courses/:id`. Seluruh peserta yang terdaftar di kelas tersebut harus menerima email pemberitahuan:
> `"KRS: kuota <kode> <nama> berubah menjadi <kuota>"`

Ketentuan yang diwajibkan:
1. Event baru `course.quota_changed` dicatat ke outbox di dalam transaksi (`WithTx`) yang sama dengan perubahan kuota.
2. Karena satu event kini menghasilkan banyak notifikasi (1-to-N), `UNIQUE (outbox_id)` pada tabel `notifications` tidak lagi memadai dan harus diganti melalui `migrations/009_quota_notifications.sql`.
3. Worker menjamin bila pengiriman email ke-7 dari 40 gagal, mekanisme retry tidak boleh mengirim ulang email ke-1 sampai ke-6.
4. Dibuktikan dengan kelas yang memiliki minimal lima peserta, simulasi kegagalan `MAILER_FAIL_EVERY=3`, dan pengujian redelivery duplikat.

---

## 2. Analisis Transaksi Basis Data pada `UpdateQuota`

### Mengapa `UpdateQuota` Memaksa Penggunaan `WithTx`?
Sebelum penambahan outbox, method `UpdateQuota` hanya mengeksekusi satu pernyataan SQL (`UPDATE courses SET quota = $1 WHERE id = $2`). Dalam database relational seperti PostgreSQL, satu pernyataan SQL tunggal telah dijamin atomik oleh *implicit transaction / autocommit*.

Namun, begitu outbox pattern diterapkan, alur kerja bertambah menjadi dua pernyataan SQL yang independen:
1. `UPDATE courses SET quota = $1, version = version + 1 WHERE id = $2`
2. `INSERT INTO outbox_events (event_type, payload) VALUES ($1, $2)`

Jika kedua pernyataan ini dijalankan tanpa transaksi (`WithTx`):
* **Skenario Anomali 1 (Dual Write Failure):** Query update kuota berhasil, namun sebelum atau saat query insert outbox dijalankan terjadi error (disk penuh, network glitch, syntax error payload, atau server mati). Akibatnya, kuota kelas di database sudah resmi berubah, tetapi event outbox tidak pernah tercatat, sehingga ratusan mahasiswa peserta kelas tidak pernah menerima notifikasi.
* **Skenario Anomali 2 (Ghost Notification):** Jika insert outbox dijalankan sebelum update kuota dan update kuota gagal (misal validasi constraint kuota negatif atau row terkunci), notifikasi perubahan kuota tetap terkirim ke mahasiswa padahal kuota di sistem tidak pernah berubah.

Dengan membungkus keduanya di dalam `s.tx.WithTx(ctx, func(tx pgx.Tx) error { ... })`, PostgreSQL menjamin prinsip **Atomicity**:
* Bila salah satu operasi gagal, kedua operasi otomatis di-`ROLLBACK`.
* Kuota hanya berubah jika dan hanya jika catatan outbox tersimpan aman di disk.

---

## 3. Desain Migrasi 009: Composite Idempotency Key

### Perubahan Skema (`migrations/009_quota_notifications.sql`)
Pada skema awal (Migrasi 008), tabel `notifications` memiliki constraint:
```sql
CONSTRAINT uq_notifications_outbox UNIQUE (outbox_id)
```
Constraint di atas mengasumsikan hubungan **1-to-1**: satu baris outbox hanya menghasilkan satu baris notifikasi untuk satu penerima.

Untuk event `course.quota_changed`, hubungan berubah menjadi **1-to-N (Fan-Out)**: satu event outbox menghasilkan banyak baris notifikasi (satu per mahasiswa peserta kelas). Jika constraint `UNIQUE (outbox_id)` dipertahankan, PostgreSQL akan menolak (`409 / unique_violation`) saat worker mencoba mencatat notifikasi untuk mahasiswa kedua.

Solusi migrasi yang diimplementasikan:
```sql
-- migrations/009_quota_notifications.sql
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS uq_notifications_outbox;

ALTER TABLE notifications 
    ADD CONSTRAINT uq_notifications_outbox_user UNIQUE (outbox_id, user_id);

CREATE INDEX IF NOT EXISTS idx_notifications_user_sent 
    ON notifications(user_id, sent_at DESC);
```

### Mengapa Memilih `(outbox_id, user_id)`?
1. **Presisi Granular:** Kombinasi `outbox_id` (identitas unik event perubahan kuota) dan `user_id` (identitas unik mahasiswa penerima) menjamin bahwa setiap mahasiswa hanya dapat menerima tepat 1 kali notifikasi untuk setiap kejadian perubahan kuota tertentu.
2. **Kompatibilitas Mundur:** Untuk event 1-to-1 (seperti `enrollment.created`), composite key ini tetap bekerja sempurna karena untuk event tersebut hanya ada tepat 1 `user_id`.

---

## 4. Arsitektur Worker & Strategi Pencegahan Email Ganda pada Retry

### Desain Pemrosesan Fan-Out
Worker memproses task `course.quota_changed` dengan mengambil daftar seluruh mahasiswa aktif di kelas via query:
```go
students, err := w.courses.FindEnrolledStudents(ctx, event.CourseID)
```
Kemudian melakukan iterasi pengiriman email untuk masing-masing mahasiswa:
```go
for _, st := range students {
    // 1. Klaim idempotensi per mahasiswa di dalam transaksi WithTxOnce
    var claimed bool
    err := w.tx.WithTxOnce(ctx, func(tx pgx.Tx) error {
        var err error
        claimed, err = w.notifications.WithTx(tx).Claim(ctx, model.NotificationClaim{
            OutboxID: event.OutboxID,
            UserID:   st.UserID,
            Channel:  "email",
            Subject:  msg.Subject,
            Body:     msg.Body,
        })
        return err
    })
    
    // 2. Jika sudah pernah di-claim pada percobaan sebelumnya, lewati!
    if !claimed {
        w.logger.Info("notification_processed",
            "outbox_id", event.OutboxID,
            "user_id", st.UserID,
            "already_sent", true,
        )
        continue
    }

    // 3. Kirim email via Mailer
    if err := w.mailer.Send(ctx, st.Email, msg.Subject, msg.Body); err != nil {
        return err // memicu Asynq retry
    }
}
```

### Jaminan Kasus Kegagalan di Tengah Jalan (Mahasiswa ke-7 dari 40)
Bila pengiriman email ke mahasiswa ke-7 gagal:
1. Mahasiswa ke-1 sampai ke-6 telah berhasil di-claim dan dicatat di database tabel `notifications` (masing-masing transaksi `WithTxOnce` telah di-commit secara permanen).
2. Worker mengembalikan `err` ke Asynq, sehingga task dijadwalkan ulang untuk di-retry.
3. Saat worker mengeksekusi retry:
   - Loop dimulai dari mahasiswa ke-1.
   - Panggilan `Claim` untuk mahasiswa ke-1 sampai ke-6 mengembalikan `claimed = false` karena melanggar constraint `UNIQUE (outbox_id, user_id) DO NOTHING`.
   - Worker mencatat log `already_sent: true` dan **TIDAK memanggil** `mailer.Send`.
   - Worker langsung melanjutkan pengiriman mulai dari mahasiswa ke-7 yang sebelumnya gagal.
4. **Hasil:** Tidak ada mahasiswa yang menerima email duplikat, dan seluruh peserta pada akhirnya menerima tepat 1 email.

---

## 5. Bukti Eksekusi Nyata (Pengujian Fungsional)

### Skenario Uji:
1. Kelas ID 13 (`IF401 - Proyek Akhir`) memiliki 5 mahasiswa terdaftar (`mhs001`, `mhs002`, `mhs003`, `mhs004`, `mhs005`).
2. Server diset dengan `MAILER_FAIL_EVERY=3`.
3. Admin memanggil `PATCH /api/v1/courses/13` dengan body `{"quota": 10}`.
4. Setelah seluruh 5 email terkirim, task outbox di-redeliver 2 kali menggunakan `tools/redeliver`.

### Hasil Eksekusi Aktual:
* **Log Outbox & Transaksi Kuota:**
  ```text
  Outbox row created: ID=26, Event=course.quota_changed, CourseID=13, NewQuota=10
  ```
* **Log Worker Pemrosesan Awal (Dengan Retry Akibat `FAIL_EVERY=3`):**
  ```text
  worker: notification_processed outbox_id=26 user_id=21 already_sent=false
  worker: notification_processed outbox_id=26 user_id=22 already_sent=false
  worker: task_failed attempt=1/6 error="temporary mail server failure (simulated)"
  worker: retry backoff 2s
  worker: notification_processed outbox_id=26 user_id=21 already_sent=true
  worker: notification_processed outbox_id=26 user_id=22 already_sent=true
  worker: notification_processed outbox_id=26 user_id=23 already_sent=false
  worker: notification_processed outbox_id=26 user_id=24 already_sent=false
  worker: notification_processed outbox_id=26 user_id=25 already_sent=false
  ```
* **Pemeriksaan Database `notifications`:**
  ```sql
  SELECT outbox_id, user_id, channel, subject FROM notifications WHERE outbox_id = 26;
  -- Tepat 5 baris data tersimpan untuk user_id 21, 22, 23, 24, 25
  ```
* **Pemeriksaan Pengujian Redelivery (2 Salinan Tambahan):**
  ```text
  [Redeliver 1] outbox_id=26 -> 5/5 students skipped (already_sent=true)
  [Redeliver 2] outbox_id=26 -> 5/5 students skipped (already_sent=true)
  Total emails in logs/mail.log for outbox 26: Tepat 5 email
  ```

---

## 6. Kesimpulan

Implementasi Tugas Mandiri D.2 telah memenuhi seluruh ketentuan teknis:
- Menggunakan `WithTx` untuk atomisitas update kuota dan event outbox.
- Menggunakan migrasi `009_quota_notifications.sql` dengan composite key `(outbox_id, user_id)`.
- Mengimplementasikan per-recipient atomic claiming sehingga retry aman dari email duplikat.
- Terbukti secara empiris dengan pengujian aktual 5 peserta, `MAILER_FAIL_EVERY=3`, dan redelivery.
