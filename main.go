package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"text/template"

	firebase "firebase.google.com/go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func recipePage() echo.HandlerFunc {
	return func(c echo.Context) error {
		recipeID := c.Param("recipeId")
		message := fmt.Sprintf("レシピID:%s", recipeID)
		return c.String(http.StatusOK, message)
	}
}

func postPage() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, "recipe", "world")
	}
}

func main() {
	// Firebaseのセットアップ
	// Use the application default credentials
	app, err := firebase.NewApp(context.Background(), nil)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	// テンプレートの読み込み
	t := &Template{
		templates: template.Must(template.ParseGlob("view/*.html")),
	}

	e := echo.New()
	e.Server.Addr = "127.0.0.1"
	e.Renderer = t

	// 全てのリクエストで差し込みたいミドルウェア（ログとか）はここ
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// ルーティング
	e.GET("/recipe/post", postPage())
	e.GET("/recipe/:recipeId", recipePage())

	// サーバー起動
	e.Start("localhost:9090") //ポート番号指定してね
}
