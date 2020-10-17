package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"text/template"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"google.golang.org/api/option"
)

type Recipe struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

type Repo struct {
	ctx    context.Context
	client *firestore.Client
}

func initRepo() (*Repo, func()) {
	ctx := context.Background()
	sa := option.WithCredentialsFile("./settings/serviceAccount.json")
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		log.Fatalln(err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	close := func() {
		client.Close()
	}

	return &Repo{client: client, ctx: ctx}, close
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func postPage(repo *Repo) echo.HandlerFunc {
	return func(c echo.Context) error {
		post := new(Recipe)
		c.Bind(post)
		repo.client.Collection("recipes").Add(repo.ctx, post)
		return c.JSON(http.StatusCreated, post)
	}
}

func createPage() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, "create", "")
	}
}

func recipePage(repo *Repo) echo.HandlerFunc {
	return func(c echo.Context) error {
		recipeID := c.Param("recipeId")
		dsnap, err := repo.client.Collection("recipes").Doc(recipeID).Get(repo.ctx)
		if err != nil {
			log.Fatal(err)
		}

		recipe := Recipe{}
		if err := dsnap.DataTo(&recipe); err != nil {
			log.Fatal(err)
		}

		return c.Render(http.StatusOK, "recipe", recipe)
	}
}

func main() {
	repo, close := initRepo()
	defer close()

	// テンプレートの読み込み
	t := &Template{
		templates: template.Must(template.ParseGlob("view/*.html")),
	}

	e := echo.New()
	e.Server.Addr = "127.0.0.1"
	e.Renderer = t

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/recipe/create/post", postPage(repo))
	e.GET("/recipe/create", createPage())
	e.GET("/recipe/:recipeId", recipePage(repo))

	e.Start("localhost:9090")
}
