package zfile

// https://github.com/src-d/go-billy
// https://blog.csdn.net/robertkun/article/details/78744464
// https://www.cnblogs.com/zheng-chuang/p/6193090.html
import (
	"bufio"
	"fmt"
	"github.com/kuaileniu/zstring"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// https://github.com/yudeguang/file/blob/master/file.go
//检察文件是否允许读
func AllowRead(path string) bool {
	_, err := os.OpenFile(path, os.O_RDONLY, 0666)
	return err == nil
}

//检察文件是否允许写
func AllowWrite(path string) bool {
	_, err := os.OpenFile(path, os.O_WRONLY, 0666)
	return err == nil
}

//打开指定文件，并从指定位置写入数据
func WriteAt(path string, b []byte, off int64) error {
	file, err := os.OpenFile(path, os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteAt(b, off)
	return err
}

//打开指定文件,并在文件末尾写入数据
func WriteAppend(path string, b []byte) error {
	file, err := os.OpenFile(path, os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(b)
	return err
}

//覆盖已有内容重新写入
// 如果已经存在则打开文件，如果之前不存在则创建文件，然后覆盖已有内容重新写入
func ReWriteFile(relitivePathAndFileName string, b []byte) error {
	file, err := os.Open(relitivePathAndFileName)
	defer file.Close()
	file, err = os.Create(relitivePathAndFileName)
	if err != nil {
		err = os.MkdirAll(filepath.Dir(relitivePathAndFileName), 0777)
		if err != nil {
			return err
		}
		file, err = os.Create(relitivePathAndFileName)
		if err != nil {
			return err
		}
	}
	_, err = file.Write(b)
	return err
}

//复制文件，目标文件所在目录不存在，则创建目录后再复制
//Copy(`d:\test\hello.txt`,`c:\test\hello.txt`)
func Copy(dstFileName, srcFileName string) (w int64, err error) {
	//打开源文件
	srcFile, err := os.Open(srcFileName)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()
	// 创建新的文件作为目标文件
	dstFile, err := os.Create(dstFileName)
	if err != nil {
		//如果出错，很可能是目标目录不存在，需要先创建目标目录
		err = os.MkdirAll(filepath.Dir(dstFileName), 0666)
		if err != nil {
			return 0, err
		}
		//再次尝试创建
		dstFile, err = os.Create(dstFileName)
		if err != nil {
			return 0, err
		}
	}
	defer dstFile.Close()
	//通过bufio实现对大文件复制的自动支持
	dst := bufio.NewWriter(dstFile)
	defer dst.Flush()
	src := bufio.NewReader(srcFile)
	w, err = io.Copy(dst, src)
	if err != nil {
		return 0, err
	}
	return w, err
}

func CreateFolder(dir string) (err error) {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	return nil
}

// 根据相对路径获取绝对路径
func AbsPath(reletivePath string) (absPath string,err error) {
	absPath,err=filepath.Abs(reletivePath)
	return
}

// 按源路径的层级不变copy
// src:源，文件夹绝对路径
// targetAbsDir：目标，文件夹的绝对路径,当为空时则复制到执行文件的目录下
// File.Name是路径信息，FileInfo.Name是文件名

func CopyFolder(srcAbsDir string, targetAbsDir string, copySubfolder ...bool) {
	files, err := ioutil.ReadDir(srcAbsDir)
	if err != nil {
		zap.L().Error("读取源文件夹异常", zap.String("srcAbsDir", srcAbsDir), zap.Error(err))
		return
	}
	if err := os.MkdirAll(targetAbsDir, os.ModePerm); err != nil {
		zap.L().Error("创建目标文件夹异常", zap.String("srcAbsDir", srcAbsDir), zap.Error(err))
		return
	}

	for _, fileInfo := range files {
		if !fileInfo.IsDir() {
			fileName := fileInfo.Name()
			data, err := ioutil.ReadFile(srcAbsDir + string(os.PathSeparator) + fileName)
			if err != nil {
				zap.L().Error("读取文件时异常，%v,%v", zap.String("文件全路径", srcAbsDir+string(os.PathSeparator)+fileName), zap.Error(err))
			}
			if err := ioutil.WriteFile(targetAbsDir+string(os.PathSeparator)+fileName, data, fileInfo.Mode()); err != nil {
				zap.L().Error("写出文件时异常", zap.Error(err))
			}
		}
		if fileInfo.IsDir() {
			subfolder := fileInfo.Name()
			CopyFolder(srcAbsDir+string(os.PathSeparator)+subfolder, targetAbsDir+string(os.PathSeparator)+subfolder, copySubfolder...)
		}
	}
}

// 判断给定文件名是否是一个目录
// 如果文件名存在并且为目录则返回 true。如果 filename 是一个相对路径，则按照当前工作目录检查其相对路径。
func IsDir(filename string) bool {
	return isFileOrDir(filename, true)
}

func IsFile(filename string) bool {
	return isFileOrDir(filename, false)
}

// 判断是文件还是目录，根据decideDir为true表示判断是否为目录；否则判断是否为文件
func isFileOrDir(filename string, decideDir bool) bool {
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return false
	}
	isDir := fileInfo.IsDir()
	if decideDir {
		return isDir
	}
	return !isDir
}

func CheckFileIsExist(filepath string) bool {
	exist := true
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

//获得文件的修改时间
func FileModTime(path string) (int64, error) {
	f, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return f.ModTime().Unix(), nil
}

//返回文件的大小
func FileSize(path string) (int64, error) {
	f, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return f.Size(), nil
}

//遍历目录及下级目录，查找符合后缀文件,如果suffix为空，则查找所有文件
func GetFileListBySuffix(dirPath, suffix string) (files []string, err error) {
	if !IsDir(dirPath) {
		return nil, fmt.Errorf("given path does not exist: %s", dirPath)
	}
	files = make([]string, 0, 30)
	suffix = strings.ToUpper(suffix)                                                      //忽略后缀匹配的大小写
	err = filepath.Walk(dirPath, func(filename string, fi os.FileInfo, err error) error { //遍历目录
		if err != nil { //忽略错误
			return err
		}
		if fi.IsDir() { // 忽略目录
			return nil
		}
		if strings.HasSuffix(strings.ToUpper(fi.Name()), suffix) {
			files = append(files, filename)
		}
		return nil
	})
	return files, err
}

//遍历指定目录下的所有文件，查找符合后缀文件,不进入下一级目录搜索
func GetFileListJustCurrentDirBySuffix(dirPath string, suffix string) (files []string, err error) {
	if !IsDir(dirPath) {
		return nil, fmt.Errorf("given path does not exist: %s", dirPath)
	}
	files = make([]string, 0, 10)
	dir, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	PathSep := string(os.PathSeparator)
	suffix = strings.ToUpper(suffix) //忽略后缀匹配的大小写
	for _, fi := range dir {
		if fi.IsDir() { // 忽略目录
			continue
		}
		if strings.HasSuffix(strings.ToUpper(fi.Name()), suffix) {
			files = append(files, dirPath+PathSep+fi.Name())
		}
	}
	return files, nil
}

//把文件大小转换成人更加容易看懂的文本
// HumaneFileSize calculates the file size and generate user-friendly string.
func HumaneFileSize(s uint64) string {
	logn := func(n, b float64) float64 {
		return math.Log(n) / math.Log(b)
	}
	humanateBytes := func(s uint64, base float64, sizes []string) string {
		if s < 10 {
			return fmt.Sprintf("%dB", s)
		}
		e := math.Floor(logn(float64(s), base))
		suffix := sizes[int(e)]
		val := float64(s) / math.Pow(base, math.Floor(e))
		f := "%.0f"
		if val < 10 {
			f = "%.1f"
		}
		return fmt.Sprintf(f+"%s", val, suffix)
	}
	sizes := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	return humanateBytes(s, 1024, sizes)
}

// 根据相对路径找出目标文件或路径
func FromRelativePath(fileOrPath string, relativePath string) (fp string, err error) {
	fileOrPath = filepath.ToSlash(fileOrPath)
	// fmt.Printf("fileOrPath:%v\n",fileOrPath)
	// fileOrPath = filepath.Clean(fileOrPath)
	relativePath = filepath.ToSlash(relativePath)
	// relativePath = filepath.Clean(relativePath)
	// fmt.Printf("内部:%v\n", fileOrPath)
	fileInfo, err := os.Stat(fileOrPath)
	if err != nil {
		return
	}
	path := ""
	if fileInfo.IsDir() {
		// fmt.Println("是路径")
		path = fileOrPath
		if strings.HasSuffix(path, "/") {
		} else {
			path += "/"
		}
	} else {
		// fmt.Println("非路径")
		// path = zstrings.BeforeNSep(fileOrPath,"/",0)
		// fmt.Printf("fileOrPath:%v\n",fileOrPath)
		path = zstring.BeforeRightNSep(fileOrPath, "/", 1)
		// fmt.Printf("55path:%v\n", path)
	}
	// fmt.Printf("path:%v\n",path)
	// ioutil.ReadDir()
	// ff := path + relativePath
	// fmt.Printf("ff:%v\n", ff)
	fp = filepath.Clean(path + relativePath)
	// fmt.Printf("ff:%v\n", ff)
	return
}

// 	file, _ :=zfiles.FromCallMethodRelativePath("iris.csv",1)
// 	file, _ :=zfiles.FromCallMethodRelativePath("iris.csv")
func FromCallMethodRelativePath(fileName string, skipOne ...int) (path string, err error) {
	var filename string
	if skipOne == nil {
		_, filename, _, _ = runtime.Caller(1)
	} else {
		_, filename, _, _ = runtime.Caller(skipOne[0])
	}
	path, err = FromRelativePath(filename, fileName)
	return
}
func substr(s string, pos, length int) string {
	runes := []rune(s)
	l := pos + length
	if l > len(runes) {
		l = len(runes)
	}
	return string(runes[pos:l])
}

func getParentDirectory(dirctory string) string {
	return substr(dirctory, 0, strings.LastIndex(dirctory, "/"))
}

func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		// log.Fatal(err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}

// dataFile, _ := filepath.Rel(filename, "../data/iris.csv")

// func GetAllFile(pathname string) error {
// 	rd, err := ioutil.ReadDir(pathname)
// 	for _, fi := range rd {
// 		if fi.IsDir() {
// 			fmt.Printf("[%s]\n", pathname+"\\"+fi.Name())
// 			GetAllFile(pathname + fi.Name() + "\\")
// 		} else {
// 			fmt.Println(fi.Name())
// 		}
// 	}
// 	return err
// }

// //获取指定目录下的所有文件和目录
// func GetFilesAndDirs(dirPth string) (files []string, dirs []string, err error) {
// 	dir, err := ioutil.ReadDir(dirPth)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	PthSep := string(os.PathSeparator)
// 	//suffix = strings.ToUpper(suffix) //忽略后缀匹配的大小写

// 	for _, fi := range dir {
// 		if fi.IsDir() { // 目录, 递归遍历
// 			dirs = append(dirs, dirPth+PthSep+fi.Name())
// 			GetFilesAndDirs(dirPth + PthSep + fi.Name())
// 		} else {
// 			// 过滤指定格式
// 			ok := strings.HasSuffix(fi.Name(), ".go")
// 			if ok {
// 				files = append(files, dirPth+PthSep+fi.Name())
// 			}
// 		}
// 	}

// 	return files, dirs, nil
// }

// //获取指定目录下的所有文件,包含子目录下的文件
// func GetAllFiles(dirPth string) (files []string, err error) {
// 	var dirs []string
// 	dir, err := ioutil.ReadDir(dirPth)
// 	if err != nil {
// 		return nil, err
// 	}

// 	PthSep := string(os.PathSeparator)
// 	//suffix = strings.ToUpper(suffix) //忽略后缀匹配的大小写

// 	for _, fi := range dir {
// 		if fi.IsDir() { // 目录, 递归遍历
// 			dirs = append(dirs, dirPth+PthSep+fi.Name())
// 			GetAllFiles(dirPth + PthSep + fi.Name())
// 		} else {
// 			// 过滤指定格式
// 			ok := strings.HasSuffix(fi.Name(), ".go")
// 			if ok {
// 				files = append(files, dirPth+PthSep+fi.Name())
// 			}
// 		}
// 	}

// 	// 读取子目录下文件
// 	for _, table := range dirs {
// 		temp, _ := GetAllFiles(table)
// 		for _, temp1 := range temp {
// 			files = append(files, temp1)
// 		}
// 	}

// 	return files, nil
// }

// func main() {
//     files, dirs, _ := GetFilesAndDirs("./simplemath")

//     for _, dir := range dirs {
//         fmt.Printf("获取的文件夹为[%s]\n", dir)
//     }

//     for _, table := range dirs {
//         temp, _, _ := GetFilesAndDirs(table)
//         for _, temp1 := range temp {
//             files = append(files, temp1)
//         }
//     }

//     for _, table1 := range files {
//         fmt.Printf("获取的文件为[%s]\n", table1)
//     }

//     fmt.Printf("=======================================\n")
//     xfiles, _ := GetAllFiles("./simplemath")
//     for _, file := range xfiles {
//         fmt.Printf("获取的文件为[%s]\n", file)
//     }
// }
