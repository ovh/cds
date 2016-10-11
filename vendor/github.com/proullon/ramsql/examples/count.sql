CREATE TABLE account (id INT, email TEXT);
INSERT INTO account ('id', 'email') VALUES (2, 'bar@bar.com');
INSERT INTO account ('id', 'email') VALUES (1, 'foo@bar.com');
SELECT COUNT(*) FROM account WHERE 1;
SELECT COUNT(*) FROM account WHERE id = 2;
