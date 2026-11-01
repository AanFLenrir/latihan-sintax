import os
import subprocess

html_content = """<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="UTF-8">
<title>Laporan Praktikum Pemrograman Backend Lanjut - Pertemuan 11</title>
<style>
  @page {
    size: A4;
    margin: 18mm 14mm 18mm 14mm;
    @bottom-right {
      content: counter(page);
    }
  }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    line-height: 1.55;
    color: #1f2328;
    font-size: 11.5px;
    background-color: #ffffff;
  }
  .header-card {
    border: 1px solid #d0d7de;
    background-color: #f6f8fa;
    border-radius: 6px;
    padding: 14px 18px;
    margin-bottom: 20px;
  }
  .header-card h1 {
    font-size: 17px;
    color: #0969da;
    margin: 0 0 6px 0;
    border: none;
    padding: 0;
  }
  .header-meta {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 4px 16px;
    font-size: 11px;
    color: #57606a;
  }
  h1, h2, h3, h4 {
    color: #1f2328;
    page-break-after: avoid;
  }
  h2 {
    font-size: 14px;
    border-bottom: 1.5px solid #d0d7de;
    padding-bottom: 4px;
    margin-top: 18px;
    margin-bottom: 8px;
    color: #0969da;
  }
  h3 {
    font-size: 12.5px;
    margin-top: 12px;
    margin-bottom: 6px;
  }
  table {
    border-collapse: collapse;
    width: 100%;
    margin: 10px 0;
    font-size: 10.5px;
  }
  th, td {
    border: 1px solid #d0d7de;
    padding: 5px 8px;
    text-align: left;
    vertical-align: top;
  }
  th {
    background-color: #f6f8fa;
    font-weight: 600;
  }
  pre, code {
    font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
    font-size: 10.5px;
  }
  pre {
    background-color: #f6f8fa;
    border: 1px solid #d0d7de;
    border-radius: 4px;
    padding: 8px 10px;
    overflow-x: auto;
    white-space: pre-wrap;
    word-break: break-all;
    margin: 8px 0;
    line-height: 1.4;
  }
  p {
    margin: 6px 0;
  }
  ul, ol {
    margin: 6px 0;
    padding-left: 20px;
  }
  li {
    margin-bottom: 3px;
  }
  .badge {
    display: inline-block;
    padding: 1px 6px;
    font-size: 10px;
    font-weight: 600;
    border-radius: 10px;
  }
  .badge-pass {
    background-color: #dafbe1;
    color: #1a7f37;
    border: 1px solid #aceebb;
  }
  .page-break {
    page-break-before: always;
  }
</style>
</head>
<body>

<div class="header-card">
  <h1>LAPORAN PRAKTIKUM PEMROGRAMAN BACKEND LANJUT — TM 11</h1>
  <div style="font-weight: 600; font-size: 13px; margin-bottom: 8px; color: #24292f;">
    Topik: Background Jobs, Asynchronous Processing, dan Transactional Outbox
  </div>
  <div class="header-meta">
    <div><strong>Nama:</strong> MOHAMMAD ANHAZ ABDILAH</div>
    <div><strong>NIM:</strong> 434241102</div>
    <div><strong>Program Studi:</strong> D4 Teknik Informatika — Fakultas Vokasi Universitas Airlangga</div>
    <div><strong>Repository:</strong> AanFLenrir/latihan-fiber (Branch: main)</div>
    <div><strong>Lingkungan:</strong> Windows 11 64-bit, Go 1.24, Fiber v2.52, PostgreSQL 18, MongoDB v2, Memurai v4.1.2 (Redis), Asynq v0.26.0</div>
    <div><strong>Tanggal Pengujian:</strong> 01 November 2026</div>
  </div>
</div>

<h2>BAGIAN D.1 — BERITA ACARA PERBAIKAN SEMBILAN KESALAHAN</h2>
<p>
  Sesuai dengan ketentuan <em>Modul 11 Edisi Puzzle</em>, kode modul memuat sembilan kesalahan yang sengaja ditanamkan. 
  Seluruh sembilan kesalahan telah diinvestigasi, dibuktikan akar masalahnya, diperbaiki, dan divalidasi melalui pengujian compiler maupun pengujian fungsional database.
</p>

<table>
  <thead>
    <tr>
      <th style="width: 25px;">No.</th>
      <th style="width: 130px;">Lokasi (File, Fungsi)</th>
      <th style="width: 140px;">Gejala yang Diamati</th>
      <th style="width: 80px;">Uji Pengungkap</th>
      <th>Akar Masalah</th>
      <th>Perbaikan yang Diterapkan</th>
      <th style="width: 70px;">Status</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>1</strong></td>
      <td><code>helper/mailer.go</code><br><code>NewLogMailer</code></td>
      <td>Compiler error: <code>LogMailer does not implement Mailer (method Send has pointer receiver)</code>.</td>
      <td>Compiler (Langkah 4)</td>
      <td><code>NewLogMailer</code> mengembalikan struct value <code>LogMailer{...}</code>, padahal method <code>Send</code> memiliki pointer receiver <code>(m *LogMailer)</code>. State counter dan mutex juga wajib pointer agar terbagi antar goroutine.</td>
      <td>Mengembalikan pointer <code>&LogMailer{...}</code>.</td>
      <td><span class="badge badge-pass">TERVERIFIKASI</span></td>
    </tr>
    <tr>
      <td><strong>2</strong></td>
      <td><code>app/service/outbox_relay.go</code><br><code>RelayOnce</code></td>
      <td>Saat Redis mati, event yang gagal di-publish tetap ditandai <code>published_at = NOW()</code> di PostgreSQL sehingga notifikasi hilang permanen saat Redis hidup kembali.</td>
      <td>Uji C.2 (Redis mati)</td>
      <td><code>ids = append(ids, e.ID)</code> dieksekusi sebelum <code>r.publish(ctx, e)</code>. ID event yang gagal tetap masuk ke slice yang ditandai published oleh <code>MarkPublished</code>.</td>
      <td>Pindahkan <code>ids = append(...)</code> setelah evaluasi keberhasilan <code>r.publish</code>.</td>
      <td><span class="badge badge-pass">TERVERIFIKASI</span></td>
    </tr>
    <tr>
      <td><strong>3</strong></td>
      <td><code>app/service/course_service.go</code><br><code>Enroll</code></td>
      <td>Dual-write atomicity violation: jika langkah 5 (pencatatan enrollment) gagal karena duplicate (409), outbox event tetap tersimpan dan email tetap terkirim.</td>
      <td>Uji C.1 (Percobaan 2 duplicate)</td>
      <td>Pemanggilan <code>s.outbox.Insert(...)</code> menggunakan koneksi pool langsung tanpa membungkusnya dalam transaksi aktif (<code>tx</code>).</td>
      <td>Ubah pemanggilan menjadi <code>s.outbox.WithTx(tx).Insert(...)</code>.</td>
      <td><span class="badge badge-pass">TERVERIFIKASI</span></td>
    </tr>
    <tr>
      <td><strong>4</strong></td>
      <td><code>app/service/course_service.go</code><br><code>Drop</code></td>
      <td>Mahasiswa yang membatalkan kelas (Drop) menerima email "KRS: berhasil terdaftar di..." alih-alih konfirmasi pembatalan.</td>
      <td>Uji C.1 (Pemeriksaan /mine)</td>
      <td>Event type yang dicatat pada operasi Drop adalah <code>EventEnrollmentCreated</code> bukannya <code>EventEnrollmentDropped</code>.</td>
      <td>Ganti event type menjadi <code>model.EventEnrollmentDropped</code>.</td>
      <td><span class="badge badge-pass">TERVERIFIKASI</span></td>
    </tr>
    <tr>
      <td><strong>5</strong></td>
      <td><code>app/service/notification_worker.go</code><br><code>HandleSend</code></td>
      <td>Compiler error: <code>cannot use worker.HandleSend as asynq.HandlerFunc in mux.HandleFunc</code>.</td>
      <td>Compiler (Langkah 8)</td>
      <td>Parameter task pada method handler didefinisikan sebagai struct value <code>t asynq.Task</code>, padahal signature <code>asynq.HandlerFunc</code> mensyaratkan pointer <code>*asynq.Task</code>.</td>
      <td>Ubah tipe parameter menjadi <code>t *asynq.Task</code>.</td>
      <td><span class="badge badge-pass">TERVERIFIKASI</span></td>
    </tr>
    <tr>
      <td><strong>6</strong></td>
      <td><code>app/service/notification_worker.go</code><br><code>HandleSend</code></td>
      <td>Task dengan kegagalan permanen (domain <code>.invalid</code>) di-retry berulang kali hingga 5 kali alih-alih langsung masuk <code>archived</code>.</td>
      <td>Uji C.4 & Eksperimen D.3</td>
      <td>Error diformat dengan <code>%v: %w</code> (<code>asynq.SkipRetry</code> diformat dengan <code>%v</code>), sehingga <code>errors.Is</code> di internal Asynq tidak mengenali <code>SkipRetry</code>.</td>
      <td>Ganti format pembungkusan error menjadi <code>fmt.Errorf("%w: %v", asynq.SkipRetry, err)</code>.</td>
      <td><span class="badge badge-pass">TERVERIFIKASI</span></td>
    </tr>
    <tr>
      <td><strong>7</strong></td>
      <td><code>app/service/notification_worker.go</code><br><code>HandleSend</code></td>
      <td>Race condition pada claim idempotensi saat beberapa goroutine worker berjalan konkuren. Notifikasi ganda bisa terkirim.</td>
      <td>Uji C.5 (Salinan task)</td>
      <td>Method <code>Claim</code> dipanggil langsung melalui <code>w.notifications.Claim(...)</code> tanpa mengikat transaksi database <code>WithTx(tx)</code>.</td>
      <td>Gunakan <code>w.notifications.WithTx(tx).Claim(...)</code> di dalam closure <code>WithTxOnce</code>.</td>
      <td><span class="badge badge-pass">TERVERIFIKASI</span></td>
    </tr>
    <tr>
      <td><strong>8</strong></td>
      <td><code>config/queue.go</code><br><code>NewWorkerServer</code></td>
      <td>Worker Asynq tidak pernah mengambil dan memproses task apa pun dari Redis. Task menggantung selamanya di antrean pending.</td>
      <td>Uji C.1 (Relay vs Worker)</td>
      <td>Konfigurasi antrean pada <code>Queues</code> memetakan nama jenis task (<code>"notification:send"</code>) bukannya nama queue (<code>"notifications"</code>).</td>
      <td>Ubah key map antrean menjadi <code>Queues: map[string]int{model.QueueNotifications: 1}</code>.</td>
      <td><span class="badge badge-pass">TERVERIFIKASI</span></td>
    </tr>
    <tr>
      <td><strong>9</strong></td>
      <td><code>app/service/outbox_relay.go</code><br><code>publish</code></td>
      <td>Compiler error: <code>cannot use job (variable of type NotificationJob) as []byte value in argument to asynq.NewTask</code>.</td>
      <td>Compiler (Langkah 7)</td>
      <td>Fungsi <code>asynq.NewTask</code> membutuhkan payload bertipe <code>[]byte</code>, sedangkan modul langsung mengoper struct <code>model.NotificationJob</code>.</td>
      <td>Lakukan <code>json.Marshal(job)</code> terlebih dahulu lalu oper byte slice ke <code>asynq.NewTask</code>.</td>
      <td><span class="badge badge-pass">TERVERIFIKASI</span></td>
    </tr>
  </tbody>
</table>

<div class="page-break"></div>

<h2>BAGIAN D.2 — NOTIFIKASI PERUBAHAN KUOTA UNTUK SELURUH PESERTA</h2>
<p>
  Fitur ini menambahkan pengiriman notifikasi massal (*fan-out*) kepada seluruh peserta kelas yang telah terdaftar ketika Admin mengubah kuota kelas melalui endpoint <code>PATCH /api/v1/courses/:id</code>.
</p>

<h3>1. Mengapa Operasi Perubahan Kuota dan Pencatatan Outbox Harus Atomik?</h3>
<p>
  Sebelum Modul 11, <code>UpdateQuota</code> hanya menjalankan satu pernyataan SQL dengan optimistic locking. Penambahan notifikasi outbox menjadikannya dua operasi: (1) update kuota pada tabel <code>courses</code>, dan (2) insert event <code>course.quota_changed</code> pada tabel <code>outbox</code>.
  Kedua pernyataan ini <strong>wajib berada di dalam satu transaksi database (WithTx)</strong> karena:
</p>
<ul>
  <li>Jika kuota berhasil diubah tetapi pencatatan outbox gagal (karena koneksi terputus atau server mati), perubahan kuota tersimpan namun mahasiswa tidak pernah menerima pemberitahuan (*lost event*).</li>
  <li>Jika outbox tersimpan mandiri tetapi update kuota gagal (karena konflik versi <code>version_conflict</code> atau kuota di bawah jumlah peserta <code>quota_below_enrolled</code>), email perubahan kuota palsu akan terkirim padahal kuota kelas sebenarnya tidak berubah.</li>
</ul>

<h3>2. Migration 009 (`migrations/009_quota_notifications.sql`)</h3>
<p>
  Pada pendaftaran reguler, relasi antara event outbox dan notifikasi adalah 1-to-1 (satu event menghasilkan satu notifikasi untuk satu mahasiswa). Namun pada perubahan kuota, satu event outbox menghasilkan banyak notifikasi (1-to-N fan-out) untuk seluruh peserta kelas. 
  Oleh karena itu, constraint lama <code>UNIQUE (outbox_id)</code> diganti menjadi <strong>composite uniqueness <code>UNIQUE (outbox_id, user_id)</code></strong>.
</p>
<pre><code>-- migrations/009_quota_notifications.sql
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_outbox_unique;
ALTER TABLE notifications ADD CONSTRAINT notifications_outbox_user_unique UNIQUE (outbox_id, user_id);</code></pre>
<p>
  <strong>Keunggulan desain ini:</strong> Idempoten per mahasiswa. Jika pengiriman ke mahasiswa ke-3 gagal sementara (temporary failure), transaksi rollback hanya membatalkan claim mahasiswa ke-3. Saat Asynq me-retry task, mahasiswa ke-1 dan ke-2 yang sudah berhasil dikirim akan di-skip (<code>already_sent = true</code>) karena pasangan <code>(outbox_id, user_id)</code> mereka sudah terkunci di database.
</p>

<h3>3. Potongan Kode Penting D.2</h3>
<pre><code>// app/service/course_service.go (UpdateQuota dalam WithTx)
var updated model.Course
err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
    var err error
    updated, err = s.courses.WithTx(tx).UpdateQuota(ctx, id, req.Quota, req.Version)
    if err != nil {
        return err
    }
    return s.outbox.WithTx(tx).Insert(ctx, model.EventCourseQuotaChanged, model.CourseQuotaChangedEvent{
        CourseID: updated.ID, CourseCode: updated.Code, CourseName: updated.Name, Quota: updated.Quota,
    })
})</code></pre>

<pre><code>// app/service/notification_worker.go (Pemrosesan Fan-Out Idempoten)
func (w *NotificationWorker) handleQuotaChanged(ctx context.Context, job model.NotificationJob) error {
    var event model.CourseQuotaChangedEvent
    json.Unmarshal(job.Payload, &event)
    students, _ := w.courses.FindEnrolledStudents(ctx, event.CourseID)

    for _, student := range students {
        msg := BuildQuotaMessage(event, student)
        alreadySent := false
        err := w.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
            claimed, err := w.notifications.WithTx(tx).Claim(ctx, model.Notification{
                OutboxID: job.OutboxID, UserID: student.ID, Channel: "email",
                Subject: msg.Subject, Body: msg.Body,
            })
            if err != nil || !claimed {
                if !claimed { alreadySent = true }
                return err
            }
            return w.mailer.Send(ctx, msg)
        })
        if err != nil { return err } // Temporary error -> retry by Asynq
    }
    return nil
}</code></pre>

<h3>4. Hasil Pengujian Fungsional D.2 (Kelas IF399 dengan 5 Peserta & MAILER_FAIL_EVERY=3)</h3>
<ul>
  <li>Admin mengubah kuota kelas <code>IF399</code> (Kelas Rebutan) dari 5 menjadi 10 melalui <code>PATCH /api/v1/courses/13</code> (Status: <code>200 OK</code>).</li>
  <li>Event outbox ID 26 tercatat dengan type <code>course.quota_changed</code> dan status <code>published = true</code>.</li>
  <li>Dengan <code>MAILER_FAIL_EVERY=3</code>, kegagalan sementara terjadi pada attempt ke-3 dan ke-6. Worker otomatis di-retry oleh Asynq. Mahasiswa yang sudah sukses terkirim tidak menerima email ganda.</li>
  <li>Seluruh 5 mahasiswa (<code>mhs001</code> s.d. <code>mhs005</code>) menerima notifikasi di database dan tercatat di <code>logs/mail.log</code> dengan subjek tepat: <code>KRS: kuota IF399 Kelas Rebutan berubah menjadi 10</code>.</li>
  <li>Saat alat <code>tools/redeliver</code> dijalankan untuk menginjeksi 2 salinan task outbox 26, seluruh 5 mahasiswa berstatus <code>already_sent: true</code> dan 0 email ganda terkirim.</li>
</ul>

<div class="page-break"></div>

<h2>BAGIAN D.3 — BUKTI PENGUJIAN DAN PERCOBAAN PEMBANDING</h2>

<h3>1. Ringkasan Hasil Pengujian Penerimaan Bagian C</h3>
<table>
  <thead>
    <tr>
      <th>Skenario</th>
      <th>Kondisi Pengujian</th>
      <th>Hasil Aktual</th>
      <th>Status</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>C.1 Jalur Normal</strong></td>
      <td>Mahasiswa mendaftar (201), duplikat (409), drop (204), daftar (201), swap (200).</td>
      <td>Outbox tepat 4 baris, 4 notifikasi di <code>/notifications/mine</code>, 4 baris di <code>logs/mail.log</code>.</td>
      <td><span class="badge badge-pass">PASS</span></td>
    </tr>
    <tr>
      <td><strong>C.2 Redis Mati</strong></td>
      <td>Redis dimatikan, daftar ke kelas 5, 6, 7. Cek admin queue, lalu nyalakan Redis kembali.</td>
      <td>Ketiga pendaftaran 201 (2–17 ms). <code>outbox_pending: 3</code> saat mati. Setelah hidup, <code>outbox_pending: 0</code> dan 7 notifikasi terkirim.</td>
      <td><span class="badge badge-pass">PASS</span></td>
    </tr>
    <tr>
      <td><strong>C.3 Temporary Failure</strong></td>
      <td><code>MAILER_FAIL_EVERY=3</code>, daftar ke 6 kelas. Menunggu siklus retry Asynq.</td>
      <td>DB notifications = 6, <code>/notifications/mine</code> = 6, <code>logs/mail.log</code> = 6. Log mencatat minimal 2 kali <code>task_failed</code> sementara.</td>
      <td><span class="badge badge-pass">PASS</span></td>
    </tr>
    <tr>
      <td><strong>C.4 Permanent Failure</strong></td>
      <td>Akun <code>salahalamat</code> dengan email domain RFC 2606 <code>.invalid</code> mendaftar ke kelas 2.</td>
      <td>KRS sukses 201. Task gagal permanen, <code>archived</code> bertambah 1, <code>retry: 0</code>. Attempt berhenti di 1/6 dengan <code>SkipRetry</code>.</td>
      <td><span class="badge badge-pass">PASS</span></td>
    </tr>
    <tr>
      <td><strong>C.5 Idempotency</strong></td>
      <td>Injeksi 5 salinan task via <code>redeliver</code> + 1 task relay (total 6 salinan konkuren).</td>
      <td>DB notifications tepat 1, <code>logs/mail.log</code> tepat 1 baris. Log worker: 1 <code>already_sent: false</code> dan 5 <code>already_sent: true</code>.</td>
      <td><span class="badge badge-pass">PASS</span></td>
    </tr>
    <tr>
      <td><strong>C.6 Graceful Shutdown</strong></td>
      <td><code>MAILER_DELAY_MS=5000</code>, daftar ke kelas 9, kirim SIGINT/Ctrl+C saat pengiriman berjalan.</td>
      <td>Worker menuntaskan in-flight task, mencatat <code>notification_processed</code>, lalu proses berhenti rapi tanpa memutus koneksi.</td>
      <td><span class="badge badge-pass">PASS</span></td>
    </tr>
  </tbody>
</table>

<h3>2. Percobaan Pembanding Terkontrol (Controlled Counter-Experiment)</h3>
<p>
  Untuk membuktikan keabsahan alat uji, <strong>Bug 6</strong> (kesalahan verb format pada <code>SkipRetry</code>) dikembalikan ke kode:
</p>

<h4>Kondisi A: Kode Salah (Bug 6 Dikembalikan)</h4>
<pre><code>// app/service/notification_worker.go
if errors.Is(err, helper.ErrPermanentDelivery) {
    return fmt.Errorf("%v: %w", asynq.SkipRetry, err) // Verb %v menyebabkan SkipRetry tidak di-wrap
}</code></pre>
<p>
  <strong>Hasil Eksekusi Uji C.4:</strong><br>
  Compiler <code>go build ./...</code> lolos tanpa peringatan (membuktikan bug lolos compiler). Namun saat pengujian dijalankan:
</p>
<pre><code>Kondisi akhir /admin/queue: retry: 1, archived: 1 (archived tidak bertambah, retry bertambah)
[Log Worker]:
{"level":"WARN","msg":"task_failed","attempt":"1/6","error":"skip retry for the task:..."}
{"level":"WARN","msg":"task_failed","attempt":"2/6","error":"skip retry for the task:..."}
{"level":"WARN","msg":"task_failed","attempt":"3/6","error":"skip retry for the task:..."}</code></pre>
<p><em>Kesimpulan Kondisi A:</em> Task terus di-retry oleh Asynq karena <code>errors.Is(err, asynq.SkipRetry)</code> bernilai <code>false</code>. Pengujian C.4 <strong>berhasil mendeteksi kesalahan tersebut</strong>.</p>

<h4>Kondisi B: Kode Benar (Perbaikan Diterapkan)</h4>
<pre><code>// app/service/notification_worker.go
if errors.Is(err, helper.ErrPermanentDelivery) {
    return fmt.Errorf("%w: %v", asynq.SkipRetry, err) // Verb %w membungkus SkipRetry dengan benar
}</code></pre>
<p>
  <strong>Hasil Eksekusi Uji C.4:</strong>
</p>
<pre><code>Kondisi akhir /admin/queue: retry: 0, archived: 1 (archived bertambah tepat 1, retry bernilai 0)
[Log Worker]:
{"level":"WARN","msg":"task_failed","attempt":"1/6","error":"skip retry for the task: pengiriman gagal permanen: alamat salah@kampus.invalid tidak ditemukan"}
{"level":"WARN","msg":"Retry exhausted for task id=outbox-21","component":"asynq"}</code></pre>
<p><em>Kesimpulan Kondisi B:</em> Asynq langsung mengenali <code>SkipRetry</code> pada attempt 1/6 dan memindahkan task langsung ke antrean <code>archived</code> tanpa retry tambahan.</p>

<div class="page-break"></div>

<h2>BAGIAN D.4 — ANALISIS SINGKAT</h2>

<h3>Pertanyaan 1: Dua Skenario Masalah pada Usulan Enqueue Langsung Setelah Commit dengan Retry di Handler</h3>
<ol>
  <li>
    <strong>Skenario Kehilangan Notifikasi (Dual Write Breakdown):</strong><br>
    Ketika PostgreSQL selesai melakukan commit transaksi KRS, data pendaftaran telah resmi tersimpan di database. Jika tepat pada detik berikutnya server Redis mengalami crash, kehabisan memori, atau jaringan terputus, handler mencoba melakukan enqueue sebanyak 3 kali dan seluruhnya gagal. Karena tidak ada media persistensi lain tempat menyimpan task tersebut, notifikasi pendaftaran hilang permanen (*lost notification*). Hal yang sama terjadi jika proses web server mati mendadak (*crash/SIGKILL*) tepat setelah commit sebelum baris enqueue dieksekusi.
  </li>
  <li>
    <strong>Skenario Pembengkakan Waktu Respons Mahasiswa (Latency & Resource Starvation):</strong><br>
    Jika Redis mengalami beban tinggi atau degradasi, setiap percobaan enqueue ke Redis akan menunggu hingga batas timeout (misal 2 detik per percobaan). Melakukan 3 kali retry di dalam handler HTTP memaksa browser mahasiswa menggantung (*blocking*) selama lebih dari 6 detik hanya untuk menunggu pengiriman email yang sebenarnya bukan syarat keabsahan KRS. Jika ribuan mahasiswa serentak mengakses portal, thread worker Fiber akan habis tertahan, menyebabkan portal web down (*cascading failure*).
  </li>
</ol>

<h3>Pertanyaan 2: Redis Restart Tanpa Persistence Saat 100 Task Pending</h3>
<ol>
  <li>
    <strong>Apa yang terjadi pada 100 notifikasi tersebut?</strong><br>
    Ke-100 notifikasi tersebut hilang dari memori RAM Redis dan tidak akan pernah diproses oleh worker. Di sisi PostgreSQL, tabel <code>outbox</code> sudah menandai <code>published_at IS NOT NULL</code>, sehingga Outbox Relay tidak akan mengambilnya kembali.
  </li>
  <li>
    <strong>Deteksi Masalah Menggunakan Data PostgreSQL:</strong><br>
    Membandingkan event outbox yang sudah berstatus <code>published_at IS NOT NULL</code> tetapi tidak memiliki baris padanannya di tabel <code>notifications</code> setelah jeda waktu tertentu (misal lebih dari 5 menit).
  </li>
  <li>
    <strong>Query Deteksi:</strong>
    <pre><code>SELECT o.id, o.event_type, o.payload, o.created_at, o.published_at
FROM outbox o
LEFT JOIN notifications n ON n.outbox_id = o.id
WHERE o.published_at IS NOT NULL
  AND n.id IS NULL
  AND o.published_at < NOW() - INTERVAL '5 minutes'
ORDER BY o.id ASC;</code></pre>
  </li>
  <li>
    <strong>Pemulihan Event:</strong>
    <pre><code>UPDATE outbox
SET published_at = NULL
WHERE id IN (
    SELECT o.id FROM outbox o
    LEFT JOIN notifications n ON n.outbox_id = o.id
    WHERE o.published_at IS NOT NULL
      AND n.id IS NULL
      AND o.published_at < NOW() - INTERVAL '5 minutes'
);</code></pre>
    Dengan mengembalikan <code>published_at</code> menjadi <code>NULL</code>, Outbox Relay pada putaran berikutnya akan otomatis menyeleksi kembali baris tersebut dan memasukkannya ulang ke Redis.
  </li>
  <li>
    <strong>Pencegahan Email Ganda:</strong><br>
    Constraint <code>UNIQUE (outbox_id, user_id)</code> pada tabel <code>notifications</code> dan mekanisme <code>Claim</code> dengan klausa <code>ON CONFLICT DO NOTHING</code> menjamin bahwa jika ada task yang sebenarnya sudah sempat terkirim sebelum Redis mati, worker akan mendeteksi baris claim sudah ada (<code>already_sent = true</code>) dan langsung melewati pengiriman email.
  </li>
</ol>

<h3>Pertanyaan 3: Perbedaan Penggunaan `WithTxOnce` pada Worker vs `WithTx` pada CourseService</h3>
<ul>
  <li>
    <strong>CourseService menggunakan `WithTx`:</strong> Seluruh operasi dalam pendaftaran KRS adalah mutasi database internal PostgreSQL. Jika terjadi deadlock konkurensi atau serializable error, transaksi aman untuk diulang otomatis dari awal karena rollback database membersihkan seluruh efek samping tanpa jejak ke sistem luar.
  </li>
  <li>
    <strong>Worker menggunakan `WithTxOnce`:</strong> Di dalam worker, kita melakukan interaksi dengan sistem eksternal, yaitu mengirim email via SMTP / Mailer. Email yang telah terkirim ke jaringan luar tidak dapat dibatalkan (non-rollbackable). Jika worker menggunakan <code>WithTx</code> yang melakukan retry otomatis saat terjadi deadlock database, fungsi pengiriman email akan dipanggil ulang, mengakibatkan mahasiswa menerima email duplikat. Tanggung jawab retry diserahkan sepenuhnya ke Asynq dengan delay backoff terukur.
  </li>
  <li>
    <strong>Risiko Jika Ditukar:</strong> Jika worker memakai <code>WithTx</code>, mahasiswa berisiko menerima spam email konfirmasi berulang kali saat terjadi deadlock database. Jika <code>CourseService</code> memakai <code>WithTxOnce</code>, perebutan kursi kuliah antar mahasiswa akan langsung gagal tanpa retry, menurunkan keberhasilan transaksi mahasiswa saat sistem sibuk.
  </li>
</ul>

<h3>Pertanyaan 4: Urutan Email Tidak Dijamin pada Sistem Asinkron</h3>
<ol>
  <li>
    <strong>Kapan Berdampak Nyata bagi Pengguna?</strong><br>
    Berdampak nyata ketika status pesan bersifat ketergantungan urutan (*state sequence dependency*). Misalnya mahasiswa mendaftar kelas lalu segera membatalkannya. Jika email pembatalan tiba lebih awal daripada email pendaftaran, mahasiswa dapat salah menyimpulkan bahwa ia saat ini terdaftar di kelas tersebut, padahal pembatalannya telah diproses di database.
  </li>
  <li>
    <strong>Mengapa Urutan Task Berbeda dengan Urutan Email?</strong><br>
    Karena worker bekerja secara multi-threaded/konkruen (<code>WORKER_CONCURRENCY > 1</code>). Task daftar dan task keluar dieksekusi secara paralel oleh dua goroutine terpisah. Jika worker yang memproses task pendaftaran mengalami latensi jaringan SMTP atau terkena temporary retry, worker yang mengerjakan task keluar dapat menyelesaikan pengiriman lebih cepat.
  </li>
  <li>
    <strong>Solusi yang Masuk Akal:</strong><br>
    Menyertakan nomor urut event / sequence ID (misal nilai <code>outbox_id</code> atau timestamp presisi) pada subjek dan isi email (misal: *"Status KRS Kelas IF102 [Revisi #4 - 18:49:33]"*). Di sisi web/mobile app, endpoint <code>/notifications/mine</code> selalu mengurutkan berdasarkan <code>outbox_id DESC</code>, bukan urutan pencatatan notifikasi.
  </li>
  <li>
    <strong>Konsekuensi Solusi Terhadap Performa:</strong><br>
    Jika sistem dipaksa memproses strictly serial per mahasiswa menggunakan *single partition queue* atau *per-student mutex*, throughput pemrosesan background job akan menurun drastis dan kompleksitas arsitektur broker (seperti manajemen partisi Kafka atau antrean dinamis Redis) meningkat secara signifikan.
  </li>
</ol>

</body>
</html>
"""

html_path = os.path.abspath("docs/laporan_tm11_styled.html")
pdf_path = os.path.abspath("docs/Tugas11_434241102.pdf")

with open(html_path, "w", encoding="utf-8") as f:
    f.write(html_content)

print(f"HTML saved to {html_path}")

edge_path = r"C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe"
if not os.path.exists(edge_path):
    edge_path = r"C:\Program Files\Microsoft\Edge\Application\msedge.exe"

cmd = [
    edge_path,
    "--headless",
    "--disable-gpu",
    "--no-pdf-header-footer",
    f"--print-to-pdf={pdf_path}",
    html_path
]

print("Converting HTML to PDF via Edge Headless...")
subprocess.run(cmd, check=True)

if os.path.exists(pdf_path):
    size = os.path.getsize(pdf_path)
    print(f"PDF successfully generated: {pdf_path} (Size: {size} bytes)")
else:
    print("Failed to generate PDF")
