package main

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	// Pastikan nama modul ini sesuai dengan isi file go.mod Anda
	"latihan-sintaks/app/model"
	"latihan-sintaks/app/repository"
)

// reqCtx membuat context dengan timeout 5 detik untuk query ke database
func reqCtx(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Context(), 5*time.Second)
}

func paramID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

func parseListQuery(c *fiber.Ctx) model.ListQuery {
	q := model.ListQuery{
		Page:   c.QueryInt("page", 1),
		Limit:  c.QueryInt("limit", 10),
		Search: c.Query("search"),
		Sort:   c.Query("sort", "id"),
		Order:  c.Query("order", "asc"),
	}

	if q.Page < 1 { q.Page = 1 }
	if q.Limit < 1 { q.Limit = 10 }
	if q.Limit > 100 { q.Limit = 100 }

	if activeStr := c.Query("is_active"); activeStr != "" {
		isActive := activeStr == "true"
		q.IsActive = &isActive
	}

	return q
}

// --- FORMATTER RESPONS ---

func fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(model.WebResponse{
		Success: false,
		Message: message,
	})
}

func failValidation(c *fiber.Ctx, errs map[string]string) error {
	return c.Status(fiber.StatusBadRequest).JSON(model.WebResponse{
		Success: false,
		Message: "validasi gagal",
		Errors:  errs,
	})
}

func ok(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func okList(c *fiber.Ctx, message string, data any, meta *model.Meta) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

func created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location)
	return c.Status(fiber.StatusCreated).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func noContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// terjemahkanError mengubah error dari layer repository menjadi HTTP Status yang sesuai
func terjemahkanError(c *fiber.Ctx, err error, pesanDefault string) error {
	if errors.Is(err, repository.ErrNotFound) {
		return fail(c, fiber.StatusNotFound, "data tidak ditemukan")
	}
	if errors.Is(err, repository.ErrDuplicate) {
		return fail(c, fiber.StatusConflict, "data sudah ada atau duplikat")
	}
	// Jika error lain (seperti koneksi database putus), kembalikan 500
	return fail(c, fiber.StatusInternalServerError, pesanDefault)
}