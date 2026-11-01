package service

import "strings"

// NormalizeTechnologies membersihkan daftar teknologi:
// - Menghapus spasi di awal dan akhir
// - Mengabaikan string kosong
// - Menghapus duplikasi tanpa membedakan huruf besar/kecil
// - Menjaga urutan kemunculan pertama
func NormalizeTechnologies(techs []string) []string {
	result := make([]string, 0, len(techs))
	seen := make(map[string]struct{}, len(techs))

	for _, t := range techs {
		clean := strings.TrimSpace(t)
		if clean == "" {
			continue
		}
		lower := strings.ToLower(clean)
		if _, dup := seen[lower]; dup {
			continue
		}
		seen[lower] = struct{}{}
		result = append(result, clean)
	}

	return result
}

// CanAddProject memeriksa apakah portofolio masih dapat menerima proyek tambahan
// berdasarkan batas maksimal 20 proyek.
func CanAddProject(currentCount int) bool {
	return currentCount < 20
}
