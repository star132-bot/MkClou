SET SESSION innodb_ft_enable_stopword = ON;
ALTER TABLE products DROP INDEX ft_name_tagline;
ALTER TABLE products ADD FULLTEXT KEY ft_name_tagline (name, tagline) WITH PARSER ngram;
