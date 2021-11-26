package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

var (
	port = 8080
)

type NaverMeResponse struct {
	ResultCode string `json:"resultcode"`
	Message    string `json:"message"`
	Response   struct {
		Id       string `json:"id"`
		Nickname string `json:"nickname"`
		Gender   string `json:"gender"`
		Email    string `json:"email"`
		Name     string `json:"name"`
	} `json:"response"`
	Token *oauth2.Token
}

func main() {
	clientId := os.Getenv("NAVER_CLIENT_ID")
	clientSecret := os.Getenv("NAVER_CLIENT_SECRET")
	redirectUri := "http://localhost:8080/auth"

	oauth2Conf := &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://nid.naver.com/oauth2.0/authorize",
			TokenURL: "https://nid.naver.com/oauth2.0/token",
		},
		RedirectURL: redirectUri,
	}

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(ctx *gin.Context) {
		url := oauth2Conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
		ctx.HTML(http.StatusOK, "index.tmpl", gin.H{
			"url": url,
		})
	})

	r.GET("/auth", func(ctx *gin.Context) {
		code := ctx.Query("code")

		tok, err := oauth2Conf.Exchange(oauth2.NoContext, code)
		if err != nil {
			ctx.JSON(500, err)
			return
		}

		client := oauth2Conf.Client(oauth2.NoContext, tok)
		res, err := client.Get("https://openapi.naver.com/v1/nid/me")
		if err != nil {
			ctx.JSON(500, err)
			return
		}

		var data NaverMeResponse
		dec := json.NewDecoder(res.Body)
		dec.Decode(&data)

		data.Token = tok

		ctx.JSON(http.StatusOK, data)
	})

	r.Run(fmt.Sprintf(":%d", port))
}
