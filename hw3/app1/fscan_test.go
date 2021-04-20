package main_test

import (
	"fmt"
	_ "fmt"
	"github.com/pehks1980/gb_go_best/hw3/app1/fscan"
	logger "github.com/pehks1980/gb_go_best/hw3/app1/logger"
	"github.com/stretchr/testify/assert"
	_ "github.com/stretchr/testify/assert"
	"os"
	"testing"
)

//////////////////////////// TestIOReadDir test func IOReadDir for error
func TestIOReadDir(t *testing.T) {
	Logger := logger.InitLoggers("log_test.txt", true, 3)
	fileSet := fscan.NewRWSet(false, Logger, false)

	_, err := fileSet.IOReadDir("/home")
	if assert.Error(t, err) {
		serr := fmt.Sprintf("err: %v",err)
		assert.Contains(t, serr, "not permitted")
	}

	_ = os.Remove("log_test.txt")

}

///////////////////////////// TestScanDir test main func scandir run see log
func TestScanDir(t *testing.T) {
	Logger := logger.InitLoggers("log_run_test.txt", true, 3)
	fileSet := fscan.NewRWSet(false, Logger, false)

	fileSet.WaitGroup.Add(1)
	go fileSet.ScanDir("/Users/user", "nil")
	//ждем пока все перемножится
	fileSet.WaitGroup.Wait()

	//_ = os.Remove("log_run_test.txt")
}

///////////////////////////// TestMockScanDir test mocks FS see log
func TestMockScanDir(t *testing.T) {
	Logger := logger.InitLoggers("log_mock_test.txt", true, 3)
	fileSet := fscan.NewRWSet(false, Logger, true)

	fileSet.WaitGroup.Add(1)
	go fileSet.ScanDir("/", "nil")
	//ждем пока все перемножится
	fileSet.WaitGroup.Wait()

	//_ = os.Remove("log_mock_test.txt")
}

///////////////////////////// test fileSet adding editing having
func TestHashTable(t *testing.T) {
	Logger := logger.InitLoggers("log_hash_test.txt", true, 3)
	fileSet := fscan.NewRWSet(false, Logger, false)
	// add one member

	elemMM := &fscan.FileElem{
		FullPath: "fullFilePath",
		Filesize: 123,
		FileHash: "hash",
		DubPaths: nil,
	}
	fileSet.Add("first", *elemMM)
	//check if we have it inside map
	want := true
	if got := fileSet.Has("first"); got != want {
		t.Errorf("fileSet Has = %t, want %t", got, want)
	}
	want = true
	if got := fileSet.Edit("first", "file"); got != want {
		t.Errorf("fileSet Has = %t, want %t", got, want)
	}
	want = true
	if got := fileSet.Edit("first", "file"); got != want {
		t.Errorf("fileSet Has = %t, want %t", got, want)
	}
	want = false
	if got := fileSet.Edit("second", "file"); got != want {
		t.Errorf("fileSet Has = %t, want %t", got, want)
	}

	_ = os.Remove("log_hash_test.txt")
}

/////////////////// GetFileMd5Hash - not existing file
func TestGetFileMd5Hash(t *testing.T) {

	Logger := logger.InitLoggers("log_md5_test.txt", true, 3)
	fileSet := fscan.NewRWSet(false, Logger, false)

	_, err := fileSet.GetFileMd5Hash("aaa")

	if assert.Error(t, err) {
		serr := fmt.Sprintf("err: %v",err)
		assert.Contains(t, serr, "no such file or directory")
	}
	_ = os.Remove("log_md5_test.txt")
}
