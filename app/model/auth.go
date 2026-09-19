package model

import "time"

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Ubah nama menjadi TokenPair agar sesuai dengan kembalian AuthService
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // detik
}

// Ubah nama menjadi RefreshRequest agar sesuai dengan AuthService
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// BARU: RefreshToken adalah row pada table refresh_tokens di database
type RefreshToken struct {
	ID         int64
	UserID     int
	TokenHash  string
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}

// BARU & SANGAT PENTING: AuthUser adalah identitas yang dibawa access token
// Ini yang tadi dicari oleh fungsi CanAccessStudent dan bikin error merah!
type AuthUser struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// --- Tambahan dari kodemu (tetap dipertahankan jika dipakai di handler) ---

type UserResponse struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type UserInternal struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}