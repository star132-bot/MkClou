// Package migrations 内嵌数据库迁移脚本，随二进制一起发布。
// 文件命名：{版本号}_{描述}.up.sql / .down.sql，版本号递增，已发布的脚本不允许修改。
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
