package config

import (
	"os"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// SetupLogger membuka atau membuat file log.
func SetupLogger() *os.File {
	file, err := os.OpenFile("./logs/app.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic("gagal membuka file log: " + err.Error())
	}
	return file
}

// LoggerConfig mengatur format log ke dalam JSON satu baris.
func LoggerConfig(file *os.File) logger.Config {
	return logger.Config{
		Format:     `{"time":"${time}","request_id":"${locals:requestid}","method":"${method}","path":"${path}","status":${status},"latency":"${latency}"}` + "\n",
		TimeFormat: "2006-01-02T15:04:05.000Z07:00",
		Output:     file,
	}
}