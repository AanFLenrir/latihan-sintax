package service

import (
	"strings"
	"latihan-sintaks/app/model"
)

// File ini berisi business rules MURNI: tidak menyentuh fiber.Ctx,
// tidak menyentuh database, dan tidak tahu apa pun tentang HTTP.

// ValidateCreate memeriksa isi permintaan pembuatan data mahasiswa baru.
func ValidateCreate(req model.CreateStudentRequest) map[string]string {
	errs := map[string]string{}
	
	if strings.TrimSpace(req.Name) == "" {
		errs["name"] = "nama wajib diisi"
	}
	if strings.TrimSpace(req.NIM) == "" {
		errs["nim"] = "NIM wajib diisi"
	}
	
	return errs
}

// ValidateReplace memeriksa isi permintaan PUT (Ubah semua).
func ValidateReplace(req model.ReplaceStudentRequest) map[string]string {
	errs := map[string]string{}
	
	if strings.TrimSpace(req.Name) == "" {
		errs["name"] = "nama wajib diisi pada PUT"
	}
	if strings.TrimSpace(req.NIM) == "" {
		errs["nim"] = "NIM wajib diisi pada PUT"
	}
	
	return errs
}

// ApplyPatch menyalin field yang dikirim ke data yang sudah ada (PATCH).
func ApplyPatch(
	current model.Student, req model.PatchStudentRequest,
) (model.Student, map[string]string) {
	errs := map[string]string{}

	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			errs["name"] = "nama tidak boleh kosong"
		} else {
			current.Name = *req.Name
		}
	}

	if req.NIM != nil {
		if strings.TrimSpace(*req.NIM) == "" {
			errs["nim"] = "NIM tidak boleh kosong"
		} else {
			current.NIM = *req.NIM
		}
	}

	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}

	return current, errs
}

// IsEmptyPatch menandai permintaan PATCH yang tidak mengubah apa pun.
func IsEmptyPatch(req model.PatchStudentRequest) bool {
	return req.Name == nil && req.NIM == nil && req.IsActive == nil
}

// CountTotalPages membulatkan ke atas tanpa memakai bilangan pecahan.
func CountTotalPages(total, limit int) int {
	if limit <= 0 {
		return 0
	}
	return (total + limit - 1) / limit
}