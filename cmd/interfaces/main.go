package main

import (
	"github.com/gin-gonic/gin"
	"newsCenter/cmd/config"
	"newsCenter/cmd/interfaces/article"
	"newsCenter/cmd/interfaces/ranking"
	"newsCenter/cmd/interfaces/search"
	"newsCenter/cmd/interfaces/user"
)

func initAll(r *gin.Engine) {
	initRouter(r)
	user.InitRpcUserClient()
	article.InitRpcArticleClient()
	ranking.InitRpcRankingClient()
	search.InitRpcSearchClient()
	config.InitConfig()
}

func main() {
	r := gin.Default()
	//r.Use(logs.GinLogger())
	initAll(r)
	err := r.Run(config.ApiConfig.ServerConfig.Addr)
	if err != nil {
		return
	}
}
