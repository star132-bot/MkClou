ALTER TABLE products DROP INDEX ft_name_tagline;
ALTER TABLE products DROP INDEX idx_status_category_published, DROP INDEX idx_status_published;
ALTER TABLE products DROP COLUMN category;
