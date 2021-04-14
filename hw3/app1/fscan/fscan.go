// Package fscan implements functions and structs to find files which have duplicates
//
//
//
// ТЗ:
//В качестве завершающего задания нужно выполнить программу поиска дубликатов файлов.
//Дубликаты файлов - это файлы, которые совпадают по имени файла и по его размеру.
//Нужно написать консольную программу, которая проверяет наличие дублирующихся файлов.
//Программа должна работать на локальном компьютере и получать на вход путь до директории.
//Программа должна вывести в стандартный поток вывода список дублирующихся файлов,
//которые находятся как в директории, так и в поддиректориях директории,
//переданной через аргумент командной строки.
//Данная функция должна работать эффективно при помощи распараллеливания программы
//
//Программа должна принимать дополнительный ключ - возможность удаления обнаруженных дубликатов файлов после поиска.
//Дополнительно нужно придумать, как обезопасить пользователей от случайного удаления файлов.
//В качестве ключей желательно придерживаться общепринятых практик по использованию командных опций.
//Критерии приемки программы:
//
//k  1. Программа компилируется.
//
//k  2. Программа выполняет функциональность, описанную выше.
//
//k  3. Программа покрыта тестами.
//
//k  4. Программа содержит документацию и примеры использования.
//
//k  5. Программа обладает флагом “-h/--help” для краткого объяснения функциональности.
//
//k  6. Программа должна уведомлять пользователя об ошибках, возникающих во время выполнения.
//
//Дополнительно можете выполнить следующие задания:
//
//1. Написать программу, которая по случайному принципу генерирует копии уже имеющихся файлов, относительно указанной директории.
//
//2. Сравнить производительность программы в однопоточном и многопоточном режимах.
//
// Некоторые методы и структуры программы:
//
package fscan

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/sirupsen/logrus"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"sync"
	"sync/atomic"
)

var (
	// init generator channel as global var (used in mocking fs)
	PyGenCh = PyGen()
)

// FileElem is структура найденного файла.
type FileElem struct {
	// FullPath полное имя файла (каталог + имя в каталоге).
	FullPath string
	// Filesize размер файла в байтах.
	Filesize int64
	// DubPaths слайс строк полных имен дубликатов файлов.
	DubPaths []string
	// FileHash мд5 хеш файла используется с ключем -ds.
	FileHash string
}

// RWSet структура хеш таблицы найденных файлов
type RWSet struct {
	sync.RWMutex
	sync.WaitGroup
	// mm элемент хеш таблицы хранит ключ (хеш)
	// и значение структуру FileElem
	MM map[string]FileElem
	// FilesHaveDubs счетчик файлов которые имеют дубли
	FilesHaveDubs int64
	// ProcCounter счик процедур запущенных через оп go
	ProcCounter int64
	// flag -  учитывать содержимое файла (считать md5)
	DeepScan bool
	Logger   *logrus.Logger
	MockFs   bool
}

// NewRWSet - конструктор Хештаблицы FileElem
func NewRWSet(ds bool, logging *logrus.Logger, mockfs bool) *RWSet {
	return &RWSet{
		MM:            map[string]FileElem{},
		FilesHaveDubs: 0,
		ProcCounter:   1,
		DeepScan:      ds,
		Logger:        logging,
		MockFs: 	   mockfs,
	}
}

// Add - добавляет в хеш таблицу новый елемент FileElem
func (s *RWSet) Add(nameMM string, ElemMM FileElem) {
	s.Lock()
	s.MM[nameMM] = ElemMM
	s.Unlock()
}

// Edit - вводит новый дубликат в соответсвтующий елемент FileElem
func (s *RWSet) Edit(nameMM string, dubPath string) bool {
	s.Lock()
	defer s.Unlock()
	if elemMM, ok := s.MM[nameMM]; ok {
		elemMM.DubPaths = append(elemMM.DubPaths, dubPath)
		s.MM[nameMM] = elemMM
		s.FilesHaveDubs++
		return true
	}
	return false
}

// Has - проверяет есть ли уже элемент в хеш таблице по такому ключу хешу
func (s *RWSet) Has(nameMM string) bool {
	s.RLock()
	defer s.RUnlock()
	_, ok := s.MM[nameMM]
	return ok
}

// GetHash вычисление хеш сrc32
// по значениям размера имени и хеша md5 в случае -ds
func (s *RWSet) GetHash(fileSz int64, fileName string, fileHash string) (string, error) {
	hashFileNameSize := crc32.NewIEEE()
	strFileSize := fmt.Sprintf("%d", fileSz)
	_, _ = hashFileNameSize.Write([]byte(fileName + strFileSize + fileHash))
	strHash := fmt.Sprintf("%d", hashFileNameSize.Sum32())
	//Logger.ErrorFileLogger.Println("hello from Gethash")
	return strHash, nil
}

// GetFileMd5Hash - вычисления md5 хеш файла для -ds
// filePath - имя файла для вычисления его md5
func (s *RWSet) GetFileMd5Hash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		s.Logger.Errorf("Error while calculate md5 hash %v\n", err)
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// DeleteDup физическое удаление дупликата
// dub - полное имя файла под удаление из фс
func (s *RWSet) DeleteDup(dub string) error {
	err := os.Remove("sdfg")
	if err != nil {
		s.Logger.Errorf("Error deleting file: %s : %v", dub, err)
		return err
	}
	return nil
}

// попытка сделать мокинг файловой системы
// интерфейс для подмены
type DirReader interface {
	// читаем выдаем слайс с файлом
	Readdir() ([]os.DirEntry, error)
}
// cтруктура для стандартной работы
type Dir struct {
	Path  string
	Files []os.DirEntry
	//SubDirs map[string]*Dir
}
// структура для мокинговой системы
type MockDir struct {
	Path  string
	Files []MockDirEntry
	//SubDirs map[string]*MockDir
}
// cтруктура для мокинга элементов фс
type MockDirEntry struct {
	FileName    string
	IsDirectory bool
}
//методы для подмены элементов фс интерфейса fs.DirEntry
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

// gen function like in python - every time gives one number to channel
func PyGen() <-chan int {
	chnl := make(chan int)
	go func() {
		gen_arr := []int{1, 1, 2, 2, 3, 3, 3, 3}
		for _, i := range(gen_arr) {
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
	if number, ok := <- PyGenCh; ok{
		if number == 1 {
			file := NewMockDirEntry("Dir1", true)
			files = append(files, file)
			file = NewMockDirEntry("Dir2", true)
			files = append(files, file)
		}
		if number == 2 {
			file := NewMockDirEntry("Dir1", true)
			files = append(files, file)
			file = NewMockDirEntry("file2.txt", false)
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

// читалка фс стандартным способом
func (fd Dir) Readdir() ([]os.DirEntry, error) {
	fileInfos, err := os.ReadDir(fd.Path)
	if err != nil {
		return nil, err
	}
	return fileInfos, nil
}

// IOReadDir - сканирование папки и поиск дублей файлов
// root  - каталог где искать
// fileSet - указатель на хеш таблицу найденных файлов
// deepScan - ключ программы делать и учитывать md5 файлов
func (s *RWSet) IOReadDir(root string) ([]string, error) {
	var files []string

	//dir := NewDir(root)
	//fileDir, err := dir.Readdir()

	// пустой указатель интерфейса
	var dirIf DirReader
	// инициализируем структуру и приравниваем указатель интерфейсу (по флагу MockFs)
	if s.MockFs{
		dirIf = MockDir{Path: root}
	}else{
		dirIf = Dir{Path: root}
	}

	// делаем чтение методом через указатель интерфейса
	fileDir, err := dirIf.Readdir()

	if err != nil {
		s.Logger.Infof("Problems with Reading dir: %s %v", root, err)
		return files, err
		//log.Fatal(err)
	}

	for _, file := range fileDir {
		if file.IsDir() {
			files = append(files, file.Name())
			s.Logger.Infof("Reading dir: %s", file.Name())
			//fmt.Printf("dir=%s\n", file.Name())
		} else {

			fullFilePath := fmt.Sprintf("%s/%s", root, file.Name())

			var NameHash string

			statFile, err := os.Stat(fullFilePath)
			//we need only File Size from os.Stat
			var statFileSize int64

			if err != nil {
				s.Logger.Infof("Problem with reading file: %s %v", fullFilePath, err)
				//continue //we go on not skip for the sake of mocking fs
			}

			// if mocking fs - we setup fixed size of file
			if s.MockFs {
				statFileSize = 1024
			} else {
				statFileSize = statFile.Size()
			}

			//additional check it s not a dir
			/*
			if statFile.IsDir() {
				continue
			}
			 */

			if s.DeepScan {
				fileMd5Hash, err := s.GetFileMd5Hash(fullFilePath)
				if err != nil {
					s.Logger.Infof("Problem with md5: hash %s %v", fullFilePath, err)
					continue
				}
				NameHash, _ = s.GetHash(statFileSize, file.Name(), fileMd5Hash)
			} else {
				NameHash, _ = s.GetHash(statFileSize, file.Name(), "")
			}

			if s.Has(NameHash) {
				// dublicat
				// update struct
				s.Edit(NameHash, fullFilePath)
			} else {
				// new element
				var elemMM *FileElem
				if s.DeepScan {
					fileMd5Hash, _ := s.GetFileMd5Hash(fullFilePath)
					elemMM = &FileElem{
						FullPath: fullFilePath,
						Filesize: statFileSize,
						FileHash: fileMd5Hash,
						DubPaths: nil,
					}
				} else {
					elemMM = &FileElem{
						FullPath: fullFilePath,
						Filesize: statFileSize,
						FileHash: "",
						DubPaths: nil,
					}
				}
				NameHash, _ := s.GetHash(statFileSize, file.Name(), elemMM.FileHash)
				s.Add(NameHash, *elemMM)
			}

		}
	}
	return files, nil
}

// ScanDir - принимает начальную папку и сканирует все подпапки
// для каждой подпапки запускает саму себя, выделяя новый поточек
func (s *RWSet) ScanDir(pathDir string, rootDir string) {
	defer s.Done()
	defer func() {
		err := recover()
		if err != nil {
			entry := err.(*logrus.Entry)
			s.Logger.WithFields(logrus.Fields{
				"dir_root":    rootDir, // рут папка
				"dir_err":     pathDir,
				"err_level":   entry.Level,
				"err_message": entry.Message,
			}).Error("Ошибка!!! Доступ к папке!!!")
		}
	}()

	dirs, err := s.IOReadDir(pathDir)
	if err != nil {
		s.Logger.Panicf("Error reading dirs: %v", err)
		//logger.Logger.Errorf("Error reading dirs: %v", err)
		return
	}

	for _, dir := range dirs {
		s.WaitGroup.Add(1)
		atomic.AddInt64(&s.ProcCounter, 1)
		sDir := pathDir + "/" + dir
		go s.ScanDir(sDir, pathDir)
	}

}
