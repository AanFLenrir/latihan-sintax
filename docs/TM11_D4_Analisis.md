# ANALISIS SISTEM ASINKRON & BACKGROUND JOBS (TM 11)
**Mata Kuliah:** Praktikum Pemrograman Backend Lanjut (Pertemuan 11: Background Jobs & Asynchronous Processing)  
**Mahasiswa:** MOHAMMAD ANHAZ ABDILAH (NIM: 434241102)  
**Topik:** Arsitektur Job Queue, Dual Write, Idempotency, dan Transactional Outbox

---

### Pertanyaan 1: Mengapa Pola "Enqueue ke Redis Setelah Commit dengan Retry di Handler" Tetap Bermasalah?

**Usulan Teman:** Enqueue task ke Redis tepat setelah commit database, dan bila enqueue gagal, lakukan retry 3 kali langsung di dalam handler HTTP.

**Analisis Skenario Konkret:**

1. **Skenario Kehilangan Notifikasi (Dual Write Breakdown):**
   * *Kasus:* Transaksi PostgreSQL berhasil di-commit (data KRS mahasiswa tersimpan di database). Tepat setelah commit, server Redis mengalami gangguan (mati, crash, atau jaringan timeout). Handler mencoba enqueue ke Redis hingga 3 kali dan semuanya gagal. Karena jatah retry habis, handler akhirnya mengembalikan respons (misalnya HTTP 500 atau error log).
   * *Dampak:* Notifikasi hilang permanen tanpa jejak (*lost event*), karena database tidak pernah mencatat riwayat event tersebut di tabel mana pun dan Redis tidak menerimanya. Mahasiswa sudah resmi terdaftar, namun email konfirmasi tidak akan pernah dikirim. Lebih parah lagi jika proses API mengalami crash (*kill -9*, mati listrik, restart pod) tepat setelah baris commit sebelum baris enqueue dijalankan.

2. **Skenario Pembengkakan Waktu Respons Mahasiswa (Latency & Connection Hogging):**
   * *Kasus:* Ketika Redis sedang mengalami degradasi atau overload, operasi network call ke Redis membutuhkan waktu timeout (misal 2 detik per percobaan). Jika handler melakukan retry 3 kali di dalam siklus HTTP request-response, mahasiswa dipaksa menunggu di layar browser selama minimal 6 detik (3 x 2 detik) hanya untuk menunggu operasi pengiriman pesan yang sebenarnya tidak dibutuhkan untuk menyusun respons KRS.
   * *Dampak:* Mengorbankan prinsip dasar pemisahan kasir dan dapur (Bagian A.1). Jika ribuan mahasiswa serentak mendaftar saat portal KRS dibuka, koneksi HTTP server (Fiber worker pool) akan habis tertahan menunggu timeout Redis, mengakibatkan *cascading failure* di mana portal web kampus menjadi *down* dan mahasiswa tidak bisa mendaftar sama sekali.

*Kesimpulan:* Pola transactional outbox menyelesaikan kedua masalah di atas secara tuntas: pencatatan event ke tabel outbox berlangsung dalam orde milidetik bersama transaksi KRS, dan relay yang berjalan di background akan terus mencoba memindahkan event ke Redis tanpa pernah membebani latensi mahasiswa maupun kehilangan data.

---

### Pertanyaan 2: Redis Restart Tanpa Persistence Saat 100 Task Masih Pending

**Kondisi:** Redis dijalankan ulang tanpa konfigurasi persistence (AOF/RDB mati), 100 task di memori Redis hilang, sedangkan di PostgreSQL tabel outbox sudah menandai ke-100 event tersebut sebagai `published_at IS NOT NULL`.

**Analisis dan Solusi Pemulihan:**

1. **Apa yang terjadi pada 100 notifikasi tersebut?**
   * Ke-100 notifikasi tersebut tidak akan pernah diproses atau dikirim oleh worker karena task-nya sudah lenyap dari memori RAM Redis. Di sisi lain, relay outbox tidak akan mengambilnya kembali karena filter query relay hanya mencari baris dengan `published_at IS NULL`. Notifikasi tersebut menjadi "terlantar" (*orphaned events*).

2. **Bagaimana mendeteksi masalah menggunakan data PostgreSQL yang tersedia?**
   * Masalah ini dapat dideteksi secara pasti dengan membandingkan tabel `outbox` dan tabel `notifications`.
   * Sebuah event outbox yang berstatus `published_at IS NOT NULL` tetapi tidak memiliki baris padanannya di tabel `notifications` setelah batas waktu toleransi pemrosesan (misal lebih dari 5 atau 10 menit) adalah bukti nyata bahwa task tersebut hilang di broker.

3. **Query SQL Deteksi:**
   ```sql
   SELECT o.id, o.event_type, o.payload, o.created_at, o.published_at
   FROM outbox o
   LEFT JOIN notifications n ON n.outbox_id = o.id
   WHERE o.published_at IS NOT NULL
     AND n.id IS NULL
     AND o.published_at < NOW() - INTERVAL '5 minutes'
   ORDER BY o.id ASC;
   ```

4. **Bagaimana memulihkan event yang hilang?**
   * Cukup ubah kembali kolom `published_at` menjadi `NULL` untuk event-event yang teridentifikasi hilang tersebut:
   ```sql
   UPDATE outbox
   SET published_at = NULL
   WHERE id IN (
       SELECT o.id
       FROM outbox o
       LEFT JOIN notifications n ON n.outbox_id = o.id
       WHERE o.published_at IS NOT NULL
         AND n.id IS NULL
         AND o.published_at < NOW() - INTERVAL '5 minutes'
   );
   ```
   * Begitu `published_at` di-set ke `NULL`, Outbox Relay pada putaran berikutnya akan otomatis mengambil kembali event-event tersebut via `FetchUnpublished` dan memasukkannya ulang ke antrean Redis.

5. **Bagaimana menghindari pengiriman email ganda?**
   * Idempotency key di tabel `notifications` (`CONSTRAINT notifications_outbox_user_unique UNIQUE (outbox_id, user_id)`) dan klausa `ON CONFLICT DO NOTHING` menjamin bahwa seandainya dari 100 task tersebut ada sebagian kecil yang sebenarnya sudah sempat dikerjakan sebelum Redis mati, worker akan mendeteksi `claimed == false` dan langsung melewatinya (`already_sent = true`) tanpa mengirim email kedua.

---

### Pertanyaan 3: Mengapa Worker Menggunakan `WithTxOnce`, Sedangkan `CourseService` Menggunakan `WithTx`?

**Analisis Mekanisme:**

1. **Mengapa `CourseService` Menggunakan `WithTx`?**
   * Di dalam `CourseService`, seluruh operasi (memeriksa SKS, mengunci kursi kelas, mencatat enrollment, dan menyimpan event outbox) adalah **operasi database PostgreSQL murni**.
   * Jika terjadi serializable error, transient connection timeout, atau deadlock pada baris database, transaksi aman untuk di-retry secara utuh dari awal oleh `TxManager.WithTx` karena rollback database mengembalikan state data persis ke kondisi awal sebelum fungsi dijalankan. Tidak ada *side-effect* ke dunia luar.

2. **Mengapa Worker Menggunakan `WithTxOnce`?**
   * Di dalam worker, kita melakukan **tindakan eksternal ke sistem luar**, yaitu mengirim email (`w.mailer.Send`) sambil memegang claim transaksi database.
   * Email yang sudah terkirim ke server SMTP atau jaringan luar **tidak dapat di-rollback**.
   * Jika worker menggunakan `WithTx` yang melakukan retry otomatis saat terjadi error database/deadlock, fungsi closure akan diulang dari awal. Akibatnya, email bisa terkirim dua kali atau lebih kepada mahasiswa!
   * Dengan `WithTxOnce`, transaksi hanya dijalankan **tepat satu kali** tanpa ada pengulangan internal. Urusan retry diserahkan sepenuhnya ke framework Asynq yang memahami status task, delay backoff, dan counter attempt.

3. **Risiko Fatal Jika Mekanismenya Ditukar:**
   * *Jika Worker memakai `WithTx`:* Terjadi pengiriman email berulang kali ke mahasiswa saat database mengalami deadlock sesaat, merusak integritas sistem dan membingungkan pengguna.
   * *Jika `CourseService` memakai `WithTxOnce`:* Deadlock konkurensi antar-mahasiswa yang berebut kelas pada detik yang sama tidak akan di-retry oleh database, sehingga request pendaftaran mahasiswa langsung gagal padahal sebenarnya kelas masih tersedia dan konflik bisa diselesaikan dalam retry 2–5 milidetik.

---

### Pertanyaan 4: Urutan Email Tidak Dijamin pada Sistem Asinkron

**Fenomena:** Pada pengujian C.1, email "keluar dari kelas" dapat diterima mahasiswa lebih dahulu daripada email "berhasil terdaftar di kelas".

1. **Kapan masalah ini berdampak nyata bagi pengguna?**
   * Masalah ini berdampak fatal pada skenario di mana urutan pesan membawa status hukum/keabsahan atau instruksi bergantung waktu (*time-sensitive dependencies*).
   * Contoh: Mahasiswa mendaftar kelas pada pukul 08:00:01, lalu membatalkannya pada pukul 08:00:02. Jika email konfirmasi pendaftaran baru tiba pukul 08:00:10 dan email pembatalan tiba pukul 08:00:05, mahasiswa mengira status terakhirnya adalah *terdaftar* di kelas tersebut, padahal di database statusnya sudah dibatalkan. Hal yang sama berlaku pada sistem perbankan (transfer uang lalu pembatalan transaksi).

2. **Mengapa urutan task tidak otomatis sama dengan urutan email?**
   * Sistem worker bersifat multi-worker dan konkruen (`WORKER_CONCURRENCY > 1`).
   * Task pertama (daftar) dan task kedua (keluar) dikerjakan oleh goroutine worker yang berbeda secara paralel. Jika worker 1 mengalami jeda network, latency SMTP lambat, atau terkena temporary error sehingga masuk antrean retry, worker 2 yang mengerjakan task kedua bisa menyelesaikan pengiriman email lebih cepat.
   * Pengurutan task di antrean (*queue FIFO*) hanya menjamin urutan saat task diambil, bukan urutan penyelesaian saat eksekusi melibatkan sistem eksternal.

3. **Satu Solusi yang Masuk Akal:**
   * **Message Versioning / Timestamping & User Mailbox In-App:**
     Setiap notifikasi menyertakan nomor versi/sequence (`outbox_id` atau timestamp event) yang tegas pada subjek dan badan email (misal: *"Status KRS Kelas IF102 per 01 Nov 2026 18:49:33 WIB (Revisi #4)"*).
   * **Atau Partisi Queue per Mahasiswa (Partitioning / Sharding Queue):**
     Memetakan task milik mahasiswa tertentu ke antrean atau worker yang memproses secara serial untuk satu user ID.
   * **Endpoint `/notifications/mine`:**
     Seperti yang diimplementasikan pada aplikasi kita, API selalu mengurutkan berdasarkan `outbox_id DESC` (urutan transaksi event), bukan `id` notifikasi atau waktu pengiriman.

4. **Konsekuensi Solusi Terhadap Performa atau Kompleksitas:**
   * Jika memaksakan pemrosesan strictly serial per mahasiswa (misal lock berbasis user atau antrean berurutan ketat), *throughput* konkurensi sistem akan menurun drastis saat seorang mahasiswa melakukan banyak aksi serentak.
   * Kompleksitas arsitektur meningkat karena memerlukan *message broker partitioning* (seperti Apache Kafka partitions atau Redis dynamic per-user queues), mekanisme *head-of-line blocking handling*, serta penanganan deadlock antrean yang jauh lebih rumit.
