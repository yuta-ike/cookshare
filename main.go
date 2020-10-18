package main

import (
	"context"
	"encoding/base64"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type Recipe struct {
	Name        string `json:"name"`
	Message     string `json:"message"`
	Pic         string `json:"pic"`
	Difficulty  int    `json:"difficulty"`
	Ingredients string `json:"ingredients"`
	HowTo1      string `json:"howTo1"`
	HowTo2      string `json:"howTo2"`
	HowTo3      string `json:"howTo3"`
	HowTo4      string `json:"howTo4"`
	HowTo5      string `json:"howTo5"`
	HowTo6      string `json:"howTo6"`
}

type Repo struct {
	ctx    context.Context
	client *firestore.Client
}

func initRepo() (*Repo, func()) {
	ctx := context.Background()
	sa := option.WithCredentialsJSON([]byte(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	config := &firebase.Config{
		StorageBucket: "cookshare-kyoto.appspot.com",
	}
	app, err := firebase.NewApp(ctx, config, sa)
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

	return &Repo{client: client, ctx: ctx},
		close
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func postPage(repo *Repo) echo.HandlerFunc {
	return func(c echo.Context) error {
		file, _, err := c.Request().FormFile("pic_complete")
		if err != nil {
			return c.Render(http.StatusOK, "error", nil)
		}
		fileData, err := ioutil.ReadAll(file)
		if err != nil {
			return c.Render(http.StatusOK, "error", nil)
		}
		enc := base64.StdEncoding.EncodeToString(fileData)

		difficulty, err := strconv.Atoi(c.FormValue("difficulty"))
		if err != nil {
			return c.Render(http.StatusOK, "error", nil)
		}

		post := Recipe{
			Name:        c.FormValue("name"),
			Message:     strings.Replace(c.FormValue("introduction"), "\n", "<br/>", -1),
			Pic:         enc,
			Difficulty:  difficulty,
			Ingredients: strings.Replace(c.FormValue("ingredients"), "\n", "<br/>", -1),
			HowTo1:      strings.Replace(c.FormValue("howTo1"), "\n", "<br/>", -1),
			HowTo2:      strings.Replace(c.FormValue("howTo2"), "\n", "<br/>", -1),
			HowTo3:      strings.Replace(c.FormValue("howTo3"), "\n", "<br/>", -1),
			HowTo4:      strings.Replace(c.FormValue("howTo4"), "\n", "<br/>", -1),
			HowTo5:      strings.Replace(c.FormValue("howTo5"), "\n", "<br/>", -1),
			HowTo6:      strings.Replace(c.FormValue("howTo6"), "\n", "<br/>", -1),
		}

		docref, _, err := repo.client.Collection("recipes").Add(repo.ctx, post)
		if err != nil {
			return c.Render(http.StatusOK, "error", nil)
		}

		return c.Redirect(http.StatusPermanentRedirect, "/recipe/tweet/"+docref.ID)
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
			return c.Render(http.StatusOK, "404", nil)
		}

		recipe := Recipe{}
		if err := dsnap.DataTo(&recipe); err != nil {
			return c.Render(http.StatusOK, "error", nil)
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
	e.Static("/recipe/css", "./css")
	e.POST("/recipe/create/post", postPage(repo))
	e.GET("/recipe/create", createPage())
	e.GET("/recipe/list", listPage(repo))
	e.POST("/recipe/tweet/:recipeId", tweetPage(repo))
	e.GET("/recipe/:recipeId", recipePage(repo))

	e.Start(":" + os.Getenv("PORT"))
}
