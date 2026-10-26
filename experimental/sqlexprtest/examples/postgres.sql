CREATE TABLE samples (host text PRIMARY KEY, bytes bigint, optional bigint);
INSERT INTO samples VALUES ('a', 9007199254740993, NULL), ('b', 42, 7);
