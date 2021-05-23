package mockfs

import (
	"io/fs"
	"os"
)

// попытка сделать мокинг файловой системы
// интерфейс для подмены
type DirReader interface {
	// читаем выдаем слайс с файлом
	Readdir() ([]os.DirEntry, error)
	New(path string, pygench <-chan int) DirReader
}

// cтруктура для стандартной работы
type Dir struct {
	Path  string
	Files []os.DirEntry
	// SubDirs map[string]*Dir
}

// структура для мокинговой системы
type MockDir struct {
	Path    string
	Files   []MockDirEntry
	PyGenCh <-chan int // указатель на канал из main переходит в обьект s а потом сюда

	// SubDirs map[string]*MockDir
}

// cтруктура для мокинга элементов фс
type MockDirEntry struct {
	FileName    string
	IsDirectory bool
}

// методы для подмены элементов фс интерфейса fs.DirEntry
func (mfi MockDirEntry) IsDir() bool {
	return mfi.IsDirectory
}

func (mfi MockDirEntry) Name() string {
	return mfi.FileName
}

func (mfi MockDirEntry) Type() fs.FileMode {
	return fs.ModePerm
}

func (mfi MockDirEntry) Info() (fs.FileInfo, error) {
	return nil, nil
}

// конструктор элемента фс имеет имя и флаг - папка / файлик
func NewMockDirEntry(name string, isDir bool) MockDirEntry {
	return MockDirEntry{
		FileName:    name,
		IsDirectory: isDir,
	}
}

func (md MockDir) New(path string, pygench <-chan int) DirReader {
	return MockDir{Path: path, PyGenCh: pygench} // NewMockDir(path)
}

func NewMockDir(path string) MockDir {
	return MockDir{Path: path}
}

// gen function like in python - every time gives one number to channel
func PyGen() <-chan int {
	chnl := make(chan int)
	go func() {
		genarr := []int{1, 1, 2, 2, 3, 3, 3, 3}
		for _, i := range genarr {
			// отдает значение и ждет как демон, следующего раза
			chnl <- i
		}
		// по завершении уст флаг закрытия канала
		close(chnl)
	}()
	return chnl
}

// мокинг метод для чтения папки фс
func (md MockDir) Readdir() ([]os.DirEntry, error) {
	// берет набор []os.DirEntry
	// и заносит с него в список файлов в зависимости от случая

	files := make([]os.DirEntry, 0, 10)
	// get numbers from generator PyGen via PyGenCh to make mocking catalog according to numbers
	if number, ok := <-md.PyGenCh; ok {
		if number == 1 {
			file := NewMockDirEntry("Dir1", true)
			files = append(files, file)
			file = NewMockDirEntry("Dir2", true  )
			files = append(files, file)
		}
		if number == 2 {
			file := NewMockDirEntry("Dir1", true)
			files = append(files, file)
			file = NewMockDirEntry("file2.txt", false  )
			files = append(files, file)
		}
		if number == 3 {
			file := NewMockDirEntry("file1.txt", false)
			files = append(files, file)
			file = NewMockDirEntry("file2.txt", false)
			files = append(files, file)
		}
	}

	return files, nil
}

// конструкор если хочется создавать структуру Dir так
func NewDir(path string) Dir {
	return Dir{Path: path}
}

func (fd Dir) New(path string, pygench <-chan int) DirReader {
	return Dir{Path: path} // NewDir(path)
}

// читалка фс стандартным способом
func (fd Dir) Readdir() ([]os.DirEntry, error) {
	fileInfos, err := os.ReadDir(fd.Path)
	if err != nil {
		return nil, err
	}
	return fileInfos, nil
}
