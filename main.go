package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"text/template"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"google.golang.org/api/iterator"
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
	fmt.Println(os.Getenv("$GOOGLE_APPLICATION_CREDENTIALS"))
	ctx := context.Background()
	// credentials, err := google.CredentialsFromJSON(ctx, []byte(os.Getenv("$GOOGLE_APPLICATION_CREDENTIALS")))
	// if err != nil {
	// 	log.Printf("error credentials from json: %v\n", err)
	// }
	// sa := option.WithCredentials(credentials)
	sa := option.WithCredentialsFile("./settings/serviceAccount.json")
	app, err := firebase.NewApp(ctx, nil, sa)
	// app, err := firebase.NewApp(ctx, nil)
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

func listPage(repo *Repo) echo.HandlerFunc {
	return func(c echo.Context) error {
		iter := repo.client.Collection("recipes").Documents(repo.ctx)
		recipes := []*Recipe{}
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			recipe := Recipe{}
			if err := doc.DataTo(&recipe); err != nil {
				log.Fatal(err)
			}
			recipes = append(recipes, &recipe)
		}
		return c.Render(http.StatusOK, "list", recipes)
	}
}

func tweetPage(repo *Repo) echo.HandlerFunc {
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

		return c.Render(http.StatusOK, "tweet", struct {
			Recipe Recipe
			ID     string
		}{
			Recipe: recipe,
			ID:     recipeID,
		})
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

	e.Static("/css", "./css")
	e.POST("/recipe/create/post", postPage(repo))
	e.GET("/recipe/create", createPage())
	e.GET("/recipe/list", listPage(repo))
	e.GET("/recipe/tweet/:recipeId", tweetPage(repo))
	e.GET("/recipe/:recipeId", recipePage(repo))

	e.Start(":9090")
}
