package main

import (
	"log"
	"os"

	"github.com/woailv/do"

	"github.com/gin-gonic/gin"
)

type A struct {
	Name string `form:"name" json:"name"`
}

var logger = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)

func main() {
	do.Get("/", func(ctx *do.Context) string {
		return "<h1>123</h1>"
	})
	do.Post("/", func(ctx *do.Context) interface{} {
		a := new(A)
		// if err := ctx.Json2(a); err != nil {
		// 	return err.Error()
		// }
		if err := ctx.FormData2(a); err != nil {
			return err.Error()
		}
		logger.Println(a)

		return a
	})
	do.Run(":9000")
}

func a(ctx *gin.Context) {

}
