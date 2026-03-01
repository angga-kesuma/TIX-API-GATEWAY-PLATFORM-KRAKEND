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

	e.POST("/order", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{
			"header":  c.Request().Header,
			"code":    "SUCCESS",
			"message": "Success",
			"data": map[string]any{
				"hotelId":      "test-hotel-id",
				"checkInDate":  "2020-01-01",
				"checkOutDate": "2020-01-02",
			},
			"serverTime": time.Now().Unix(),
		})
	})

	e.POST("/partner/test", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{
			"header":  c.Request().Header,
			"code":    "SUCCESS",
			"message": "Success",
			"data": map[string]any{
				"partnerID":   "123",
				"createDate":  "2020-01-01",
				"updatedDate": "2020-01-02",
			},
			"serverTime": time.Now().Unix(),
		})
	})

	e.Logger.Fatal(e.Start(":1324"))
}
