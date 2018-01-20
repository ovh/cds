-- the below is equal to 'DROP TABLE IF EXISTS'
DECLARE
    C INT;
BEGIN
    -- DROPPING TABLES IF EXISTS
    SELECT COUNT(*) INTO c FROM user_tables WHERE table_name = 'COMMENTS';
    IF c = 1 THEN
        EXECUTE IMMEDIATE 'DROP TABLE COMMENTS';
    END IF;

    SELECT COUNT(*) INTO c FROM user_tables WHERE table_name = 'POSTS_TAGS';
    IF c = 1 THEN
        EXECUTE IMMEDIATE 'DROP TABLE POSTS_TAGS';
    END IF;

    SELECT COUNT(*) INTO c FROM user_tables WHERE table_name = 'POSTS';
    IF c = 1 THEN
        EXECUTE IMMEDIATE 'DROP TABLE POSTS';
    END IF;

    SELECT COUNT(*) INTO c FROM user_tables WHERE table_name = 'TAGS';
    IF c = 1 THEN
        EXECUTE IMMEDIATE 'DROP TABLE TAGS';
    END IF;

    SELECT COUNT(*) INTO c FROM user_tables WHERE table_name = 'USERS';
    IF c = 1 THEN
        EXECUTE IMMEDIATE 'DROP TABLE USERS';
    END IF;

    -- DROPPING SEQUENCES IF EXISTS
    SELECT COUNT(*) INTO c FROM all_sequences WHERE sequence_name = 'POSTS_SEQ';
    IF c = 1 THEN
        EXECUTE IMMEDIATE 'DROP SEQUENCE POSTS_SEQ';
    END IF;

    SELECT COUNT(*) INTO c FROM all_sequences WHERE sequence_name = 'TAGS_SEQ';
    IF c = 1 THEN
        EXECUTE IMMEDIATE 'DROP SEQUENCE TAGS_SEQ';
    END IF;

    SELECT COUNT(*) INTO c FROM all_sequences WHERE sequence_name = 'COMMENTS_SEQ';
    IF c = 1 THEN
        EXECUTE IMMEDIATE 'DROP SEQUENCE COMMENTS_SEQ';
    END IF;

    SELECT COUNT(*) INTO c FROM all_sequences WHERE sequence_name = 'USERS_SEQ';
    IF c = 1 THEN
        EXECUTE IMMEDIATE 'DROP SEQUENCE USERS_SEQ';
    END IF;

    -- CREATING SQUEMA
    EXECUTE IMMEDIATE 'CREATE SEQUENCE posts_seq';
    EXECUTE IMMEDIATE 'CREATE SEQUENCE tags_seq';
    EXECUTE IMMEDIATE 'CREATE SEQUENCE comments_seq';
    EXECUTE IMMEDIATE 'CREATE SEQUENCE users_seq';

    EXECUTE IMMEDIATE 'CREATE TABLE posts (
    	id INTEGER PRIMARY KEY
    	,title VARCHAR(255) NOT NULL
    	,content VARCHAR2(4000) NOT NULL
    	,created_at TIMESTAMP NOT NULL
    	,updated_at TIMESTAMP NOT NULL
    )';

    EXECUTE IMMEDIATE 'CREATE TABLE tags (
    	id INTEGER PRIMARY KEY
    	,name VARCHAR(255) NOT NULL
    	,created_at TIMESTAMP NOT NULL
    	,updated_at TIMESTAMP NOT NULL
    )';

    EXECUTE IMMEDIATE 'CREATE TABLE posts_tags (
    	post_id INTEGER NOT NULL
    	,tag_id INTEGER NOT NULL
    	,PRIMARY KEY (post_id, tag_id)
    	,FOREIGN KEY (post_id) REFERENCES posts (id)
    	,FOREIGN KEY (tag_id) REFERENCES tags (id)
    )';

    EXECUTE IMMEDIATE 'CREATE TABLE comments (
    	id INTEGER PRIMARY KEY NOT NULL
    	,post_id INTEGER NOT NULL
    	,author_name VARCHAR(255) NOT NULL
    	,author_email VARCHAR(255) NOT NULL
    	,content VARCHAR2(4000) NOT NULL
    	,created_at TIMESTAMP NOT NULL
    	,updated_at TIMESTAMP NOT NULL
    	,FOREIGN KEY (post_id) REFERENCES posts (id)
    )';

    EXECUTE IMMEDIATE 'CREATE TABLE users (
        id INTEGER PRIMARY KEY NOT NULL
        ,attributes VARCHAR(255) NOT NULL
    )';
END;
