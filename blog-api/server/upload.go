package server

import (
	"awesomeProject1/utils"

	"github.com/gin-gonic/gin"
)

func UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{
			"msg": "未获取到文件",
		})
		return
	}

	// 简单校验类型（可扩展）
	if file.Size > 5<<20 {
		c.JSON(400, gin.H{
			"msg": "文件不能超过5MB",
		})
		return
	}

	url, err := utils.UploadToOSS(file)
	if err != nil {
		c.JSON(500, gin.H{
			"msg": "上传失败",
		})
		return
	}

	c.JSON(200, gin.H{
		"msg": "上传成功",
		"url": url,
	})
}
