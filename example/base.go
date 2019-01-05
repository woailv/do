package main

import (
	"log"
	"os"

	"github.com/woailv/do"
)

type A struct {
	Name string
}

var logger = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)

func main() {
	do.Get("/", func(ctx *do.Context) string {
		logger.Println(ctx.Params)
		return "<h1>123</h1>"
	})
	do.Run(":9000")
}
