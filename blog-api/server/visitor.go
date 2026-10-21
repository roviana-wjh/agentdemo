package server

import (
	"awesomeProject1/middleware"
	"awesomeProject1/model"
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func Login(c *gin.Context) {
	context := gin.H{
		"state": "fail",
		"msg":   "登陆失败",
	}

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	c.BindJSON(&body)
	username := body.Username
	password := body.Password
	m := md5.New()
	if username != "" && password != "" {
		context["state"] = "success"
		lastLogin := time.Now()
		context["last_login"] = lastLogin.Format("2006-01-02 15:04:05")
		m.Write([]byte(password))
		password = hex.EncodeToString(m.Sum(nil))
		var userId uint
		var user model.User
		model.DB.Where("name = ?", username).First(&user)
		if user.ID > 0 {
			if user.Password == password {
				userId = user.ID
				user.Lastlogin = time.Now()
				model.DB.Save(&user)
				context["msg"] = "登录成功"
			} else {
				context["msg"] = "请输入正确密码"
				context["state"] = "fail"
			}
		} else {
			context["msg"] = "注册成功"
			ne := model.User{
				Name:      username,
				Password:  password,
				Lastlogin: time.Now(),
			}
			model.DB.Create(&ne)
			if ne.ID > 0 {
				userId = ne.ID
			} else {
				context["state"] = "fail"
				context["msg"] = "注册失败"
			}
		}

		if userId > 0 {
			token, err := middleware.GenerateToken(username, int(userId))
			if err != nil {
				return
			}
			context["token"] = token
		}
	}
	c.JSON(http.StatusOK, context)
}

func Logout(c *gin.Context) {
	context := gin.H{
		"state": "fail",
		"msg":   "退出失败",
	}
	userId, _ := c.Get("userId")
	if userId != 0 {
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader != "" {
			var jwt model.JWts
			model.DB.Where("token = ?", authHeader).First(&jwt)
			model.DB.Delete(&jwt).Unscoped()
			context["state"] = "success"
			context["msg"] = "退出成功"
		}
	}
	c.JSON(http.StatusOK, context)
}

func Home(c *gin.Context) {
	//context := gin.H{
	//	"state": "fail",
	//	"msg":   "获取成功",
	//}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize
	var total int64
	var articles []model.Article
	model.DB.Model(&model.Article{}).Count(&total)
	model.DB.Model(&model.Article{}).Offset(offset).Limit(pageSize).Where("status = ?", "published").Find(&articles)
	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"list":  articles,
	})
}
