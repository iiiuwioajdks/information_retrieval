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