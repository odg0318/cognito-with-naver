package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

const (
	NAVER_URL_AUTH      = "https://nid.naver.com/oauth2.0/authorize"
	NAVER_URL_TOKEN     = "https://nid.naver.com/oauth2.0/token"
	NAVER_URL_MY_INFO   = "https://openapi.naver.com/v1/nid/me"
	OAUTH2_REDIRECT_URL = "http://localhost:8080/auth"
)

var (
	port             = 8080
	userPoolId       = os.Getenv("COGNITO_UESR_POOL_ID")
	userPoolClientId = os.Getenv("COGNITO_USER_POOL_CLIENT_ID")
	cognitoPassword  = "AwsCognito100$"
	clientId         = os.Getenv("NAVER_CLIENT_ID")
	clientSecret     = os.Getenv("NAVER_CLIENT_SECRET")
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
}

func main() {
	oauth2Conf := &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  NAVER_URL_AUTH,
			TokenURL: NAVER_URL_TOKEN,
		},
		RedirectURL: OAUTH2_REDIRECT_URL,
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

		token, err := oauth2Conf.Exchange(oauth2.NoContext, code)
		if err != nil {
			ctx.JSON(500, err)
			return
		}

		ctx.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/info?access_token=%s", token.AccessToken))
	})

	r.GET("/info", func(ctx *gin.Context) {
		accessToken := ctx.Query("access_token")

		ctx.HTML(http.StatusOK, "my.tmpl", gin.H{
			"accessToken": accessToken,
		})
	})

	r.POST("/get-my-naver-info", func(ctx *gin.Context) {
		accessToken := ctx.PostForm("access_token")

		data, err := getMyNaverInfo(accessToken)
		if err != nil {
			ctx.JSON(500, err)
			return
		}

		ctx.JSON(200, data)
	})

	r.POST("/check-user-exist", func(ctx *gin.Context) {
		accessToken := ctx.PostForm("access_token")

		data, err := getMyNaverInfo(accessToken)
		if err != nil {
			ctx.JSON(500, err)
			return
		}

		exist := checkUserExist(data.Response.Email)

		ctx.JSON(200, gin.H{"exist": exist})
	})

	r.POST("/signup", func(ctx *gin.Context) {
		accessToken := ctx.PostForm("access_token")

		data, err := getMyNaverInfo(accessToken)
		if err != nil {
			ctx.JSON(500, err)
			return
		}

		err = signupCognito(data.Response.Email, data.Response.Name)
		if err != nil {
			ctx.JSON(500, err)
			return
		}

		err = confirmSignupCognito(data.Response.Email)
		if err != nil {
			ctx.JSON(500, err)
			return
		}

		ctx.JSON(200, gin.H{"status": "ok"})
	})

	r.POST("/signin", func(ctx *gin.Context) {
		accessToken := ctx.PostForm("access_token")

		data, err := getMyNaverInfo(accessToken)
		if err != nil {
			ctx.JSON(500, err)
			return
		}

		res, err := signinCognito(data.Response.Email)
		if err != nil {
			ctx.JSON(500, err)
			return
		}

		ctx.JSON(200, res)
	})

	r.POST("/leave", func(ctx *gin.Context) {
		accessToken := ctx.PostForm("access_token")

		res, err := deleteCognitoUser(accessToken)
		if err != nil {
			ctx.JSON(500, err)
			return
		}

		ctx.JSON(200, res)
	})

	r.POST("/get-my-cognito-info", func(ctx *gin.Context) {
		accessToken := ctx.PostForm("access_token")

		data, err := getMyCognitoInfo(accessToken)
		if err != nil {
			ctx.JSON(500, err)
			return
		}

		ctx.JSON(200, data)
	})

	r.Run(fmt.Sprintf(":%d", port))
}

func signupCognito(email string, name string) error {
	sess := session.Must(session.NewSession())
	cognito := cognitoidentityprovider.New(sess)

	input := cognitoidentityprovider.SignUpInput{
		ClientId: &userPoolClientId,
		Username: aws.String(email),
		Password: aws.String(cognitoPassword),
		UserAttributes: []*cognitoidentityprovider.AttributeType{
			{
				Name:  aws.String("email"),
				Value: aws.String(email),
			},
			{
				Name:  aws.String("name"),
				Value: aws.String(name),
			},
		},
	}
	_, err := cognito.SignUp(&input)
	if err != nil {
		return err
	}

	return nil
}

func confirmSignupCognito(email string) error {
	sess := session.Must(session.NewSession())
	cognito := cognitoidentityprovider.New(sess)

	input := cognitoidentityprovider.AdminConfirmSignUpInput{
		Username:   aws.String(email),
		UserPoolId: aws.String(userPoolId),
	}
	_, err := cognito.AdminConfirmSignUp(&input)
	if err != nil {
		return err
	}

	return nil
}

func signinCognito(email string) (*cognitoidentityprovider.AuthenticationResultType, error) {
	sess := session.Must(session.NewSession())
	cognito := cognitoidentityprovider.New(sess)

	input := cognitoidentityprovider.AdminInitiateAuthInput{
		AuthFlow:   aws.String(cognitoidentityprovider.AuthFlowTypeAdminUserPasswordAuth),
		ClientId:   aws.String(userPoolClientId),
		UserPoolId: aws.String(userPoolId),
		AuthParameters: map[string]*string{
			"USERNAME": aws.String(email),
			"PASSWORD": aws.String(cognitoPassword),
		},
	}

	res, err := cognito.AdminInitiateAuth(&input)
	if err != nil {
		return nil, err
	}

	return res.AuthenticationResult, nil
}

func checkUserExist(email string) bool {
	sess := session.Must(session.NewSession())
	cognito := cognitoidentityprovider.New(sess)

	input := cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: aws.String(userPoolId),
		Username:   aws.String(email),
	}
	_, err := cognito.AdminGetUser(&input)
	if err != nil {
		return false
	}

	return true
}

func getMyNaverInfo(accessToken string) (*NaverMeResponse, error) {
	req, err := http.NewRequest("GET", NAVER_URL_MY_INFO, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var data NaverMeResponse
	dec := json.NewDecoder(res.Body)
	dec.Decode(&data)

	return &data, nil
}

func getMyCognitoInfo(accessToken string) (*cognitoidentityprovider.GetUserOutput, error) {
	sess := session.Must(session.NewSession())
	cognito := cognitoidentityprovider.New(sess)

	input := cognitoidentityprovider.GetUserInput{
		AccessToken: aws.String(accessToken),
	}
	output, err := cognito.GetUser(&input)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func deleteCognitoUser(accessToken string) (*cognitoidentityprovider.DeleteUserOutput, error) {
	sess := session.Must(session.NewSession())
	cognito := cognitoidentityprovider.New(sess)

	input := cognitoidentityprovider.DeleteUserInput{
		AccessToken: aws.String(accessToken),
	}
	output, err := cognito.DeleteUser(&input)
	if err != nil {
		return nil, err
	}

	return output, nil
}
