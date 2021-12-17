package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"

	"github.com/ovh/cds/tools/smtpmock"
)

type ConfigAPI struct {
	Port      int
	PortSMTP  int
	WithAuth  bool
	JwtSecret string
}

var configAPI ConfigAPI

func StartAPI(ctx context.Context, c ConfigAPI) error {
	configAPI = c

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	if configAPI.WithAuth {
		fmt.Println("Starting with auth enabled")
		if err := InitJWT([]byte(configAPI.JwtSecret)); err != nil {
			return err
		}
	}

	e.GET("/", httpRootHandler)
	e.POST("/signin", httpSigninHandler)

	mess := e.Group("/messages", middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		Skipper: func(c echo.Context) bool {
			return !configAPI.WithAuth
		},
		KeyLookup:  "header:" + echo.HeaderAuthorization,
		AuthScheme: "Bearer",
		Validator: func(key string, c echo.Context) (bool, error) {
			if _, err := CheckSessionToken(key); err != nil {
				return false, nil
			}
			return true, nil
		},
	}))

	{ // sub routes for /messages
		mess.GET("", func(c echo.Context) error {
			fmt.Println(c.Request().Header.Get("Authorization"))
			return c.JSON(http.StatusOK, StoreGetMessages())
		})
		mess.GET("/:recipent", func(c echo.Context) error {
			return c.JSON(http.StatusOK, StoreGetRecipientMessages(c.Param("recipent")))
		})
		mess.GET("/:recipent/latest", func(c echo.Context) error {
			messages := StoreGetRecipientMessages(c.Param("recipent"))
			if len(messages) == 0 {
				return c.JSON(http.StatusNotFound, "not found")
			}
			return c.JSON(http.StatusOK, messages[0])
		})
	}

	return e.Start(fmt.Sprintf(":%d", configAPI.Port))
}

func httpRootHandler(c echo.Context) error {
	var s = fmt.Sprintf("SMTP server listenning on %d\n", configAPI.PortSMTP)
	s += fmt.Sprintf("%d mails received to %d recipents\n", StoreCountMessages(), StoreCountRecipients())
	return c.String(http.StatusOK, s)
}

func httpSigninHandler(c echo.Context) error {
	if !configAPI.WithAuth {
		return c.JSON(http.StatusOK, smtpmock.SigninResponse{})
	}

	var data smtpmock.SigninRequest
	if err := c.Bind(&data); err != nil {
		return errors.WithStack(err)
	}

	subjectID, err := CheckSigninToken(data.SigninToken)
	if err != nil {
		return errors.WithStack(err)
	}

	sessionID, sessionToken, err := NewSessionToken(subjectID)
	if err != nil {
		return errors.WithStack(err)
	}

	StoreAddSession(sessionID)

	return c.JSON(http.StatusOK, smtpmock.SigninResponse{
		SessionToken: sessionToken,
	})
}
