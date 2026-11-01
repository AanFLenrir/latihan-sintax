-- Migration 007: Dosen Pembimbing Tugas Akhir (Tugas Mandiri D.2)
-- psql -U postgres -d praktikum_backend -f migrations/007_supervisors.sql

CREATE TABLE IF NOT EXISTS supervisors (
    id SERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    quota INTEGER NOT NULL CHECK (quota > 0),
    supervised_count INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT supervisors_count_within_quota
        CHECK (supervised_count >= 0 AND supervised_count <= quota)
);

CREATE TABLE IF NOT EXISTS supervisions (
    id BIGSERIAL PRIMARY KEY,
    student_id INTEGER NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    supervisor_id INTEGER NOT NULL REFERENCES supervisors(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Aturan bisnis: seorang mahasiswa hanya boleh memiliki SATU pembimbing aktif
    CONSTRAINT supervisions_student_unique UNIQUE (student_id)
);

CREATE INDEX IF NOT EXISTS supervisions_supervisor_idx ON supervisions (supervisor_id);

-- Seed data dosen pembimbing
INSERT INTO supervisors (name, quota) VALUES
('Dr. Eng. Budi Santoso', 3),
('Dr. Siti Aminah, M.Kom', 3),
('Prof. Joko Widodo, Ph.D', 10)
ON CONFLICT DO NOTHING;

-- Seed permission baru untuk pengelolaan pembimbing
INSERT INTO permissions (name, description) VALUES
('supervision:manage', 'Mengelola penetapan dosen pembimbing')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
('admin', 'supervision:manage')
ON CONFLICT DO NOTHING;

-- Seed data mahasiswa uji tambahan pada tabel students agar cukup untuk uji beban 30 request
INSERT INTO students (nim, name, grade, is_active, owner_id)
SELECT 'NIM' || LPAD(g::TEXT, 5, '0'),
       'Mahasiswa ' || LPAD(g::TEXT, 3, '0'),
       3.50,
       true,
       u.id
FROM generate_series(1, 35) AS g
JOIN users u ON u.username = 'mhs' || LPAD(g::TEXT, 3, '0')
ON CONFLICT (nim) DO NOTHING;
