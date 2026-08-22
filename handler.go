package main

import (
	"sort"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Data awal agar API langsung bisa dites
var students = []Student{
	{ID: 1, NIM: "434241102", Name: "Mohammad Anhaz Abdilah", Grade: 3.75, IsActive: true},
}
var nextID = 2

func findStudentIndex(id int) int {
	for i := range students {
		if students[i].ID == id {
			return i
		}
	}
	return -1
}

func paramID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

func listStudents(c *fiber.Ctx) error {
	q := parseListQuery(c)

	// 1) Saring berdasarkan IsActive dan pencarian nama (case-insensitive)
	hasil := []Student{}
	for _, s := range students {
		if q.IsActive != nil && s.IsActive != *q.IsActive {
			continue
		}
		if q.Search != "" && !strings.Contains(strings.ToLower(s.Name), strings.ToLower(q.Search)) {
			continue
		}
		hasil = append(hasil, s)
	}

	// 2) Urutkan
	sort.SliceStable(hasil, func(i, j int) bool {
		var lebihKecil bool
		switch q.Sort {
		case "nim":
			lebihKecil = hasil[i].NIM < hasil[j].NIM
		case "name":
			lebihKecil = hasil[i].Name < hasil[j].Name
		case "grade":
			lebihKecil = hasil[i].Grade < hasil[j].Grade
		default:
			lebihKecil = hasil[i].ID < hasil[j].ID
		}
		if q.Order == "desc" {
			return !lebihKecil
		}
		return lebihKecil
	})

	// 3) Potong sesuai halaman
	total := len(hasil)
	totalPages := (total + q.Limit - 1) / q.Limit
	mulai := (q.Page - 1) * q.Limit
	if mulai > total { mulai = total }
	akhir := mulai + q.Limit
	if akhir > total { akhir = total }

	return okList(c, "daftar mahasiswa berhasil diambil", hasil[mulai:akhir], &Meta{
		Page: q.Page, Limit: q.Limit, Total: total, TotalPages: totalPages,
	})
}

func getStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	i := findStudentIndex(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan")
	}

	return ok(c, "mahasiswa ditemukan", students[i])
}

func createStudent(c *fiber.Ctx) error {
	var req CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	errs := map[string]string{}
	req.Name = strings.TrimSpace(req.Name)
	req.NIM = strings.TrimSpace(req.NIM)

	if req.Name == "" { errs["name"] = "wajib diisi" }
	if req.NIM == "" { errs["nim"] = "wajib diisi" }
	if req.Grade < 0 || req.Grade > 4 { errs["grade"] = "harus antara 0 dan 4" }

	// Pengecekan 409 Conflict: NIM tidak boleh ganda
	for _, s := range students {
		if s.NIM == req.NIM {
			return fail(c, fiber.StatusConflict, "NIM sudah terdaftar")
		}
	}

	if len(errs) > 0 { return failValidation(c, errs) }

	baru := Student{
		ID:       nextID,
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    req.Grade,
		IsActive: req.IsActive,
	}
	students = append(students, baru)
	nextID++

	return created(c, "mahasiswa berhasil ditambah", baru, "/api/v1/students/"+strconv.Itoa(baru.ID))
}

func replaceStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid { return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif") }

	i := findStudentIndex(id)
	if i == -1 { return fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan") }

	var req ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	errs := map[string]string{}
	if strings.TrimSpace(req.Name) == "" { errs["name"] = "wajib diisi pada PUT" }
	if strings.TrimSpace(req.NIM) == "" { errs["nim"] = "wajib diisi pada PUT" }
	if req.Grade < 0 || req.Grade > 4 { errs["grade"] = "harus antara 0 dan 4" }

	// Validasi NIM ganda (kecuali milik sendiri)
	for _, s := range students {
		if s.NIM == req.NIM && s.ID != id {
			return fail(c, fiber.StatusConflict, "NIM sudah dipakai mahasiswa lain")
		}
	}

	if len(errs) > 0 { return failValidation(c, errs) }

	students[i].NIM = req.NIM
	students[i].Name = req.Name
	students[i].Grade = req.Grade
	students[i].IsActive = req.IsActive

	return ok(c, "mahasiswa berhasil diganti seluruhnya", students[i])
}

func patchStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid { return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif") }

	i := findStudentIndex(id)
	if i == -1 { return fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan") }

	var req PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	if req.Name == nil && req.NIM == nil && req.Grade == nil && req.IsActive == nil {
		return fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
	}

	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return failValidation(c, map[string]string{"name": "tidak boleh kosong"})
		}
		students[i].Name = *req.Name
	}
	
	if req.NIM != nil {
		nim := strings.TrimSpace(*req.NIM)
		if nim == "" { return failValidation(c, map[string]string{"nim": "tidak boleh kosong"}) }
		for _, s := range students {
			if s.NIM == nim && s.ID != id {
				return fail(c, fiber.StatusConflict, "NIM sudah dipakai mahasiswa lain")
			}
		}
		students[i].NIM = nim
	}

	if req.Grade != nil {
		if *req.Grade < 0 || *req.Grade > 4 {
			return failValidation(c, map[string]string{"grade": "harus antara 0 dan 4"})
		}
		students[i].Grade = *req.Grade
	}

	if req.IsActive != nil { students[i].IsActive = *req.IsActive }

	return ok(c, "mahasiswa berhasil diperbarui sebagian", students[i])
}

func deleteStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid { return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif") }

	i := findStudentIndex(id)
	if i == -1 { return fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan") }

	students = append(students[:i], students[i+1:]...)

	return noContent(c)
}