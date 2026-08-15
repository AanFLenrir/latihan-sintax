package main

import (
	"fmt"
)

// =========================
// 3. POINTER
// =========================

// Menukar dua nilai integer menggunakan pointer
func swap(a, b *int) {
	*a, *b = *b, *a
}

// Menambahkan item baru ke slice menggunakan pointer
func updateSlice(s *[]string, newItem string) {
	*s = append(*s, newItem)
}

// =========================
// 4. STRUCT STUDENT
// =========================

type Student struct {
	ID       int
	Name     string
	Grade    float64
	IsActive bool
}

// Value receiver karena method hanya membaca data
func (s Student) GetInfo() string {
	return fmt.Sprintf(
		"ID: %d, Name: %s, Grade: %.2f, Active: %t",
		s.ID, s.Name, s.Grade, s.IsActive,
	)
}

// Pointer receiver karena method mengubah Grade
func (s *Student) UpdateGrade(grade float64) {
	s.Grade = grade
}

// Pointer receiver karena mengubah status IsActive
func (s *Student) Activate() {
	s.IsActive = true
}

func (s *Student) Deactivate() {
	s.IsActive = false
}

func main() {

	// =========================
	// 2. VARIABEL
	// =========================

	var nama string = "Anhaz"
	var umur int = 21
	var ipk float64 = 3.75
	var aktif bool = true
	var buah []string = []string{"Apel", "Mangga", "Jeruk"}

	fmt.Println("=== VARIABEL ===")
	fmt.Println("Nama  :", nama)
	fmt.Println("Umur  :", umur)
	fmt.Println("IPK   :", ipk)
	fmt.Println("Aktif :", aktif)
	fmt.Println("Buah  :", buah)

	// =========================
	// MAP MAHASISWA
	// =========================

	mahasiswa := map[string]string{
		"Anhaz": "Teknik Informatika",
		"Budi":  "Sistem Informasi",
		"Citra": "Teknik Komputer",
	}

	fmt.Println("\n=== MAP MAHASISWA ===")

	// Menambahkan data
	mahasiswa["Dina"] = "Informatika"

	fmt.Println("Setelah menambahkan Dina:")
	fmt.Println(mahasiswa)

	// Membaca dengan pengecekan keberadaan
	nilai, ada := mahasiswa["Anhaz"]

	if ada {
		fmt.Println("Anhaz ditemukan:", nilai)
	} else {
		fmt.Println("Anhaz tidak ditemukan")
	}

	// Menghapus data
	delete(mahasiswa, "Citra")

	fmt.Println("Setelah Citra dihapus:")
	fmt.Println(mahasiswa)

	// Menelusuri seluruh isi map
	fmt.Println("Seluruh isi map:")

	for nama, jurusan := range mahasiswa {
		fmt.Println(nama, "->", jurusan)
	}

	// =========================
	// 3. POINTER
	// =========================

	fmt.Println("\n=== POINTER ===")

	a := 10
	b := 20

	fmt.Println("Sebelum swap:")
	fmt.Println("a =", a)
	fmt.Println("b =", b)

	swap(&a, &b)

	fmt.Println("Sesudah swap:")
	fmt.Println("a =", a)
	fmt.Println("b =", b)

	// Update slice
	buah2 := []string{"Apel", "Mangga"}

	fmt.Println("\nSlice sebelum update:")
	fmt.Println(buah2)

	updateSlice(&buah2, "Jeruk")

	fmt.Println("Slice sesudah update:")
	fmt.Println(buah2)

	// =========================
	// PASS BY VALUE VS POINTER
	// =========================

	fmt.Println("\n=== PASS BY VALUE VS POINTER ===")

	x := 10

	// Pass by value
	changeValue := func(n int) {
		n = 100
	}

	changeValue(x)

	fmt.Println("Setelah pass by value:", x)

	// Pass by pointer
	changePointer := func(n *int) {
		*n = 100
	}

	changePointer(&x)

	fmt.Println("Setelah pass by pointer:", x)

	// =========================
	// 4. STRUCT STUDENT
	// =========================

	fmt.Println("\n=== STRUCT STUDENT ===")

	student := Student{
		ID:       101,
		Name:     "Anhaz",
		Grade:    85.5,
		IsActive: false,
	}

	fmt.Println("Data awal:")
	fmt.Println(student.GetInfo())

	// Update nilai
	student.UpdateGrade(90.5)

	fmt.Println("\nSetelah UpdateGrade:")
	fmt.Println(student.GetInfo())

	// Aktifkan student
	student.Activate()

	fmt.Println("\nSetelah Activate:")
	fmt.Println(student.GetInfo())

	// Nonaktifkan student
	student.Deactivate()

	fmt.Println("\nSetelah Deactivate:")
	fmt.Println(student.GetInfo())
}