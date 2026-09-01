package service

import (
	"testing"
	"latihan-sintaks/app/model"
)

func TestValidateCreate(t *testing.T) {
	req := model.CreateStudentRequest{
		Name: "", // Sengaja dikosongkan untuk memicu error
		NIM:  "434241102",
	}

	errs := ValidateCreate(req)
	if len(errs) == 0 {
		t.Error("harap error pada nama yang kosong, tetapi lolos")
	}
	if errs["name"] == "" {
		t.Error("pesan error untuk name tidak ditemukan")
	}
}

func TestValidateReplace(t *testing.T) {
	req := model.ReplaceStudentRequest{
		Name: "Mohammad Anhaz Abdilah",
		NIM:  "", // Sengaja dikosongkan
	}

	errs := ValidateReplace(req)
	if len(errs) == 0 {
		t.Error("harap error pada NIM yang kosong, tetapi lolos")
	}
}

func TestApplyPatch(t *testing.T) {
	initial := model.Student{ID: 1, Name: "Aan", NIM: "434241102", IsActive: true}
	newName := "AanFLenrir"

	result, errs := ApplyPatch(initial, model.PatchStudentRequest{Name: &newName})

	if len(errs) != 0 {
		t.Fatalf("tidak seharusnya ada error: %v", errs)
	}
	if result.Name != "AanFLenrir" {
		t.Error("nama seharusnya berubah menjadi AanFLenrir")
	}
	if result.NIM != "434241102" {
		t.Error("NIM yang tidak dikirim seharusnya tidak berubah")
	}
}