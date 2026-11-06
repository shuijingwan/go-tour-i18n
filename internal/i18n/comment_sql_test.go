package i18n

import "testing"

func TestCommentResidualSQL(t *testing.T) {
	for _, s := range []string{"SELECT * FROM user WHERE id = 1 OR 1=1;", "SELECT '-- literal' FROM user;", "DROP TABLE IF EXISTS album; CREATE TABLE album (id INT AUTO_INCREMENT NOT NULL, title VARCHAR(128) NOT NULL, price DECIMAL(5,2) NOT NULL, PRIMARY KEY (`id`)); INSERT INTO album (title,price) VALUES ('It''s -- literal #',1.00),('x',2.00);"} {
		if !SQLCommentFreeProof(s) {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"SELECT * FROM user; -- Teaching comment.", "SELECT * FROM user; # Teaching comment.", "SELECT * FROM user; /* comment */", "SELECT 'unterminated FROM user;", "SELECT * FROM user; unknown syntax", "unknown language"} {
		if SQLCommentFreeProof(s) {
			t.Fatal("unproved SQL", s)
		}
	}
}
