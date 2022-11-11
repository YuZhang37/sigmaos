DROP TABLE IF EXISTS book;
CREATE TABLE book (
  id         INT AUTO_INCREMENT NOT NULL,
  title      VARCHAR(128) NOT NULL,
  author     VARCHAR(255) NOT NULL,
  price      DECIMAL(5,2) NOT NULL,
  PRIMARY KEY (`id`)
);

CREATE TABLE user (
  id         INT AUTO_INCREMENT NOT NULL,
  username   VARCHAR(128) NOT NULL,
  password   VARCHAR(255) NOT NULL,
  PRIMARY KEY (`id`)
);

INSERT INTO book
  (title, author, price) 
VALUES 
  ('Computer systems engineering', 'J. Saltzer', 56.99),
  ('Xv6', 'R. Morris', 63.99),
  ('Odyssey', 'Homer', 34.98);

