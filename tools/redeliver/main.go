package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hibiken/asynq"
	"latihan-fiber/app/model"
	"latihan-fiber/config"
	"latihan-fiber/database"
)

func main() {
	outboxID := flag.Int64("outbox", 0, "ID outbox yang akan di-redeliver")
	copies := flag.Int("copies", 5, "Jumlah salinan task yang akan di-enqueue")
	flag.Parse()

	if *outboxID <= 0 {
		fmt.Println("Gunakan: go run ./tools/redeliver -outbox <id> -copies <jumlah>")
		os.Exit(1)
	}

	config.LoadEnv()
	ctx := context.Background()

	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("gagal terhubung ke database: %v", err)
	}
	defer pool.Close()

	var eventType string
	var payload []byte
	err = pool.QueryRow(ctx, "SELECT event_type, payload FROM outbox WHERE id = $1", *outboxID).
		Scan(&eventType, &payload)
	if err != nil {
		log.Fatalf("gagal mengambil outbox event %d: %v", *outboxID, err)
	}

	job := model.NotificationJob{
		OutboxID:  *outboxID,
		EventType: eventType,
		Payload:   payload,
	}

	payloadBytes, err := json.Marshal(job)
	if err != nil {
		log.Fatalf("gagal menyusun payload: %v", err)
	}

	client := asynq.NewClient(config.RedisOpt())
	defer client.Close()

	for i := 1; i <= *copies; i++ {
		// Buat task salinan tanpa task ID statis (atau task ID unik) agar diterima Redis
		task := asynq.NewTask(
			model.TaskSendNotification,
			payloadBytes,
			asynq.Queue(model.QueueNotifications),
			asynq.MaxRetry(5),
			asynq.Timeout(30*time.Second),
			asynq.Retention(24*time.Hour),
		)

		info, err := client.Enqueue(task)
		if err != nil {
			log.Fatalf("gagal enqueue salinan ke-%d: %v", i, err)
		}
		fmt.Printf("salinan %d/%d ter-enqueue (task_id: %s)\n", i, *copies, info.ID)
	}

	fmt.Printf("berhasil menambahkan %d salinan task untuk outbox %d ke queue %s\n",
		*copies, *outboxID, model.QueueNotifications)
}
