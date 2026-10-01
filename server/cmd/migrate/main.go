// 数据库迁移命令。
//
//	go run ./cmd/migrate up          执行全部未执行的迁移
//	go run ./cmd/migrate down 1      回滚最近 1 个迁移
//	go run ./cmd/migrate version     查看当前版本
//	go run ./cmd/migrate force <v>   迁移失败后手动标记版本（修复脏状态）
package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"mkclou/server/internal/pkg/config"
	"mkclou/server/migrations"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: migrate up | down <n> | version | force <version>")
	}

	cfg, err := config.Load("configs")
	if err != nil {
		return err
	}

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}
	dsn := "mysql://" + cfg.MySQL.DSN() + "&multiStatements=true"
	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer m.Close()

	switch args[0] {
	case "up":
		err = m.Up()
	case "down":
		n, convErr := stepArg(args)
		if convErr != nil {
			return convErr
		}
		err = m.Steps(-n)
	case "force":
		v, convErr := stepArg(args)
		if convErr != nil {
			return convErr
		}
		err = m.Force(v)
	case "version":
		// 仅输出版本
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	v, dirty, verr := m.Version()
	switch {
	case errors.Is(verr, migrate.ErrNilVersion):
		fmt.Println("version: none")
	case verr != nil:
		return verr
	default:
		fmt.Printf("version: %d, dirty: %v\n", v, dirty)
	}
	return nil
}

func stepArg(args []string) (int, error) {
	if len(args) < 2 {
		return 0, fmt.Errorf("%s requires a number", args[0])
	}
	n, err := strconv.Atoi(args[1])
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid number %q", args[1])
	}
	return n, nil
}
