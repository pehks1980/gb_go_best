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
	"github.com/pehks1980/gb_go_best/hw2/app1/logger"
	"hash/crc32"
	"io"
	"os"
	"sync"
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
	// mm элемент хеш таблицы хранит ключ (хеш)
	// и значение структуру FileElem
	MM map[string]FileElem
	// FilesHaveDubs счетчик файлов которые имеют дубли
	FilesHaveDubs int64
}

// NewRWSet - конструктор Хештаблицы FileElem
func NewRWSet() *RWSet {
	return &RWSet{
		MM:            map[string]FileElem{},
		FilesHaveDubs: 0,
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
func GetHash(fileSz int64, fileName string, fileHash string) (string, error) {
	hashFileNameSize := crc32.NewIEEE()
	strFileSize := fmt.Sprintf("%d", fileSz)
	_, _ = hashFileNameSize.Write([]byte(fileName + strFileSize + fileHash))
	strHash := fmt.Sprintf("%d", hashFileNameSize.Sum32())
	//Logger.ErrorFileLogger.Println("hello from Gethash")
	return strHash, nil
}

// GetFileMd5Hash - вычисления md5 хеш файла для -ds
// filePath - имя файла для вычисления его md5
func GetFileMd5Hash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		logger.Logger.Errorf("Error while calculate md5 hash %v\n", err)
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// DeleteDup физическое удаление дупликата
// dub - полное имя файла под удаление из фс
func DeleteDup(dub string) error {
	err := os.Remove("sdfg")
	if err != nil {
		logger.Logger.Errorf("Error deleting file: %s : %v", dub, err)
		return err
	}
	return nil
}

// IOReadDir - сканирование папки и поиск дублей файлов
// root  - каталог где искать
// fileSet - указатель на хеш таблицу найденных файлов
// deepScan - ключ программы делать и учитывать md5 файлов
func IOReadDir(root string, fileSet *RWSet, deepScan *bool) ([]string, error) {
	var files []string

	fileDir, err := os.ReadDir(root)
	if err != nil {
		logger.Logger.Infof("Problems with Reading dir: %s %v", root, err)
		return files, err
		//log.Fatal(err)
	}

	/*fileInfo, err := ioutil.ReadDir(root)
	if err != nil {
		return files, err
	}*/

	for _, file := range fileDir {
		if file.IsDir() {
			files = append(files, file.Name())
			logger.Logger.Infof("Reading dir: %s", file.Name())
			//fmt.Printf("dir=%s\n", file.Name())
		} else {

			fullFilePath := fmt.Sprintf("%s/%s", root, file.Name())

			var NameHash string

			statFile, err := os.Stat(fullFilePath)
			if err != nil {
				logger.Logger.Infof("Problem with reading file: %s %v", fullFilePath, err)
				continue
			}
			if statFile.IsDir() {
				continue
			}

			if *deepScan {
				fileMd5Hash, _ := GetFileMd5Hash(fullFilePath)
				if err != nil {
					logger.Logger.Infof("Problem with md5: hash %s %v", fullFilePath, err)
					continue
				}
				NameHash, _ = GetHash(statFile.Size(), file.Name(), fileMd5Hash)
			} else {
				NameHash, _ = GetHash(statFile.Size(), file.Name(), "")
			}

			if fileSet.Has(NameHash) {
				// dublicat
				// update struct
				fileSet.Edit(NameHash, fullFilePath)
			} else {
				// new element
				var elemMM *FileElem
				if *deepScan {
					fileMd5Hash, _ := GetFileMd5Hash(fullFilePath)
					elemMM = &FileElem{
						FullPath: fullFilePath,
						Filesize: statFile.Size(),
						FileHash: fileMd5Hash,
						DubPaths: nil,
					}
				} else {
					elemMM = &FileElem{
						FullPath: fullFilePath,
						Filesize: statFile.Size(),
						FileHash: "",
						DubPaths: nil,
					}
				}
				NameHash, _ := GetHash(statFile.Size(), file.Name(), elemMM.FileHash)
				fileSet.Add(NameHash, *elemMM)
			}

		}
	}
	return files, nil
}
