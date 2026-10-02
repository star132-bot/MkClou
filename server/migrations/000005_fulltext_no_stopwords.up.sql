-- ngram 分词会丢弃包含停用词（如 a、i）的 2 字词元，导致“AI”“UI”等常见词搜不到。
-- 停用词设置在建索引时生效，这里关闭停用词后重建全文索引。
SET SESSION innodb_ft_enable_stopword = OFF;
ALTER TABLE products DROP INDEX ft_name_tagline;
ALTER TABLE products ADD FULLTEXT KEY ft_name_tagline (name, tagline) WITH PARSER ngram;
