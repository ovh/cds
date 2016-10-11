CREATE TABLE account (id INT, email TEXT);

INSERT INTO account ('id', 'email') VALUES (1, 'foo@bar.com');
INSERT INTO account ('id', 'email') VALUES (2, 'bar@golang.org');
INSERT INTO account ('id', 'email') VALUES (3, 'titi@golang.org');

SELECT * FROM account WHERE 1;
SELECT * FROM account WHERE email = 'foo@bar.com';

