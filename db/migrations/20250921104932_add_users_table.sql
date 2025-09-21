-- +goose Up
-- Этот блок выполняется, когда мы накатываем миграцию (goose up)

-- Шаг 1: Создаем таблицу для пользователей
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    api_key VARCHAR(64) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Шаг 2: Добавляем колонку user_id в существующую таблицу tunnels
ALTER TABLE tunnels ADD COLUMN user_id UUID;

-- Шаг 3: Создаем внешний ключ (foreign key), чтобы связать таблицы
-- Это гарантирует, что в tunnels.user_id может быть только ID существующего пользователя
ALTER TABLE tunnels 
ADD CONSTRAINT fk_user
FOREIGN KEY (user_id) 
REFERENCES users(id) 
ON DELETE CASCADE; -- Если удалить пользователя, все его туннели удалятся автоматически

-- +goose Down
-- Этот блок выполняется, когда мы откатываем миграцию (goose down)

-- Выполняем действия в обратном порядке

-- Шаг 1: Удаляем внешний ключ и колонку из таблицы tunnels
ALTER TABLE tunnels DROP CONSTRAINT fk_user;
ALTER TABLE tunnels DROP COLUMN user_id;

-- Шаг 2: Удаляем таблицу users
DROP TABLE IF EXISTS users;