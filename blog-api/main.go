package main

import (
	"awesomeProject1/model"
	"awesomeProject1/router"
	"awesomeProject1/utils"
	"log"
	"net/http"
	"time"
)

// TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>
func init() {
	model.Setup()
	utils.InitES()
}

func main() {

	server := &http.Server{
		Addr:        ":8080",
		Handler:     router.InitRouter(),
		ReadTimeout: time.Duration(10) * time.Second,
		// SSE responses can legitimately stream for minutes. A global write
		// deadline would cut the connection and surface as a browser network error.
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}
	err := server.ListenAndServe()

	// 检查并记录错误。如果是正常关闭，err 通常是 http.ErrServerClosed
	if err != nil && err != http.ErrServerClosed {
		// 使用 log 包打印致命错误，这会确保输出被看到
		log.Fatalf("Gin server listen failed: %v", err)
	}

}
