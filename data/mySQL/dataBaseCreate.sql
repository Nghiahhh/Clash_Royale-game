-- Tạo database nếu chưa tồn tại
CREATE DATABASE IF NOT EXISTS Clash_Royale;

-- Sử dụng database vừa tạo
USE Clash_Royale;

-- Tạo bảng users nếu chưa tồn tại
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    gmail VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(100) NOT NULL,
    password VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS user_stats (
    user_id INT PRIMARY KEY,             -- Khóa chính, liên kết đến bảng users
    level INT NOT NULL DEFAULT 1 CHECK (level >= 1),               -- Cấp độ
    experience INT NOT NULL DEFAULT 0 CHECK (experience >= 0),     -- Kinh nghiệm (>= 0, vì người mới có thể bắt đầu từ 0)
    gold INT NOT NULL DEFAULT 500 CHECK (gold >= 0),               -- Vàng
    gems INT NOT NULL DEFAULT 10 CHECK (gems >= 0),                -- Đá quý
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

