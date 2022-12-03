package main

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/tayalone/go-circuit-breaker/green/puched"
)

func main() {
	e := echo.New()

	ph := puched.New()

	e.GET("/ping", func(c echo.Context) error {
		// return c.String(http.StatusOK, "Hello, World!")
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "OK",
		})
	})

	e.GET("/puched", func(c echo.Context) error {
		counter, state := ph.Hit()

		payload := map[string]interface{}{
			"message": "OK",
			"counter": counter,
		}

		switch state {
		case puched.StateMidful:
			{
				return c.JSON(http.StatusOK, payload)
			}
		case puched.StateAnnoy:
			{
				time.Sleep(420 * time.Millisecond)
				return c.JSON(http.StatusOK, payload)
			}
		default:
			{
				return c.JSON(http.StatusGone, map[string]interface{}{})
			}
		}
	})

	e.Logger.Fatal(e.Start(":8081"))
}
