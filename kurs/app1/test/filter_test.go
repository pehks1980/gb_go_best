package main

import (
	"github.com/pehks1980/gb_go_best/kurs/app1/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// minimock -i DB укзывает что надо мокить (

//go install github.com/gojuno/minimock/v3/cmd/minimock # устанавали
//minimock -i DB # запускаем генерацию заглушки для интерфейса DB


func TestFilter_New(t *testing.T)  {
	// инициализция мокинг структур
	dbMock := &ParseMock{}
	//инициализация калькулятора с подстановкой мокинга в db
	//calc := &Calculator{db: dbMock}
	// настройка поведения функций черный ящик вход <-> результ
	dbMock.GetUserNameMock.Expect(42).Return("Bob", nil)
	dbMock.GetOrderItemsMock.Expect(100500).Return([]uint64{100, 200, 250}, nil)

	//собственно пробуем тестить функцию калькулятера (в которой эти два метода дают вот эти данные)
	filter.NewFilter()
	res, err := calc.ProcessOrder(42, 100500)
	require.NoError(t, err)
	assert.Equal(t, "user Bob spent $550", res)
}