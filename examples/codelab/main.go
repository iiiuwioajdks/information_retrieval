// 一个微博搜索的例子。
package main

import (
	"encoding/gob"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"io/ioutil"
	"path/filepath"
	"time"
	"strings"

	"github.com/huichen/wukong/engine"
	"github.com/huichen/wukong/types"
)

const (
	SecondsInADay     = 86400
	MaxTokenProximity = 2
)

var (
	searcher      = engine.Engine{}
	wbs           = map[uint64]Weibo{}
	dictFile      = flag.String("dict_file", "../../data/dictionary.txt", "词典文件")
	stopTokenFile = flag.String("stop_token_file", "../../data/stop_tokens.txt", "停用词文件")
	staticFolder  = flag.String("static_folder", "static", "静态文件目录")
)

type Weibo struct {
	Id           uint64 `json:"id"`
	Timestamp    uint64 `json:"timestamp"`
	UserName     string `json:"user_name"`
	RepostsCount uint64 `json:"reposts_count"`
	Text         string `json:"text"`
}

type Data struct {
	User User `json:"user"`
	WeiboData []WeiboData `json:"weibo"`
}
type User struct {
	ID string `json:"id"`
	ScreenName string `json:"screen_name"`
	Gender string `json:"gender"`
	Birthday string `json:"birthday"`
	Location string `json:"location"`
	Education string `json:"education"`
	Company string `json:"company"`
	RegistrationTime string `json:"registration_time"`
	Sunshine string `json:"sunshine"`
	StatusesCount int `json:"statuses_count"`
	FollowersCount int `json:"followers_count"`
	FollowCount int `json:"follow_count"`
	Description string `json:"description"`
	ProfileURL string `json:"profile_url"`
	ProfileImageURL string `json:"profile_image_url"`
	AvatarHd string `json:"avatar_hd"`
	Urank int `json:"urank"`
	Mbrank int `json:"mbrank"`
	Verified bool `json:"verified"`
	VerifiedType int `json:"verified_type"`
	VerifiedReason string `json:"verified_reason"`
}
type WeiboData struct {
	UserID uint64 `json:"user_id"`
	ScreenName string `json:"screen_name"`
	ID uint64 `json:"id"`
	Bid string `json:"bid"`
	Text string `json:"text"`
	ArticleURL string `json:"article_url"`
	Pics string `json:"pics"`
	VideoURL string `json:"video_url"`
	Location string `json:"location"`
	CreatedAt string `json:"created_at"`
	Source string `json:"source"`
	AttitudesCount int `json:"attitudes_count"`
	CommentsCount int `json:"comments_count"`
	RepostsCount int `json:"reposts_count"`
	Topics string `json:"topics"`
	AtUsers string `json:"at_users"`
	FullCreatedAt string `json:"full_created_at"`
}

/*******************************************************************************
    索引
*******************************************************************************/
func indexWeibo() {
	// 读入微博数据
	layout := "2006-01-02 15:04:05"
	files, err := ioutil.ReadDir("../../testdata/weibo-crawler/weibo")
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue // 忽略文件夹
		}
		filePath := filepath.Join("../../testdata/weibo-crawler/weibo", file.Name())
		log.Println(file.Name())
		fileContent, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Printf("无法读取文件 %s: %v\n", filePath, err)
			continue
		}

		var data Data
		err = json.Unmarshal(fileContent, &data)
		if err != nil {
			log.Printf("无法解析文件 %s: %v\n", filePath, err)
			continue
		}
		for _, w := range data.WeiboData {
			wb := Weibo{}
			wb.Id = w.ID
			t, err := time.Parse(layout, w.FullCreatedAt)
			if err != nil {
				log.Println("无法解析日期:", err)
				continue
			}
			wb.Timestamp = uint64(t.Unix())
			wb.UserName = w.ScreenName
			wb.RepostsCount = uint64(w.RepostsCount)
			wb.Text = w.Text
			wbs[wb.Id] = wb
		}
	}

	log.Print("添加索引")
	for docId, weibo := range wbs {
		searcher.IndexDocument(docId, types.DocumentIndexData{
			Content: weibo.Text,
			Fields: WeiboScoringFields{
				Timestamp:    weibo.Timestamp,
				RepostsCount: weibo.RepostsCount,
			},
		}, false)
	}

	searcher.FlushIndex()
	log.Printf("索引了%d条微博\n", len(wbs))
}

/*******************************************************************************
    评分
*******************************************************************************/
type WeiboScoringFields struct {
	Timestamp    uint64
	RepostsCount uint64
}

type WeiboScoringCriteria struct {
}

func (criteria WeiboScoringCriteria) Score(
	doc types.IndexedDocument, fields interface{}) []float32 {
	if reflect.TypeOf(fields) != reflect.TypeOf(WeiboScoringFields{}) {
		return []float32{}
	}
	wsf := fields.(WeiboScoringFields)
	output := make([]float32, 3)
	if doc.TokenProximity > MaxTokenProximity {
		output[0] = 1.0 / float32(doc.TokenProximity)
	} else {
		output[0] = 1.0
	}
	output[1] = float32(wsf.Timestamp / (SecondsInADay * 3))
	output[2] = float32(doc.BM25 * (1 + float32(wsf.RepostsCount)/10000))
	return output
}

/*******************************************************************************
    JSON-RPC
*******************************************************************************/
type JsonResponse struct {
	Docs []*Weibo `json:"docs"`
}

func JsonRpcServer(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query().Get("query")
	output := searcher.Search(types.SearchRequest{
		Text: query,
		RankOptions: &types.RankOptions{
			ScoringCriteria: &WeiboScoringCriteria{},
			OutputOffset:    0,
			MaxOutputs:      100,
		},
	})

	// 整理为输出格式
	docs := []*Weibo{}
	for _, doc := range output.Docs {
		wb := wbs[doc.DocId]
		for _, t := range output.Tokens {
			wb.Text = strings.Replace(wb.Text, t, "<font color=red>"+t+"</font>", -1)
		}
		docs = append(docs, &wb)
	}
	response, _ := json.Marshal(&JsonResponse{Docs: docs})

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, string(response))
}

/*******************************************************************************
	主函数
*******************************************************************************/
func main() {
	// 解析命令行参数
	flag.Parse()

	// 初始化
	gob.Register(WeiboScoringFields{})
	log.Print("引擎开始初始化")
	searcher.Init(types.EngineInitOptions{
		SegmenterDictionaries: *dictFile,
		StopTokenFile:         *stopTokenFile,
		IndexerInitOptions: &types.IndexerInitOptions{
			IndexType: types.LocationsIndex,
		},
		// 如果你希望使用持久存储，启用下面的选项
		// 默认使用boltdb持久化，如果你希望修改数据库类型
		// 请修改 WUKONG_STORAGE_ENGINE 环境变量
		// UsePersistentStorage: true,
		// PersistentStorageFolder: "weibo_search",
	})
	log.Print("引擎初始化完毕")
	wbs = make(map[uint64]Weibo)

	// 索引
	log.Print("建索引开始")
	go indexWeibo()
	log.Print("建索引完毕")

	// 捕获ctrl-c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			log.Print("捕获Ctrl-c，退出服务器")
			searcher.Close()
			os.Exit(0)
		}
	}()

	http.HandleFunc("/json", JsonRpcServer)
	http.Handle("/", http.FileServer(http.Dir(*staticFolder)))
	log.Print("服务器启动")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
