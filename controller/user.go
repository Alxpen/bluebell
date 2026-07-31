package controller

import (
	"bluebell/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SignUpHandler(c *gin.Context) {
	//获取参数和参数校验
	var p models.ParamSignUp
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"msg" : "请求参数有误",
		})
		return
	}
	//业务处理

	//返回响应
}
