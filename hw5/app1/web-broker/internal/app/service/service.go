package service

import (
	"github.com/vlslav/web-broker/internal/pkg/model"
)
// repo имеет тип интерфейс (2 метода)
type repo interface {
	Get(key string) (string, error)
	Put(putReq *model.PutValue) error
}
// service имеет тип структура
// содержит член repo
type Service struct {
	repo repo
}
// конструктор Service
// возвращает указатель на структуру с интерфейсом
// что в repo подставим та структура и будет - главное методы должны иметь одинаковую сигнатуру и имя!
func New(repo repo) *Service {
	return &Service{repo: repo}
}
