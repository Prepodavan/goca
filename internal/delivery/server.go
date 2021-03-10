package delivery

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"path/filepath"
)

type API interface {
	Apply(router gin.IRouter)
}

func NewServer(docsDirPath string, issuerAPI API) (engine *gin.Engine) {
	engine = gin.Default()
	issuerAPI.Apply(engine.Group("/issuing"))
	engine.GET("/doc", func(ctx *gin.Context) {
		ctx.Redirect(http.StatusFound, ctx.Request.URL.Path+"/openapi.html")
	})
	engine.GET("/doc/:file", func(ctx *gin.Context) {
		ctx.File(filepath.Join(docsDirPath, ctx.Param("file")))
	})
	return
}
