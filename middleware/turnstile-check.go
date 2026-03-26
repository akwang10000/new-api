package middleware

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

type turnstileCheckResponse struct {
	Success bool `json:"success"`
}

func VerifyTurnstileToken(response string, remoteIP string) error {
	if response == "" {
		return errors.New("Turnstile token is required")
	}

	rawRes, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", url.Values{
		"secret":   {common.TurnstileSecretKey},
		"response": {response},
		"remoteip": {remoteIP},
	})
	if err != nil {
		return err
	}
	defer rawRes.Body.Close()

	var res turnstileCheckResponse
	err = common.DecodeJson(rawRes.Body, &res)
	if err != nil {
		return err
	}
	if !res.Success {
		return errors.New("Turnstile verification failed, please refresh and try again")
	}
	return nil
}

func TurnstileCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !common.TurnstileCheckEnabled {
			c.Next()
			return
		}

		err := VerifyTurnstileToken(c.Query("turnstile"), c.ClientIP())
		if err != nil {
			common.SysLog(err.Error())
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
