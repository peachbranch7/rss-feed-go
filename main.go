package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"rss-feed/pkg"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

var RSSFeedURLs = []string{
 "https://tech.uzabase.com/rss",
 "https://news.yahoo.co.jp/rss/topics/top-picks.xml",
 "https://news.yahoo.co.jp/rss/topics/domestic.xml",
 "https://news.yahoo.co.jp/rss/topics/business.xml",
 "https://news.yahoo.co.jp/rss/media/vingtcinqw/all.xml",
}

const (
 removeTarget  = "NewsPicks"
 maxConcurrent = 3
)

// 同時実行数を制御するため
var s = semaphore.NewWeighted(maxConcurrent)

// 生成されたGoroutineが全て終わるのを待つため
var wg = &sync.WaitGroup{}

func main() {
 ctx := context.TODO()
 for _, url := range RSSFeedURLs {
  wg.Add(1)
  go exec(ctx, url)
 }
 // Addされた回数、Doneされるまで待つ
 // 待つ理由は main() が終了すると、goroutineも中断されるため
 wg.Wait()

}

func exec(ctx context.Context, url string) {
 // セマフォから一つボールを取る
 // 実行中であることをセマフォに伝える
 err := s.Acquire(ctx, 1)
 if err != nil {
  fmt.Fprintln(os.Stderr,fmt.Errorf("failed to acquire semaphore: %s", err))
  os.Exit(1)
 }
 // 関数が終了する時に呼び出される
 defer func() {
  // セマフォにボールを返す
  // 返すと別のGoroutineがボールを取れるようになる（実行できるようになる）
  s.Release(1)
  wg.Done()
 }()

 byteXML, _ := getRSSFeed(url)
 fc := pkg.NewFeedChecker(string(byteXML))
 removed := fc.RemoveWords(removeTarget)
 fmt.Println(removed)
 fmt.Println("-------------------------------------------------------------------------------------")

 filename, _ := createFilename(url)
 err = saveToFile(filename, removed)
 if err != nil {
  fmt.Fprintln(os.Stderr, fmt.Errorf("exit because error happened: %s", err))
  os.Exit(1)
 }
}

func getRSSFeed(feedURL string) ([]byte, error) {
 // URLのパース
 parsedURL, err := url.Parse(feedURL)
 if err != nil {
  fmt.Fprintln(os.Stderr, fmt.Errorf("failed to parse url, error: %s", err))
  return nil, err
 }

 // リクエストの生成
 req, _ := http.NewRequest("GET", parsedURL.String(), nil)
 // クライアントの生成
 client := new(http.Client)
 // リクエスト送信
 resp, err := client.Do(req)
 if err != nil {
  fmt.Fprintln(os.Stderr, fmt.Errorf("failed to send request, err: %s", err))
  return nil, err
 }

 // レスポンスのボディをバイト型の配列に変換
 body, err := ioutil.ReadAll(resp.Body)
 if err != nil {
 fmt.Fprintln(os.Stderr, fmt.Errorf("failed to read body, err: %s", err))
  return nil, err
 }

 return body, nil
}

func saveToFile(filename, contents string) error {
 f, err := os.Create(filename)
 if err != nil && err != os.ErrExist {
  fmt.Fprintln(os.Stderr, fmt.Errorf("failed to create file, err = %s", err.Error()))
  return err
 }
 _, err = f.Write([]byte(contents))
 if err != nil {
  fmt.Fprintln(os.Stderr, fmt.Errorf("failed to write to file, err = %s", err.Error()))
  return err
 }
 return nil
}

// filename
func createFilename(feedURL string) (string, error) {
 current := time.Now().Unix()
 splitURL := strings.Split(feedURL, "/")
 if len(splitURL) < 2 {
  errStr := "failed to create filename"
  fmt.Fprintln(os.Stderr, fmt.Errorf(errStr))
  return "", errors.New(errStr)
 }
 return strings.Join(splitURL[2:], "-") + "_" + strconv.Itoa(int(current)) + ".txt", nil
}