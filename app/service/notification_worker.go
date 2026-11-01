package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

// NotificationWorker mengerjakan task "notification:send".
//
// Bagi worker, Asynq adalah framework-nya — sama seperti Fiber bagi API.
// Karena itu method di sini menerima *asynq.Task, seperti method service
// lain menerima *fiber.Ctx.
type NotificationWorker struct {
	users         repository.UserRepository
	notifications repository.NotificationRepository
	courses       repository.CourseRepository
	tx            *repository.TxManager
	mailer        helper.Mailer
	logger        *slog.Logger
}

func NewNotificationWorker(
	users repository.UserRepository,
	notifications repository.NotificationRepository,
	courses repository.CourseRepository,
	tx *repository.TxManager,
	mailer helper.Mailer,
	logger *slog.Logger,
) *NotificationWorker {
	return &NotificationWorker{users: users, notifications: notifications,
		courses: courses, tx: tx, mailer: mailer, logger: logger}
}

// HandleSend mengembalikan error untuk meminta Asynq melakukan retry nanti,
// atau error yang membungkus asynq.SkipRetry untuk menyatakan "jangan di-retry,
// task ini tidak mungkin berhasil".
//
// Asynq menjamin setiap task dijalankan PALING SEDIKIT satu kali, bukan
// TEPAT satu kali. Task yang sama bisa datang dua kali, misalnya bila
// worker mati setelah mengirim email tetapi sebelum sempat melapor ke Redis.
// Karena itu handler ini harus idempoten: menjalankannya dua kali untuk
// task yang sama harus berakibat sama dengan menjalankannya sekali.
//
// PERBAIKAN BUG 5: Parameter t diubah dari asynq.Task menjadi *asynq.Task (pointer).
// asynq.HandlerFunc membutuhkan signature func(context.Context, *asynq.Task) error.
func (w *NotificationWorker) HandleSend(ctx context.Context, t *asynq.Task) error {
	var job model.NotificationJob
	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("%w: payload task rusak: %v", asynq.SkipRetry, err)
	}

	// Tugas Mandiri D.2: Event perubahan kuota mengirim notifikasi ke banyak mahasiswa (fan-out)
	if job.EventType == model.EventCourseQuotaChanged {
		return w.handleQuotaChanged(ctx, job)
	}

	var event model.EnrollmentEvent
	if err := json.Unmarshal(job.Payload, &event); err != nil {
		return fmt.Errorf("%w: payload event %d rusak: %v", asynq.SkipRetry, job.OutboxID, err)
	}

	user, err := w.users.FindByID(ctx, event.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		// Akunnya sudah dihapus. Melakukan retry tidak akan memunculkannya.
		return fmt.Errorf("%w: user %d tidak ditemukan", asynq.SkipRetry, event.UserID)
	}
	if err != nil {
		return err // database sedang bermasalah: layak di-retry
	}

	msg, err := BuildMessage(job.EventType, event, user)
	if err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	alreadySent := false
	err = w.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
		// Claim dan pengiriman berada di dalam SATU transaction:
		// - pengiriman gagal -> claim ikut dibatalkan -> attempt
		// berikutnya masih dapat mengirim;
		// - pengiriman sukses -> claim di-commit -> attempt berikutnya
		// melihat claim ini dan tidak mengirim lagi.
		//
		// PERBAIKAN BUG 7: Gunakan w.notifications.WithTx(tx), BUKAN w.notifications langsung!
		// Jika dipanggil tanpa WithTx(tx), Claim dieksekusi di luar transaksi (autocommit).
		// Akibatnya, saat mailer.Send gagal sementara, baris claim sudah terlanjur tersimpan di database.
		// Pada attempt retry berikutnya, Claim mengembalikan false dan email tidak pernah dikirim ulang!
		claimed, err := w.notifications.WithTx(tx).Claim(ctx, model.Notification{
			OutboxID: job.OutboxID, UserID: user.ID, Channel: "email",
			Subject: msg.Subject, Body: msg.Body,
		})
		if err != nil {
			return err
		}
		if !claimed {
			alreadySent = true
			return nil
		}
		return w.mailer.Send(ctx, msg)
	})

	if errors.Is(err, helper.ErrPermanentDelivery) {
		// PERBAIKAN BUG 6: Gunakan %w untuk asynq.SkipRetry, BUKAN %v!
		// Pada kode modul asli: fmt.Errorf("%v: %w", asynq.SkipRetry, err)
		// Karena asynq.SkipRetry diformat dengan %v (string), errors.Is(err, asynq.SkipRetry)
		// bernilai false. Asynq tidak mengenali SkipRetry dan tetap me-retry task 5 kali
		// alih-alih langsung mengarsipkan (archived) pada attempt 1.
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}
	if err != nil {
		return err
	}

	retry, _ := asynq.GetRetryCount(ctx)
	w.logger.Info("notification_processed",
		slog.Int64("outbox_id", job.OutboxID),
		slog.String("event", job.EventType),
		slog.Int("retry", retry),
		slog.Bool("already_sent", alreadySent))
	return nil
}

// handleQuotaChanged memproses event perubahan kuota kelas (Tugas Mandiri D.2).
// Mengirimkan email notifikasi ke setiap peserta kelas secara atomik dan idempoten per mahasiswa.
func (w *NotificationWorker) handleQuotaChanged(ctx context.Context, job model.NotificationJob) error {
	var event model.CourseQuotaChangedEvent
	if err := json.Unmarshal(job.Payload, &event); err != nil {
		return fmt.Errorf("%w: payload event quota %d rusak: %v", asynq.SkipRetry, job.OutboxID, err)
	}

	students, err := w.courses.FindEnrolledStudents(ctx, event.CourseID)
	if err != nil {
		return err // database error, layak di-retry oleh Asynq
	}

	retry, _ := asynq.GetRetryCount(ctx)
	for _, student := range students {
		msg := BuildQuotaMessage(event, student)
		alreadySent := false

		err := w.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
			// Menggunakan UNIQUE (outbox_id, user_id) dari migration 009
			claimed, err := w.notifications.WithTx(tx).Claim(ctx, model.Notification{
				OutboxID: job.OutboxID,
				UserID:   student.ID,
				Channel:  "email",
				Subject:  msg.Subject,
				Body:     msg.Body,
			})
			if err != nil {
				return err
			}
			if !claimed {
				alreadySent = true
				return nil
			}
			return w.mailer.Send(ctx, msg)
		})

		if errors.Is(err, helper.ErrPermanentDelivery) {
			// Jika satu mahasiswa memiliki alamat email tidak valid (.invalid),
			// catat log tapi jangan menggagalkan notifikasi mahasiswa lainnya.
			w.logger.Warn("quota_notification_permanent_failure",
				slog.Int("user_id", student.ID),
				slog.String("email", student.Email),
				slog.String("error", err.Error()))
			continue
		}
		if err != nil {
			// Temporary delivery failure: kembalikan error agar Asynq me-retry task.
			// Mahasiswa yang sudah sukses (already claimed) TIDAK akan menerima email ganda saat retry!
			return err
		}

		w.logger.Info("notification_processed",
			slog.Int64("outbox_id", job.OutboxID),
			slog.String("event", job.EventType),
			slog.Int("user_id", student.ID),
			slog.Int("retry", retry),
			slog.Bool("already_sent", alreadySent))
	}
	return nil
}
