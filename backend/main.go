package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func run(args []string) error {
	app, err := newApp(env("DATA_DIR", "/data"), env("COOKIE_SECURE", "true") != "false")
	if err != nil {
		return err
	}
	defer app.db.Close()

	cmd := "serve"
	if len(args) > 0 {
		cmd = args[0]
	}
	switch {
	case cmd == "serve":
		addr := env("ADDR", ":8080")
		slog.Info("listening", "addr", addr)
		return http.ListenAndServe(addr, app.routes())
	case cmd == "adduser" && len(args) == 3:
		fmt.Fprint(os.Stderr, "Senha: ")
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if err := createUser(app.db, args[1], args[2], strings.TrimRight(line, "\r\n")); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Usuário criado.")
		return nil
	case cmd == "backup" && len(args) == 2:
		return backup(app.db, args[1])
	}
	return fmt.Errorf("uso: viagens [serve | adduser <email> <nome> | backup <arquivo>]")
}
