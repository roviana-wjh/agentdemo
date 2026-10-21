package utils

import (
	"fmt"
	"mime/multipart"
	"path"
	"time"

	"awesomeProject1/setting"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

func UploadToOSS(file *multipart.FileHeader) (string, error) {
	client, err := oss.New(
		setting.OSSEndpoint,
		setting.OSSAccessKeyID,
		setting.OSSAccessKeySecret,
	)
	if err != nil {
		return "", err
	}

	bucket, err := client.Bucket(setting.OSSBucketName)
	if err != nil {
		return "", err
	}

	// 文件名：article/时间戳.xxx
	ext := path.Ext(file.Filename)
	objectName := fmt.Sprintf(
		"article/%d%s",
		time.Now().UnixNano(),
		ext,
	)

	f, err := file.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	err = bucket.PutObject(objectName, f)
	if err != nil {
		return "", err
	}

	return setting.OSSBaseURL + objectName, nil
}
