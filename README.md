# go-dict-server-prepare

## 功能
1. 释放```csv```格式的单词库；
2. 将```csv```文件中的单词，导入到```sqlite3```数据库中；

## 使用
1. 进入 ```go-dict-server-prepare``` 文件夹
2. ```go build``` 然后会生成 ```go-dict-server-prepare``` 可执行文件
3. 执行此文件，会生成 ```dict.db```


## 备注 
```
csv结构：
word,phonetic,definition,translation,pos,collins,oxford,tag,bnc,frq,exchange,detail,audio
```