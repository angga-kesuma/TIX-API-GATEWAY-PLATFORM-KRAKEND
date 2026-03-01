package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus: true,
		LogURI:    true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Status != http.StatusOK {
				fmt.Printf("REQUEST: uri: %v, status: %v\n", v.URI, v.Status)
			}
			return nil
		},
	}))

	e.POST("/auth", func(c echo.Context) error {
		time.Sleep(100 * time.Millisecond)
		req := struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}{}
		err := c.Bind(&req)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"msg": "Internal server error",
			})
		}
		return c.JSON(http.StatusOK, map[string]any{
			"code":    "SUCCESS",
			"message": "Success",
			"data": map[string]any{
				"username":    req.Username,
				"partnerId":   "35010227",
				"partnerName": "Agoda",
				"storeId":     "TIKETB2B",
				"channelId":   "DESKTOP",
				"login":       true,
			},
			"serverTime": fmt.Sprintf("%d", time.Now().Unix()),
		})
	})
	e.Logger.Fatal(e.Start(":1323"))
}
