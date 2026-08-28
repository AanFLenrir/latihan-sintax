package main

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	
	// Pastikan nama modulnya sudah sesuai dengan go.mod Anda
	"latihan-sintaks/app/model"
	"latihan-sintaks/app/repository"
)

type StudentHandler struct {
	repo repository.StudentRepository
}

func NewStudentHandler(repo repository.StudentRepository) *StudentHandler {
	return &StudentHandler{repo: repo}
}

func (h *StudentHandler) List(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	q := parseListQuery(c)
	students, total, err := h.repo.FindAll(ctx, q)
	if err != nil {
		return fail(c, fiber.StatusInternalServerError, "gagal mengambil data mahasiswa")
	}

	totalPages := 0
	if q.Limit > 0 {
		totalPages = (total + q.Limit - 1) / q.Limit
	}

	return okList(c, "daftar mahasiswa berhasil diambil", students, &model.Meta{
		Page: q.Page, Limit: q.Limit, Total: total, TotalPages: totalPages,
	})
}

func (h *StudentHandler) Get(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	student, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(c, err, "gagal mengambil mahasiswa")
	}
	return ok(c, "mahasiswa ditemukan", student)
}

func (h *StudentHandler) Create(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	errs := map[string]string{}
	req.Name = strings.TrimSpace(req.Name)
	req.NIM = strings.TrimSpace(req.NIM)

	if req.Name == "" { errs["name"] = "wajib diisi" }
	if req.NIM == "" { errs["nim"] = "wajib diisi" }
	if req.Grade < 0 || req.Grade > 4 { errs["grade"] = "harus antara 0 dan 4" }

	if len(errs) > 0 {
		return failValidation(c, errs)
	}

	baru, err := h.repo.Create(ctx, model.Student{
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: req.IsActive,
	})
	if err != nil {
		return terjemahkanError(c, err, "gagal menyimpan mahasiswa")
	}
	return created(c, "mahasiswa berhasil ditambah", baru, "/api/v1/students/"+strconv.Itoa(baru.ID))
}

func (h *StudentHandler) Replace(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	var req model.ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	errs := map[string]string{}
	req.Name = strings.TrimSpace(req.Name)
	req.NIM = strings.TrimSpace(req.NIM)

	if req.Name == "" { errs["name"] = "wajib diisi pada PUT" }
	if req.NIM == "" { errs["nim"] = "wajib diisi pada PUT" }
	if req.Grade < 0 || req.Grade > 4 { errs["grade"] = "harus antara 0 dan 4" }

	if len(errs) > 0 { return failValidation(c, errs) }

	hasil, err := h.repo.Update(ctx, model.Student{
		ID: id, NIM: req.NIM, Name: req.Name, Grade: req.Grade, IsActive: req.IsActive,
	})
	if err != nil {
		return terjemahkanError(c, err, "gagal update mahasiswa")
	}
	return ok(c, "mahasiswa berhasil diganti seluruhnya", hasil)
}

func (h *StudentHandler) Patch(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	// 1. Tarik data yang ada di database sekarang
	studentLama, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return terjemahkanError(c, err, "gagal mengambil mahasiswa")
	}

	// 2. Timpa field yang dikirimkan (tidak nil)
	if req.Name != nil { studentLama.Name = strings.TrimSpace(*req.Name) }
	if req.NIM != nil { studentLama.NIM = strings.TrimSpace(*req.NIM) }
	if req.Grade != nil { studentLama.Grade = *req.Grade }
	if req.IsActive != nil { studentLama.IsActive = *req.IsActive }

	// 3. Simpan perubahannya
	hasil, err := h.repo.Update(ctx, studentLama)
	if err != nil {
		return terjemahkanError(c, err, "gagal memperbarui sebagian")
	}

	return ok(c, "mahasiswa berhasil diperbarui sebagian", hasil)
}

func (h *StudentHandler) Delete(c *fiber.Ctx) error {
	ctx, cancel := reqCtx(c)
	defer cancel()

	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	if err := h.repo.Delete(ctx, id); err != nil {
		return terjemahkanError(c, err, "gagal menghapus mahasiswa")
	}
	return noContent(c)
}