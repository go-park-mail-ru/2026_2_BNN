-- Учётные записи для локальной проверки.
-- password_hash — bcrypt от строки password.

INSERT INTO "user" (
    id,
    login,
    password_hash
)
VALUES
    ('11111111-1111-4111-8111-111111111111', 'ada', '$2b$10$cubeQnGNTCV.tMHzTrN4v.BX9sHYhpfXAa4mxJ3Kw82JhXaXQD2oO'),
    ('22222222-2222-4222-8222-222222222222', 'grace', '$2b$10$cubeQnGNTCV.tMHzTrN4v.BX9sHYhpfXAa4mxJ3Kw82JhXaXQD2oO'),
    ('33333333-3333-4333-8333-333333333333', 'linus', '$2b$10$cubeQnGNTCV.tMHzTrN4v.BX9sHYhpfXAa4mxJ3Kw82JhXaXQD2oO');
