package main

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/chenjiandongx/ginprom"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sony/gobreaker"
	ess "github.com/tayalone/go-ess-package/otel"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
	r := gin.Default()

	r.Use(otelgin.Middleware(os.Getenv("SERVICE_NAME")))
	// // -------- prmetheus -------------------------------
	// p := ginprometheus.NewPrometheus("gin")
	// p.Use(r)

	r.Use(ginprom.PromMiddleware(nil))
	r.GET("/metrics", ginprom.PromHandler(promhttp.Handler()))
	// // -------------------------------------------------

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})

	r.GET("/puch-without-cb", func(ctx *gin.Context) {
		// resp, err := http.Get("http://green:8081/puched")
		resp, err := otelhttp.Get(ctx.Request.Context(), "http://green:8081/puched")
		if err != nil {

			otelSugar.Ctx(ctx.Request.Context()).Errorw(err.Error())

			ctx.JSON(http.StatusGone, gin.H{
				"message": "Green Services Gone",
			})
			return
		}

		if resp.StatusCode == 200 {
			otelSugar.Ctx(ctx.Request.Context()).Infow("Green Still Ok")

			ctx.JSON(http.StatusOK, gin.H{
				"message": "OK",
			})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "http://green:8081/error return 500",
		})
		return
	})

	// Define Services Breaker
	var st gobreaker.Settings
	st.Name = "HTTP GET Punch Green services"
	st.ReadyToTrip = func(counts gobreaker.Counts) bool {
		// failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
		return counts.TotalFailures > 80
	}
	st.Timeout = time.Duration(6 * time.Second)

	cb := gobreaker.NewCircuitBreaker(st)
	// ---------------------------------------------

	r.GET("/puch-with-cb", func(ctx *gin.Context) {
		_, err := cb.Execute(func() (interface{}, error) {
			start := time.Now()
			// resp, err := http.Get("http://green:8081/puched")
			resp, err := otelhttp.Get(ctx.Request.Context(), "http://green:8081/puched")

			elapsed := time.Now().Sub(start)

			if err != nil {
				return nil, err
			}

			// // Gress is Rage State
			if resp.StatusCode == http.StatusGone {
				return nil, errors.New("Green'state is Rage")
			}

			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}

			if elapsed > time.Duration(400*time.Millisecond) {
				return body, errors.New("Green state is Anoying")
			}

			return body, nil
		})
		if err != nil {
			switch err.Error() {
			case "Green state is Anoying":
				{
					otelSugar.Ctx(ctx.Request.Context()).Warnw(err.Error())
					break
				}
			case "too many requests":
				{
					//  is returned when the CB state is half open and the requests count is over the cb maxRequests
					otelSugar.Ctx(ctx.Request.Context()).Errorw(err.Error())
					ctx.JSON(http.StatusTooManyRequests, gin.H{
						"message": "Don't Be Green Rage",
					})
					return
				}
			case "circuit breaker is open":
				{
					otelSugar.Ctx(ctx.Request.Context()).Errorw(err.Error())
					// is returned when the CB state is open
					ctx.JSON(http.StatusForbidden, gin.H{
						"message": "Green Going to Get Rage",
					})
					return
				}
			case "Green'state is Rage":
				{
					otelSugar.Ctx(ctx.Request.Context()).Errorw(err.Error())
					ctx.JSON(http.StatusTooManyRequests, gin.H{
						"message": "Green Still Rage",
					})
					return
				}
			default:
				{
					otelSugar.Ctx(ctx.Request.Context()).Errorw(err.Error())
					ctx.JSON(http.StatusBadRequest, gin.H{
						"message": "Green got some problem",
					})
					return
				}
			}
		}

		ctx.JSON(http.StatusOK, gin.H{
			"message": "OK",
		})
		return
	})

	r.Run(":" + os.Getenv("PORT")) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
