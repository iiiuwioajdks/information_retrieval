信息检索大作业：
实现一个微博搜索引擎

爬虫代码在 testdata/weibo-crawler 中，爬取后的用户数据存在 testdata/weibo-crawler/weibo_user（这个版本没用到这个），微博数据存在 testdata/weibo-crawler/weibo
微博全文搜索代码在 examples/codelab 中

启动步骤
```
git clone git@github.com:iiiuwioajdks/information_retrieval.git

cd information_retrieval

go mod tidy

cd examples/codelab

go mod tidy

go run main.go
```

然后访问：localhost:8088 即可进行搜索

如何进行爬虫：
```
cd testdata/weibo-crawler
```
只需要更改 config.json 里面的：
```
"user_id_list": [
        
]
```
将想爬的 user_id 加入这里，然后查看 weibo 文件夹下是否有多处这个对应 id 的 json 文件，有的话返回 examples/codelab，重新执行 `go run main.go` 即可
