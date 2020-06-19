CREATE DATABASE fibbage_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'newuser'@'localhost' IDENTIFIED BY 'password';
GRANT ALL PRIVILEGES ON fibbage_db.* TO 'newuser'@'localhost';
FLUSH PRIVILEGES;

