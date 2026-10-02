package product

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSearchQuery(t *testing.T) {
	q := buildSearchQuery(SearchInput{Query: `  Figma +模板 -"注入" 字 `, Category: "design", Price: "0-50", Sort: "bogus", Page: 0})
	assert.Equal(t, []string{"Figma", "模板", "注入"}, q.terms, "去掉全文检索运算符，按空格拆词")
	assert.Equal(t, []string{"字"}, q.short, "单字无法命中 ngram 索引，改用 LIKE")
	assert.Equal(t, "design", q.category)
	assert.Equal(t, 1, *q.minPrice)
	assert.Equal(t, 5000, *q.maxPrice)
	assert.Equal(t, "default", q.sort)
	assert.Equal(t, 1, q.page)

	q = buildSearchQuery(SearchInput{Category: "food", Price: "free", Sort: "price_asc", Page: 3})
	assert.Empty(t, q.category, "未知分类被忽略")
	assert.Equal(t, 0, *q.minPrice)
	assert.Equal(t, 0, *q.maxPrice)
	assert.Equal(t, "price_asc", q.sort)
	assert.Equal(t, 3, q.page)
	assert.Nil(t, q.terms)

	q = buildSearchQuery(SearchInput{Query: strings.Repeat("长", 80), Price: "200+"})
	assert.Equal(t, maxQueryLen, len([]rune(q.raw)), "关键词截断为 50 个字符")
	assert.Equal(t, 20001, *q.minPrice)
	assert.Nil(t, q.maxPrice)
}
