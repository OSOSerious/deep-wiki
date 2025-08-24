CREATE TABLE IF NOT EXISTS products (
  id    SERIAL PRIMARY KEY,
  name  TEXT NOT NULL,
  price NUMERIC(10,2) NOT NULL,
  stock INT NOT NULL
);

INSERT INTO products (name, price, stock) VALUES
('Laptop', 999.99, 10),
('Phone', 499.99, 25);