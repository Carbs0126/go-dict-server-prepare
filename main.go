package main

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"github.com/gen2brain/go-unarr"
	_ "github.com/mattn/go-sqlite3"
	"math"
	"os"
	"path/filepath"
	"strings"
)

const SrcFilePath = "stardict.7z"
const SrcDirPath = "stardict"
const TableName = "dict"
const DatabaseName = "dict.db"

var sqliteDB *sql.DB

func main() {
	targetCSVFilePath := filepath.Join("stardict", "stardict.csv")
	if !checkFileExist(targetCSVFilePath) {
		extract7Z(SrcFilePath, SrcDirPath)
	}
	if checkFileExist(targetCSVFilePath) {
		sqliteDB = initSQLite3DB()
		if sqliteDB != nil {
			defer sqliteDB.Close()
		}
	}
	readCsvAndInsertIntoDB(targetCSVFilePath, 100000)
	//clearTable()
}

func extract7Z(srcFilePath string, targetDirPath string) {
	a, err := unarr.NewArchive(srcFilePath)
	if err != nil {
		panic(err)
	}
	defer a.Close()
	_, err = a.Extract(targetDirPath)
	if err != nil {
		fmt.Println("解压文件时发生错误:", err)
		panic(err)
	}
}

func initSQLite3DB() *sql.DB {
	db, err := sql.Open("sqlite3", DatabaseName)
	checkError(err, "err1 :")
	err = db.Ping()
	checkError(err, "err2 :")

	// 查询数据库中是否存在指定的表
	strQuery := fmt.Sprintf("SELECT name FROM sqlite_master WHERE type='table' AND name='%s'", TableName)
	var result string
	err = db.QueryRow(strQuery).Scan(&result)
	if err != nil {
		if err == sql.ErrNoRows {
			strCreate := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
									id INTEGER PRIMARY KEY AUTOINCREMENT,
    								word VARCHAR(64) NULL,
    								translation VARCHAR(64) NULL)`, TableName)
			_, err = db.Exec(strCreate)
			checkError(err, "err3 :")
		} else {
			checkError(err, "err7 :")
		}
	}
	return db
}

func insertIntoDict(stmt *sql.Stmt, word string, definition string, row int) int64 {
	if sqliteDB == nil {
		panic(errors.New("err 8"))
	}
	res, err := stmt.Exec(word, definition)
	if err != nil {
		fmt.Println("insertIntoDict error2, word:", word, " row: ", row)
	}
	id, err := res.LastInsertId()
	if err != nil {
		fmt.Println("insertIntoDict error3, word:", word, " row: ", row)
	}
	return id
}

func checkFileExist(filePath string) bool {
	_, err := os.Stat(filePath)
	// 如果文件存在，err将为nil
	if err == nil {
		return true
	} else {
		return false
	}
}

func checkError(err error, msg string) {
	if err != nil {
		fmt.Println(msg, err)
		panic(err)
	}
}

func readCsvAndInsertIntoDB(csvFilePath string, maxWordCount int) error {
	// 打开文本文件以供读取
	file, err := os.Open(csvFilePath)
	if err != nil {
		fmt.Println("无法打开文件:", err)
		return err
	}
	defer file.Close()
	if maxWordCount <= 0 {
		maxWordCount = math.MaxInt
	}

	// 创建一个新的Scanner来从文件中读取数据
	scanner := bufio.NewScanner(file)
	// 略过第一条
	scanner.Scan()
	row := 1
	wordCount := 0
	currWordCountHundred := 0
	prevWordCountHundred := 0
	var firstLetter uint8 = 0
	var splitLine []string = nil
	var word string
	var definition string
	strInsert := fmt.Sprintf("INSERT INTO %s(word, translation) values(?,?)", TableName)
	stmt, err := sqliteDB.Prepare(strInsert)
	checkError(err, "readCsv: error 0")
	// 逐行读取文件内容
	for wordCount < maxWordCount && scanner.Scan() {
		row++
		line := scanner.Text()
		firstLetter = line[0]
		if (96 < firstLetter && firstLetter < 123) || (64 < firstLetter && firstLetter < 91) {
			wordCount++
			splitLine = strings.Split(line, ",")
			if len(splitLine) > 4 {
				word = splitLine[0]
				definition = getDefinition(splitLine)
				insertIntoDict(stmt, word, definition, row)
			}
			currWordCountHundred = wordCount / 100
			if currWordCountHundred != prevWordCountHundred {
				fmt.Printf("\rCount of Words Inserted Into Dict -----> %d", currWordCountHundred*100)
				prevWordCountHundred = currWordCountHundred
			}
		}
	}

	// 检查扫描是否出错
	if err = scanner.Err(); err != nil {
		fmt.Println("扫描文件时出错:", err)
	}
	return err
}

func getDefinition(splitStrings []string) string {
	index := 0
	quoteMode := false

	var result strings.Builder
	splitStringsLength := len(splitStrings)
	for i := 0; i < splitStringsLength; i++ {
		curStringLength := len(splitStrings[i])
		if quoteMode {
			// 第三个是definition
			if index == 3 {
				result.WriteString(",")
				result.WriteString(splitStrings[i])
			}

			// 以英文双引号"结尾
			if curStringLength > 0 && splitStrings[i][curStringLength-1] == 34 {
				quoteMode = false
				index++
			}
		} else {
			// 以英文双引号"开头
			if curStringLength > 0 && splitStrings[i][0] == 34 {
				quoteMode = true
				if index == 3 {
					result.WriteString(splitStrings[i])
				}
				// 以英文双引号"开头，并且以英文双引号"结尾
				if len(splitStrings[i]) > 1 && splitStrings[i][curStringLength-1] == 34 {
					quoteMode = false
				} else {
					continue
				}
			} else {
				if index == 3 {
					return splitStrings[i]
				}
			}
			index++
		}
		if !quoteMode && index == 4 {
			break
		}
	}
	str := result.String()
	if strings.HasPrefix(str, "\"") {
		if strings.HasSuffix(str, "\"") && len(str) > 2 {
			return str[1 : len(str)-1]
		} else {
			return str[1:]
		}
	}
	return str
}

func clearTable() {
	if sqliteDB == nil {
		panic(errors.New("clearTable error"))
	}
	strDeleteTable := fmt.Sprintf("DELETE FROM %s", TableName)
	_, err := sqliteDB.Exec(strDeleteTable)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("表已成功清空！")
}
