package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/tayalone/go-circuit-breaker/green/puched"
	ess "github.com/tayalone/go-ess-package/otel"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.uber.org/zap"
)

func main() {
	// // ------ Setup Otel with Jeager  -------------------------------------------
	tp, err := ess.JaegertracerProvider(os.Getenv("JEAGER_ENDPOINT"), os.Getenv("SERVICE_NAME"), os.Getenv("ENVIROMENT"))
	if err != nil {
		log.Fatal(err)
	}

	otelCtx := context.Background()
	defer func(ctx context.Context) {
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}(otelCtx)
	// // ---------------------------------------------------------
	// // --------- set up zapp --------------------------------
	logger, _ := zap.NewProduction(zap.AddStacktrace(zap.ErrorLevel))
	defer logger.Sync() // flushes buffer, if any

	otelLogger := otelzap.New(logger, otelzap.WithMinLevel(zap.DebugLevel), otelzap.WithTraceIDField(true))
	undo := otelzap.ReplaceGlobals(otelLogger)
	defer undo()
	otelSugar := otelLogger.Sugar()

	// // ------------------------------------------------------

	e := echo.New()

	e.Use(otelecho.Middleware(os.Getenv("SERVICE_NAME")))

	ph := puched.New()

	e.GET("/ping", func(c echo.Context) error {
		// return c.String(http.StatusOK, "Hello, World!")
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "OK",
		})
	})

	e.GET("/error", func(c echo.Context) error {
		return c.String(http.StatusInternalServerError, "Internal Server Error")
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
				otelSugar.Ctx(c.Request().Context()).Infow("I'm still OK")

				return c.JSON(http.StatusOK, payload)
			}
		case puched.StateAnnoy:
			{
				otelSugar.Ctx(c.Request().Context()).Warnw("I'm Anoying")

				time.Sleep(420 * time.Millisecond)
				return c.JSON(http.StatusOK, payload)
			}
		default:
			{
				otelSugar.Ctx(c.Request().Context()).Errorw("I'm Raging")

				return c.JSON(http.StatusGone, map[string]interface{}{})
			}
		}
	})

	e.Logger.Fatal(e.Start(":" + os.Getenv("PORT")))
}
