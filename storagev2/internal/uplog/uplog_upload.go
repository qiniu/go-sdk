package uplog

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	clientv1 "github.com/qiniu/go-sdk/v7/client"
)

const (
	X_LOG_CLIENT_ID = "X-Log-Client-Id"
	originUplogUrl  = "https://uplog.qbox.me"
)

var (
	uplogUrl                  = originUplogUrl
	uplogUrlMutex             sync.Mutex
	lastUpToken, xLogClientId string
)

func uploadUplogLog(archivedPaths []string) error {
	if getUpToken == nil {
		return nil
	} else if newUpToken, err := getUpToken(); err == nil {
		lastUpToken = newUpToken
	}

	headers := make(http.Header, 2)
	headers.Set("Authorization", "UpToken "+lastUpToken)
	if xLogClientId != "" {
		headers.Set(X_LOG_CLIENT_ID, xLogClientId)
	}
	mfr, err := newMultipleFileReader(archivedPaths)
	if err != nil {
		return err
	}
	resp, err := clientv1.DefaultClient.DoRequestWithBodyGetter(
		context.Background(),
		http.MethodPost,
		GetUplogUrl()+"/log/4?compressed=gzip",
		headers,
		mfr,
		func() (io.ReadCloser, error) {
			return newMultipleFileReader(archivedPaths)
		},
		-1,
	)
	if err != nil {
		return err
	}
	if curXLogClientId := resp.Header.Get(X_LOG_CLIENT_ID); curXLogClientId != "" {
		xLogClientId = curXLogClientId
	}
	return clientv1.CallRet(context.Background(), nil, resp)
}

func uploadAllClosedFileBuffers() {
	fileLock := getUplogFileDirectoryLock()

	if err := fileLock.Lock(); err != nil {
		return
	}
	defer fileLock.Close()

	archivedPaths, err := getArchivedUplogFileBufferPaths(filepath.Dir(fileLock.Path()))
	if err != nil || len(archivedPaths) == 0 {
		return
	}

	if err = uploadUplogLog(archivedPaths); err == nil {
		for _, archarchivedPath := range archivedPaths {
			os.Remove(archarchivedPath)
		}
	}
}

func GetUplogUrl() string {
	uplogUrlMutex.Lock()
	defer uplogUrlMutex.Unlock()

	return uplogUrl
}

func SetUplogUrl(url string) {
	uplogUrlMutex.Lock()
	defer uplogUrlMutex.Unlock()

	if url == "" {
		url = originUplogUrl
	}

	uplogUrl = url
}
