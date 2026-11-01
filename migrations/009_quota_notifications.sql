-- Migration 009: quota notifications composite idempotency key
-- psql -U postgres -d praktikum_backend -f migrations/009_quota_notifications.sql
-- ---------------------------------------------------------------------
-- Mengganti constraint UNIQUE (outbox_id) menjadi UNIQUE (outbox_id, user_id)
-- agar satu event (outbox_id) dapat menghasilkan banyak notifikasi ke seluruh
-- peserta kelas (fan-out) secara aman dan idempoten per mahasiswa.
-- ---------------------------------------------------------------------

ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_outbox_unique;

ALTER TABLE notifications ADD CONSTRAINT notifications_outbox_user_unique UNIQUE (outbox_id, user_id);
